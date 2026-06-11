# Qmby License Server

独立的 Qmby 激活码 / 会员授权服务。

## 功能

- 公开购买页：游客填写邮箱并通过支付宝付款购买激活码
- 管理员生成激活码并绑定邮箱
- 会员等级：`trial` 7 天试用、`yearly` 年费、`permanent` 永久、`beta` 内测
- Qmby 联网后通过激活码校验会员状态
- 首次校验时开始计算有效期
- SQLite 持久化，适合单机 Docker 部署
- 激活码只保存 SHA-256 哈希，生成后明文只在管理页显示一次
- 记录最近联网时间、客户端北京时间、实例 ID、Qmby 版本、IP、校验次数
- 统计累计联网安装数、最近 10 分钟活跃客户端数、24 小时校验次数
- 独立 IP 归属地上报与查询接口

## 运行

```bash
docker compose up -d --build
```

管理页：

```text
http://localhost:2090/admin
```

购买页：

```text
http://localhost:2090/buy
```

默认账号密码在 `docker-compose.yml` 中配置：

```yaml
ADMIN_USER=admin
ADMIN_PASSWORD=admin123
LICENSE_ED25519_PRIVATE_KEY=base64-ed25519-private-key
```

请部署前改掉 `ADMIN_PASSWORD`。
`LICENSE_ED25519_PRIVATE_KEY` 必须从服务端环境变量或密钥管理注入，值为 base64 编码的 Ed25519 64 字节私钥或 32 字节 seed；客户端构建时需要内置对应的 base64 Ed25519 公钥。

## 支付宝与邮件

购买页使用支付宝电脑网站支付。正式使用前可以在管理页 `/admin` 的“支付与邮件配置”里填写，也可以用环境变量配置：

```yaml
PUBLIC_BASE_URL=https://license.example.com
ALIPAY_GATEWAY=https://openapi.alipay.com/gateway.do
ALIPAY_APP_ID=你的应用 AppID
ALIPAY_PRIVATE_KEY=应用私钥
ALIPAY_PUBLIC_KEY=支付宝公钥
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=mailer@example.com
SMTP_PASSWORD=邮箱 SMTP 密码
SMTP_FROM=Qmby License <mailer@example.com>
```

付款成功后，支付宝会请求：

```text
/api/pay/alipay/notify
```

服务端验签通过后会生成激活码、绑定购买邮箱，并发送到该邮箱。管理员在 `/admin/codes` 手动生成激活码时，如果已配置 SMTP，也会自动发送激活码邮件。

## Qmby 校验接口

```http
POST /api/license/verify
Content-Type: application/json
```

请求：

```json
{
  "activation_code": "QMBY-ABCD-EFGH-IJKL-MNOP-QRST",
  "beijing_time": "2026-05-21 20:00:00",
  "instance_id": "qmby-instance-id",
  "qmby_version": "0.0.15"
}
```

这个接口只负责会员校验和在线心跳，不接收归属地字段。云端通过 `activation_code` 查到绑定邮箱，首次成功校验时绑定当前 `instance_id`；后续不同实例使用同一激活码会返回非会员状态。IP 归属地请使用下面的 `/api/ip/report` 单独上报。

响应：

```json
{
  "license": {
    "email": "user@example.com",
    "instance_id": "qmby-instance-id",
    "member": true,
    "level": "trial",
    "level_label": "试用会员",
    "status": "ok",
    "starts_at": "2026-05-21T20:00:00+08:00",
    "expires_at": "2026-05-28T20:00:00+08:00",
    "server_beijing_time": "2026-05-21T20:00:00+08:00",
    "features": {
      "upload_monitor": {
        "label": "文件监控",
        "access": "member",
        "enabled": true
      }
    }
  },
  "signature": "base64_ed25519_signature"
}
```

服务端会先稳定序列化 `license` 对象，再用 Ed25519 私钥签名这份 JSON 字节；响应中不会再返回顶层裸 `member` / `features` 作为可信授权依据。客户端需要用内置公钥验签，并检查云端返回的 `email`、`instance_id`、`starts_at`、`expires_at`。

如果设置了 `LICENSE_API_KEY`，Qmby 请求需要带：

```http
Authorization: Bearer your-secret-key
```

或者：

```http
X-License-Key: your-secret-key
```

`POST /api/ip/report` 也使用同一个密钥，并支持 `X-API-Key` 请求头。

