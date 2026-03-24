# GeoCache

EmbyQ 归属地共享中心服务

## 启动

```bash
docker compose up -d --build
```

## 健康检查

- http://127.0.0.1:18080/healthz

## 接口

- POST /v1/ip/report  (Header: X-API-Key)
- GET  /v1/ip/lookup?ip=1.2.3.4

## 默认 API Key

`change_me_api_key` (请在 docker-compose.yml 中修改)