## 客户端统计与 IP 归属地

管理页 `/admin` 会显示：

- 累计联网安装：按 `instance_id` 去重；没有 `instance_id` 时按邮箱和 IP 兜底
- 正在使用：最近 10 分钟内有校验心跳的客户端
- 活跃会员客户端：最近 10 分钟内仍是会员状态的客户端
- 24 小时校验次数
- IP 归属地记录数

IP 归属地上报：

```http
POST /api/ip/report
Content-Type: application/json
X-API-Key: your-secret-key
```

```json
{
  "email": "user@example.com",
  "beijing_time": "2026-05-21 20:00:00",
  "instance_id": "qmby-instance-id",
  "qmby_version": "0.0.15",
  "ip": "1.2.3.4",
  "location": "中国湖北省武汉市",
  "district": "",
  "street": "",
  "isp": "联通",
  "latitude": 30.5928,
  "longitude": 114.3055,
  "provider": "Qmby"
}
```

`/api/ip/report` 主字段与校验接口保持一致：`email`、`beijing_time`、`instance_id`、`qmby_version`。

IP 归属地查询：

```http
GET /api/ip/lookup?ip=1.2.3.4
```

响应：

```json
{
  "found": true,
  "ip": "1.2.3.4",
  "location": "中国湖北省武汉市",
  "district": "",
  "street": "",
  "isp": "联通",
  "latitude": 30.5928,
  "longitude": 114.3055,
  "provider": "Qmby",
  "count": 1,
  "updated_at": "2026-05-25T12:00:00+08:00"
}
```

## Qshare 共享中心接口

Qshare 使用与 license verify 相同的 `LICENSE_API_KEY` 鉴权头，并继续用 `email + instance_id` 标识 Qmby 实例。云端只保存展示元数据和秒传文件元数据，不保存 115 账号凭据，也不保存 115 分享链接。

所有 Qshare 接口都使用 `POST + JSON`：

```text
POST /api/qshare/status
POST /api/qshare/resources/publish
POST /api/qshare/resources/list
POST /api/qshare/resources/detail
POST /api/qshare/resources/delete
```

基础身份字段：

```json
{
  "email": "user@example.com",
  "instance_id": "qmby-instance-id",
  "beijing_time": "2026-06-09 17:30:00",
  "publish_folder_configured": true
}
```

发布 / 重复发布更新：

```json
{
  "email": "user@example.com",
  "instance_id": "qmby-instance-id",
  "beijing_time": "2026-06-09 17:30:00",
  "publish_folder_configured": true,
  "resource": {
    "id": 1,
    "media_type": "movie",
    "tmdb_id": 157336,
    "title": "Interstellar",
    "year": 2014,
    "poster_url": "https://image.tmdb.org/t/p/w500/poster.jpg",
    "file_count": 1,
    "total_size": 123456789,
    "files": [
      {
        "id": "DIR:.",
        "name": "Interstellar",
        "relative_path": ".",
        "is_dir": true,
        "size": 123456789,
        "sha1": "",
        "publisher_115_id": "4577361",
        "chat_mid": "mid-1",
        "chat_contact_id": "1182480"
      }
    ]
  }
}
```

删除 / 详情请求在基础身份字段外增加 `resource_id`。浏览共享中心不再要求当前用户已发布资源；只要 `publish_folder_configured=true` 且 license 仍有效，就允许浏览。`publish_folder_configured=false` 时，状态接口返回 `can_browse=false`，列表返回 `can_browse=false` 和空 `resources`，详情接口返回明确错误。

`files` 可以是文件或文件夹条目。文件夹条目使用 `is_dir=true`，允许 `sha1` 为空；转发时使用该条目的聊天消息 `chat_mid`。

重复发布按同一发布者 `email + instance_id` 内去重：优先使用 `media_type + tmdb_id`；如果 `tmdb_id` 缺失，则使用 `media_type + title`。对外展示只返回匿名 `owner_label`，不会返回发布者 email、instance_id 或本地发布路径。

## SQLite 是否够用

够用，前提是这个授权服务是单容器 / 单实例部署。当前配置启用了 WAL、`busy_timeout`，并限制单连接写入，适合管理员低频生成激活码、Qmby 实例低到中等频率校验。

如果以后变成多实例授权服务、大量用户高并发、需要复杂报表或支付系统，再迁到 PostgreSQL 会更稳。
