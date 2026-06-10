package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdmin(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, adminHTML)
}

const adminHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Qmby License Admin</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #f6f8fb;
      --panel: rgba(255,255,255,.82);
      --text: #172033;
      --muted: #617086;
      --line: #dfe6ef;
      --primary: #0ea5e9;
      --teal: #0fbaa7;
      --danger: #ef4444;
      --success: #4fbd31;
      --warn: #f59e0b;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #101722;
        --panel: rgba(22,31,45,.82);
        --text: #eef4fb;
        --muted: #9caabc;
        --line: #263244;
      }
	    }
	    * { box-sizing: border-box; }
	    body {
	      margin: 0;
      font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: radial-gradient(circle at 12% 18%, rgba(14,165,233,.20), transparent 28rem),
                  radial-gradient(circle at 86% 26%, rgba(15,186,167,.18), transparent 28rem),
                  var(--bg);
	      color: var(--text);
	      min-height: 100vh;
	    }
	    .login-screen {
	      min-height: 100vh;
	      display: grid;
	      place-items: center;
	      padding: 24px;
	    }
	    .login-card {
	      width: min(420px, 100%);
	      background: var(--panel);
	      border: 1px solid rgba(148,163,184,.24);
	      border-radius: 10px;
	      box-shadow: 0 24px 80px rgba(15,23,42,.14);
	      backdrop-filter: blur(16px);
	      padding: 28px;
	    }
	    .brand {
	      display: flex;
	      align-items: center;
	      gap: 12px;
	      margin-bottom: 24px;
	    }
	    .brand-mark {
	      width: 42px;
	      height: 42px;
	      border-radius: 10px;
	      display: grid;
	      place-items: center;
	      color: white;
	      font-weight: 800;
	      background: linear-gradient(135deg, var(--primary), var(--teal));
	      box-shadow: 0 12px 28px rgba(14,165,233,.25);
	    }
	    .brand-title { font-size: 19px; font-weight: 750; line-height: 1.2; }
	    .brand-subtitle { color: var(--muted); font-size: 13px; margin-top: 3px; }
	    .login-form { display: grid; gap: 14px; }
	    .login-form input { height: 44px; }
	    .login-form button { height: 44px; margin-top: 4px; }
	    .login-error { min-height: 20px; color: var(--danger); font-size: 13px; }
	    .hidden { display: none !important; }
	    .wrap { max-width: 1180px; margin: 0 auto; padding: 28px 18px 48px; }
	    header { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 22px; }
    h1 { font-size: 26px; margin: 0 0 6px; letter-spacing: 0; }
    p { color: var(--muted); margin: 0; }
    .panel {
      background: var(--panel);
      border: 1px solid rgba(148,163,184,.22);
      border-radius: 8px;
      box-shadow: 0 18px 60px rgba(15,23,42,.10);
      backdrop-filter: blur(14px);
      margin-bottom: 18px;
    }
    .panel h2 { font-size: 16px; margin: 0; }
    .panel-head { padding: 18px 18px 0; }
    .panel-body { padding: 18px; }
    .grid { display: grid; grid-template-columns: 1.3fr .75fr 1fr auto; gap: 12px; align-items: end; }
    label { display: block; font-size: 12px; color: var(--muted); margin-bottom: 6px; }
    input, select {
      width: 100%;
      height: 38px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: rgba(255,255,255,.65);
      color: var(--text);
      padding: 0 11px;
      font-size: 14px;
      outline: none;
    }
    @media (prefers-color-scheme: dark) {
      input, select { background: rgba(15,23,42,.58); }
    }
    button {
      height: 38px;
      border: 0;
      border-radius: 8px;
      background: var(--primary);
      color: white;
      padding: 0 14px;
      font-weight: 650;
      cursor: pointer;
	    }
	    button.secondary { background: rgba(100,116,139,.16); color: var(--text); border: 1px solid var(--line); }
	    button.danger { background: rgba(239,68,68,.12); color: var(--danger); border: 1px solid rgba(239,68,68,.25); }
	    button:disabled { opacity: .58; cursor: default; }
	    .header-actions { display: flex; gap: 10px; align-items: center; }
    .filters { display: grid; grid-template-columns: 1fr 150px 150px auto; gap: 12px; align-items: end; }
    .client-filters { display: grid; grid-template-columns: 1fr 130px 130px auto; gap: 12px; align-items: end; }
    .ip-bests-filters { display: grid; grid-template-columns: 1fr auto; gap: 12px; align-items: end; }
    .qshare-filters { display: grid; grid-template-columns: 1fr 150px auto; gap: 12px; align-items: end; }
    .stats-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 12px; }
    .features-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 12px; }
    .feature-card { border: 1px solid var(--line); border-radius: 8px; padding: 14px; background: rgba(255,255,255,.38); display: grid; gap: 10px; }
    .feature-title { font-weight: 700; }
    .stat { border: 1px solid var(--line); border-radius: 8px; padding: 14px; background: rgba(255,255,255,.38); }
    .stat-value { font-size: 25px; font-weight: 780; line-height: 1.1; }
    .stat-label { color: var(--muted); font-size: 12px; margin-top: 6px; }
    textarea {
      width: 100%;
      min-height: 92px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: rgba(255,255,255,.65);
      color: var(--text);
      padding: 10px 11px;
      font-size: 13px;
      outline: none;
      resize: vertical;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    }
    .generated {
      margin-top: 14px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
      padding: 13px;
      border: 1px solid rgba(79,189,49,.24);
      border-radius: 8px;
      background: rgba(79,189,49,.10);
    }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-weight: 700; overflow-wrap: anywhere; }
    .msg { margin-top: 12px; color: var(--muted); font-size: 14px; min-height: 20px; }
    table { width: 100%; border-collapse: collapse; min-width: 920px; font-size: 14px; }
    th, td { padding: 12px 14px; border-top: 1px solid var(--line); text-align: left; vertical-align: top; }
    th { color: var(--muted); font-size: 12px; font-weight: 650; background: rgba(100,116,139,.08); }
    .table-wrap { overflow-x: auto; border: 1px solid var(--line); border-radius: 8px; }
    .badge { display: inline-flex; padding: 4px 8px; border-radius: 7px; font-size: 12px; }
    .issued { background: rgba(14,165,233,.12); color: var(--primary); }
    .active { background: rgba(79,189,49,.12); color: var(--success); }
    .disabled { background: rgba(100,116,139,.16); color: var(--muted); }
    .expired { background: rgba(245,158,11,.13); color: var(--warn); }
	    .note { color: var(--muted); font-size: 12px; margin-top: 3px; }
    .modal-backdrop { position: fixed; inset: 0; z-index: 20; display: grid; place-items: center; padding: 20px; background: rgba(15,23,42,.58); }
    .modal { width: min(860px, 100%); max-height: calc(100vh - 40px); overflow: auto; background: var(--bg); border: 1px solid var(--line); border-radius: 8px; box-shadow: 0 24px 80px rgba(15,23,42,.35); }
    .modal-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 18px; border-bottom: 1px solid var(--line); }
    .modal-body { padding: 18px; }
    .modal-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
    .modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }
    .file-list { margin-top: 16px; max-height: 260px; overflow: auto; }
    @media (max-width: 820px) {
	      header, .generated { align-items: stretch; flex-direction: column; }
      .grid, .filters { grid-template-columns: 1fr; }
      .client-filters, .stats-grid, .features-grid, .ip-bests-filters, .qshare-filters, .modal-grid { grid-template-columns: 1fr; }
	      .header-actions { flex-direction: column; align-items: stretch; }
	      button { width: 100%; }
	    }
	  </style>
	</head>
	<body>
	  <main id="loginView" class="login-screen">
	    <section class="login-card">
	      <div class="brand">
	        <div class="brand-mark">Q</div>
	        <div>
	          <div class="brand-title">Qmby License</div>
	          <div class="brand-subtitle">授权服务管理后台</div>
	        </div>
	      </div>
	      <div class="login-form">
	        <div>
	          <label>管理员账号</label>
	          <input id="adminUser" autocomplete="username" placeholder="admin" />
	        </div>
	        <div>
	          <label>管理员密码</label>
	          <input id="adminPass" type="password" autocomplete="current-password" placeholder="请输入密码" onkeydown="handleLoginKey(event)" />
	        </div>
	        <button id="loginButton" onclick="login()">登录</button>
	        <div id="loginError" class="login-error"></div>
	      </div>
	    </section>
	  </main>

	  <div id="appView" class="wrap hidden">
	    <header>
	      <div>
	        <h1>Qmby License Admin</h1>
	        <p>独立授权服务：生成邮箱绑定激活码，Qmby 联网后按邮箱校验会员。</p>
	      </div>
	      <div class="header-actions">
        <button class="secondary" onclick="toggleQshare()">Qshare管理</button>
        <button class="secondary" onclick="toggleIPBests()">IP归属地数据库</button>
        <button class="secondary" onclick="exportOrganizerFailedRecords()">导出整理失败记录</button>
        <button class="secondary" onclick="refreshAll()">刷新</button>
        <button class="secondary" onclick="logout()">退出登录</button>
      </div>
	    </header>

    <section class="panel">
      <div class="panel-head"><h2>客户端统计</h2></div>
      <div class="panel-body">
        <div class="stats-grid">
          <div class="stat"><div id="statInstalled" class="stat-value">-</div><div class="stat-label">累计联网安装</div></div>
          <div class="stat"><div id="statActive" class="stat-value">-</div><div class="stat-label">正在使用（10分钟）</div></div>
          <div class="stat"><div id="statActiveMembers" class="stat-value">-</div><div class="stat-label">活跃会员客户端</div></div>
          <div class="stat"><div id="statChecks24h" class="stat-value">-</div><div class="stat-label">24小时校验</div></div>
          <div class="stat"><div id="statIPReports" class="stat-value">-</div><div class="stat-label">IP归属地记录</div></div>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head"><h2>功能权限</h2></div>
      <div class="panel-body">
        <div id="featureRows" class="features-grid"></div>
        <div style="height:14px"></div>
        <button onclick="saveFeatures()">保存功能权限</button>
        <div id="featureMessage" class="msg"></div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head"><h2>生成激活码</h2></div>
      <div class="panel-body">
        <div class="grid">
          <div><label>绑定邮箱</label><input id="email" placeholder="user@example.com" /></div>
          <div>
            <label>会员等级</label>
            <select id="level">
              <option value="trial">7天试用</option>
              <option value="yearly">年费会员</option>
              <option value="permanent">永久会员</option>
              <option value="beta">内测会员</option>
            </select>
          </div>
          <div><label>备注</label><input id="note" placeholder="来源、订单号或说明" /></div>
          <button onclick="createCode()">生成</button>
        </div>
        <div id="generated"></div>
        <div id="message" class="msg"></div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head"><h2>激活码列表</h2></div>
      <div class="panel-body">
        <div class="filters">
          <div><label>搜索</label><input id="search" placeholder="邮箱、前缀或备注" oninput="loadCodesDebounced()" /></div>
          <div><label>状态</label><select id="status" onchange="loadCodes()"><option value="">全部</option><option value="issued">待激活</option><option value="active">已激活</option><option value="disabled">已禁用</option><option value="expired">已过期</option></select></div>
          <div><label>等级</label><select id="filterLevel" onchange="loadCodes()"><option value="">全部</option><option value="trial">7天试用</option><option value="yearly">年费</option><option value="permanent">永久</option><option value="beta">内测</option></select></div>
          <button class="secondary" onclick="loadCodes()">查询</button>
        </div>
        <div style="height:14px"></div>
        <div class="table-wrap"><table>
          <thead><tr><th>邮箱</th><th>等级</th><th>状态</th><th>前缀</th><th>开始</th><th>到期</th><th>最近联网</th><th>客户端数</th><th>操作</th></tr></thead>
          <tbody id="rows"><tr><td colspan="9">加载中...</td></tr></tbody>
        </table></div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head"><h2>联网客户端</h2></div>
      <div class="panel-body">
        <div class="client-filters">
          <div><label>搜索</label><input id="clientSearch" placeholder="邮箱、实例或IP" oninput="loadClientsDebounced()" /></div>
          <div><label>活跃</label><select id="clientActive" onchange="loadClients()"><option value="">全部</option><option value="1">10分钟内</option></select></div>
          <div><label>会员</label><select id="clientMember" onchange="loadClients()"><option value="">全部</option><option value="1">会员</option></select></div>
          <button class="secondary" onclick="loadClients()">查询</button>
        </div>
        <div style="height:14px"></div>
        <div class="table-wrap"><table>
          <thead><tr><th>邮箱 / 实例</th><th>会员</th><th>版本</th><th>Emby</th><th>IP</th><th>首次联网</th><th>最近联网</th><th>次数</th></tr></thead>
          <tbody id="clientRows"><tr><td colspan="8">加载中...</td></tr></tbody>
        </table></div>
      </div>
    </section>

    <section id="qsharePanel" class="panel hidden">
      <div class="panel-head"><h2>Qshare 共享内容</h2></div>
      <div class="panel-body">
        <div class="qshare-filters">
          <div><label>搜索标题、TMDB、发布者或路径</label><input id="qshareSearch" placeholder="标题、TMDB、邮箱、实例或路径" oninput="loadQshareDebounced()" /></div>
          <div><label>状态</label><select id="qshareStatus" onchange="loadQshareResources()"><option value="">全部</option><option value="published">已发布</option><option value="deleted">已下架</option></select></div>
          <button class="secondary" onclick="loadQshareResources()">查询</button>
          <button id="batchDeleteQshareBtn" class="danger" onclick="batchDeleteQshareResources()" disabled>批量删除</button>
        </div>
        <div style="height:14px"></div>
        <div class="table-wrap"><table>
          <thead><tr><th style="width:40px"><input type="checkbox" id="qshareSelectAll" onchange="toggleAllQshare()" /></th><th>资源</th><th>类型 / TMDB</th><th>发布者</th><th>文件</th><th>状态</th><th>更新时间</th><th>操作</th></tr></thead>
          <tbody id="qshareRows"><tr><td colspan="8">加载中...</td></tr></tbody>
        </table></div>
      </div>
    </section>

    <section id="ipBestsPanel" class="panel hidden">
      <div class="panel-head"><h2>IP归属地数据库</h2></div>
      <div class="panel-body">
        <div class="ip-bests-filters">
          <div><label>搜索 IP、地址或运营商</label><input id="ipBestSearch" placeholder="IP、地址或运营商" oninput="loadIPBestsDebounced()" /></div>
          <button class="secondary" onclick="loadIPBests()">查询</button>
        </div>
        <div style="height:14px"></div>
        <div class="table-wrap"><table>
          <thead><tr><th>IP</th><th>位置</th><th>区划</th><th>街道</th><th>运营商</th><th>经纬度</th><th>上报次数</th><th>更新时间</th></tr></thead>
          <tbody id="ipBestRows"><tr><td colspan="8">加载中...</td></tr></tbody>
        </table></div>
      </div>
    </section>
  </div>

  <div id="qshareModal" class="modal-backdrop hidden" onclick="closeQshareModal(event)">
    <section class="modal" onclick="event.stopPropagation()">
      <div class="modal-head">
        <div><h2 style="margin:0">编辑 Qshare 资源</h2><div id="qshareOwner" class="note"></div></div>
        <button class="secondary" onclick="closeQshareModal()">关闭</button>
      </div>
      <div class="modal-body">
        <input id="qshareEditID" type="hidden" />
        <div class="modal-grid">
          <div><label>标题</label><input id="qshareEditTitle" /></div>
          <div><label>年份</label><input id="qshareEditYear" type="number" /></div>
          <div><label>媒体类型</label><select id="qshareEditMediaType"><option value="movie">movie</option><option value="tv">tv</option></select></div>
          <div><label>TMDB ID</label><input id="qshareEditTMDBID" /></div>
          <div style="grid-column:1/-1"><label>海报地址</label><input id="qshareEditPosterURL" /></div>
          <div style="grid-column:1/-1"><label>来源路径</label><input id="qshareEditSourcePath" /></div>
        </div>
        <div id="qshareEditMeta" class="msg"></div>
        <div class="file-list table-wrap"><table>
          <thead><tr><th>文件</th><th>相对路径</th><th>大小</th><th>SHA1</th></tr></thead>
          <tbody id="qshareFileRows"></tbody>
        </table></div>
        <div class="modal-actions">
          <button id="qshareUnpublishButton" class="danger" onclick="unpublishQshareResource()">下架</button>
          <button onclick="saveQshareResource()">保存修改</button>
        </div>
        <div id="qshareEditMessage" class="msg"></div>
      </div>
    </section>
  </div>

  <script>
    const levelLabels = { trial: "7天试用", yearly: "年费会员", permanent: "永久会员", beta: "内测会员" };
    const statusLabels = { issued: "待激活", active: "已激活", disabled: "已禁用", expired: "已过期" };
    const featureAccessLabels = { free: "普通可用", member: "会员可用", disabled: "关闭功能" };
    let features = [];
    let timer = 0;
    let clientTimer = 0;
    let ipBestTimer = 0;
    let qshareTimer = 0;

    function $(id) { return document.getElementById(id); }
	    function authHeader() {
	      const user = localStorage.getItem("adminUser") || "";
	      const pass = localStorage.getItem("adminPass") || "";
	      return "Basic " + btoa(user + ":" + pass);
	    }
	    function initAuth() {
	      $("adminUser").value = localStorage.getItem("adminUser") || "admin";
	      $("adminPass").value = localStorage.getItem("adminPass") || "";
	    }
	    function showLogin() {
	      $("loginView").classList.remove("hidden");
	      $("appView").classList.add("hidden");
	      $("loginButton").disabled = false;
	    }
	    function showApp() {
	      $("loginView").classList.add("hidden");
	      $("appView").classList.remove("hidden");
	    }
	    function handleLoginKey(event) {
	      if (event.key === "Enter") login();
	    }
	    async function login() {
	      $("loginError").textContent = "";
	      $("loginButton").disabled = true;
	      localStorage.setItem("adminUser", $("adminUser").value.trim());
	      localStorage.setItem("adminPass", $("adminPass").value);
	      try {
	        await api("/api/admin/codes?limit=1");
	        showApp();
        await refreshAll();
	      } catch (err) {
	        localStorage.removeItem("adminPass");
	        $("adminPass").value = "";
	        $("loginError").textContent = err.message === "unauthorized" ? "账号或密码不正确" : err.message;
	        showLogin();
	      }
	    }
	    function logout() {
	      localStorage.removeItem("adminPass");
	      $("adminPass").value = "";
	      showLogin();
	    }
	    async function api(path, options = {}) {
	      const headers = Object.assign({ Authorization: authHeader() }, options.headers || {});
	      if (options.body) headers["Content-Type"] = "application/json";
      const res = await fetch(path, Object.assign({}, options, { headers }));
      const data = await res.json().catch(() => ({}));
	      if (!res.ok) throw new Error(data.error || res.statusText);
	      return data;
	    }
	    function fmt(value, permanent) {
      if (!value) return permanent ? "永久" : "-";
      return new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
    }
    function fmtFull(value) {
      if (!value) return "-";
      return new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" });
    }
    function esc(value) {
      return String(value || "").replace(/[&<>"']/g, s => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[s]));
    }
    async function refreshAll() {
      await Promise.all([loadStats(), loadFeatures(), loadCodes(), loadClients()]);
    }
    async function loadStats() {
      try {
        const data = await api("/api/admin/stats");
        $("statInstalled").textContent = data.installed_clients || 0;
        $("statActive").textContent = data.active_clients || 0;
        $("statActiveMembers").textContent = data.active_members || 0;
        $("statChecks24h").textContent = data.license_checks_24h || 0;
        $("statIPReports").textContent = data.ip_reports || 0;
      } catch (err) {
        if (err.message === "unauthorized") showLogin();
      }
    }
    function loadCodesDebounced() {
      clearTimeout(timer);
      timer = setTimeout(loadCodes, 250);
    }
    function loadClientsDebounced() {
      clearTimeout(clientTimer);
      clientTimer = setTimeout(loadClients, 250);
    }
    async function loadFeatures() {
      try {
        const data = await api("/api/admin/features");
        features = data.features || [];
        $("featureRows").innerHTML = features.map((feature, index) =>
          "<div class=\"feature-card\">" +
            "<div>" +
              "<div class=\"feature-title\">" + esc(feature.label) + "</div>" +
              "<div class=\"note\"><code>" + esc(feature.key) + "</code></div>" +
            "</div>" +
            "<div><label>权限</label><select id=\"featureAccess" + index + "\">" +
              Object.keys(featureAccessLabels).map(key => "<option value=\"" + key + "\"" + (feature.access === key ? " selected" : "") + ">" + featureAccessLabels[key] + "</option>").join("") +
            "</select></div>" +
            "<div><label>状态</label><select id=\"featureEnabled" + index + "\">" +
              "<option value=\"1\"" + (feature.enabled ? " selected" : "") + ">启用</option>" +
              "<option value=\"0\"" + (!feature.enabled ? " selected" : "") + ">禁用</option>" +
            "</select></div>" +
          "</div>"
        ).join("") || "<p>暂无功能配置</p>";
      } catch (err) {
        if (err.message === "unauthorized") {
          showLogin();
          return;
        }
        $("featureRows").innerHTML = "<p>" + esc(err.message) + "</p>";
      }
    }
    async function saveFeatures() {
      $("featureMessage").textContent = "";
      try {
        const payload = features.map((feature, index) => ({
          key: feature.key,
          label: feature.label,
          access: $("featureAccess" + index).value,
          enabled: $("featureEnabled" + index).value === "1"
        }));
        const data = await api("/api/admin/features", {
          method: "PUT",
          body: JSON.stringify({ features: payload })
        });
        features = data.features || [];
        $("featureMessage").textContent = "功能权限已保存，Qmby 点击立即验证后生效";
        loadFeatures();
      } catch (err) {
        $("featureMessage").textContent = err.message;
      }
    }
    async function loadCodes() {
      const params = new URLSearchParams();
      if ($("search").value.trim()) params.set("search", $("search").value.trim());
      if ($("status").value) params.set("status", $("status").value);
      if ($("filterLevel").value) params.set("level", $("filterLevel").value);
      params.set("limit", "100");
	      try {
	        const data = await api("/api/admin/codes?" + params.toString());
        $("rows").innerHTML = (data.codes || []).map(row =>
          "<tr>" +
            "<td><strong>" + esc(row.email) + "</strong>" + (row.note ? "<div class=\"note\">" + esc(row.note) + "</div>" : "") + "</td>" +
            "<td>" + (levelLabels[row.level] || row.level) + "</td>" +
            "<td><span class=\"badge " + row.status + "\">" + (statusLabels[row.status] || row.status) + "</span></td>" +
            "<td><code>" + esc(row.code_prefix) + "</code></td>" +
            "<td>" + fmt(row.starts_at) + "</td>" +
            "<td>" + fmt(row.expires_at, !row.expires_at) + "</td>" +
            "<td>" + fmt(row.last_seen_at) + "</td>" +
            "<td>" + ((data.client_counts || {})[row.id] || 0) + "</td>" +
            "<td>" + (row.status !== "disabled" ? "<button class=\"danger\" onclick=\"disableCode(" + row.id + ")\">禁用</button>" : "") + "</td>" +
          "</tr>").join("") || "<tr><td colspan=\"9\">暂无激活码</td></tr>";
	      } catch (err) {
	        if (err.message === "unauthorized") {
	          showLogin();
	          return;
	        }
	        $("rows").innerHTML = "<tr><td colspan=\"9\">" + esc(err.message) + "</td></tr>";
	      }
    }
    async function createCode() {
      $("message").textContent = "";
      $("generated").innerHTML = "";
      try {
        const data = await api("/api/admin/codes", {
          method: "POST",
          body: JSON.stringify({ email: $("email").value, level: $("level").value, note: $("note").value })
        });
        $("generated").innerHTML = "<div class=\"generated\"><div><p>新激活码，只显示一次</p><code>" + esc(data.plain_code) + "</code></div><button class=\"secondary\" onclick=\"navigator.clipboard.writeText('" + esc(data.plain_code) + "')\">复制</button></div>";
        $("email").value = "";
        $("note").value = "";
        $("message").textContent = "激活码已生成并绑定邮箱";
        refreshAll();
      } catch (err) {
        $("message").textContent = err.message;
      }
    }
    async function disableCode(id) {
      if (!confirm("确定禁用这个激活码吗？")) return;
      try {
        await api("/api/admin/codes/" + id + "/disable", { method: "POST" });
        refreshAll();
      } catch (err) {
        alert(err.message);
      }
	    }
    async function loadClients() {
      const params = new URLSearchParams();
      if ($("clientSearch").value.trim()) params.set("search", $("clientSearch").value.trim());
      if ($("clientActive").value) params.set("active", $("clientActive").value);
      if ($("clientMember").value) params.set("member", $("clientMember").value);
      params.set("limit", "100");
      try {
        const data = await api("/api/admin/clients?" + params.toString());
        $("clientRows").innerHTML = (data.clients || []).map(row => {
          const member = row.member ? "是" : "否";
          const instance = row.instance_id ? "<div class=\"note\"><code>" + esc(row.instance_id) + "</code></div>" : "";
          return "<tr>" +
            "<td><strong>" + esc(row.email || "-") + "</strong>" + instance + "</td>" +
            "<td><span class=\"badge " + (row.member ? "active" : "disabled") + "\">" + member + "</span><div class=\"note\">" + esc(row.status || "-") + "</div></td>" +
            "<td>" + esc(row.qmby_version || "-") + "</td>" +
            "<td>" + esc(row.emby_server || "-") + "</td>" +
            "<td><code>" + esc(row.last_ip || "-") + "</code></td>" +
            "<td>" + fmtFull(row.first_seen_at) + "</td>" +
            "<td>" + fmtFull(row.last_seen_at) + "</td>" +
            "<td>" + (row.report_count || 0) + "</td>" +
          "</tr>";
        }).join("") || "<tr><td colspan=\"8\">暂无客户端记录</td></tr>";
      } catch (err) {
        if (err.message === "unauthorized") {
          showLogin();
          return;
        }
        $("clientRows").innerHTML = "<tr><td colspan=\"8\">" + esc(err.message) + "</td></tr>";
      }
    }
    function toggleQshare() {
      const panel = $("qsharePanel");
      panel.classList.toggle("hidden");
      if (!panel.classList.contains("hidden")) loadQshareResources();
    }
    function loadQshareDebounced() {
      clearTimeout(qshareTimer);
      qshareTimer = setTimeout(loadQshareResources, 250);
    }
    function fmtBytes(value) {
      let size = Number(value || 0);
      const units = ["B", "KB", "MB", "GB", "TB"];
      let unit = 0;
      while (size >= 1024 && unit < units.length - 1) {
        size /= 1024;
        unit++;
      }
      return (unit === 0 ? size : size.toFixed(2)) + " " + units[unit];
    }
    async function loadQshareResources() {
      const params = new URLSearchParams();
      if ($("qshareSearch").value.trim()) params.set("search", $("qshareSearch").value.trim());
      if ($("qshareStatus").value) params.set("status", $("qshareStatus").value);
      params.set("limit", "200");
      try {
        const data = await api("/api/admin/qshare/resources?" + params.toString());
        $("qshareRows").innerHTML = (data.resources || []).map(row =>
          "<tr>" +
            "<td><input type=\"checkbox\" class=\"qshare-checkbox\" value=\"" + row.id + "\" onchange=\"updateBatchDeleteBtn()\" /></td>" +
            "<td><strong>" + esc(row.title) + "</strong><div class=\"note\">" + esc(row.source_path) + "</div></td>" +
            "<td><span class=\"badge issued\">" + esc(row.media_type) + "</span><div class=\"note\">TMDB " + esc(row.tmdb_id || "-") + " · " + esc(row.year) + "</div></td>" +
            "<td><strong>" + esc(row.owner_label) + "</strong><div class=\"note\">" + esc(row.publisher_email) + "</div><div class=\"note\"><code>" + esc(row.instance_id) + "</code></div></td>" +
            "<td>" + (row.file_count || 0) + "<div class=\"note\">" + fmtBytes(row.total_size) + "</div></td>" +
            "<td><span class=\"badge " + (row.status === "published" ? "active" : "disabled") + "\">" + (row.status === "published" ? "已发布" : "已下架") + "</span></td>" +
            "<td>" + fmtFull(row.updated_at) + "</td>" +
            "<td><button class=\"secondary\" onclick=\"openQshareResource(" + row.id + ")\">查看 / 编辑</button></td>" +
          "</tr>"
        ).join("") || "<tr><td colspan=\"8\">暂无 Qshare 内容</td></tr>";
        $("qshareSelectAll").checked = false;
        updateBatchDeleteBtn();
      } catch (err) {
        if (err.message === "unauthorized") { showLogin(); return; }
        $("qshareRows").innerHTML = "<tr><td colspan=\"7\">" + esc(err.message) + "</td></tr>";
      }
    }
    async function openQshareResource(id) {
      try {
        const data = await api("/api/admin/qshare/resources/" + id);
        const row = data.resource;
        $("qshareEditID").value = row.id;
        $("qshareEditTitle").value = row.title || "";
        $("qshareEditYear").value = row.year || "";
        $("qshareEditMediaType").value = row.media_type || "movie";
        $("qshareEditTMDBID").value = row.tmdb_id || "";
        $("qshareEditPosterURL").value = row.poster_url || "";
        $("qshareEditSourcePath").value = row.source_path || "";
        $("qshareOwner").textContent = (row.owner_label || "") + " · " + (row.publisher_email || "") + " · " + (row.instance_id || "");
        $("qshareEditMeta").textContent = "状态：" + (row.status === "published" ? "已发布" : "已下架") + " · " + (row.file_count || 0) + " 个文件 · " + fmtBytes(row.total_size);
        $("qshareUnpublishButton").disabled = row.status !== "published";
        $("qshareEditMessage").textContent = "";
        $("qshareFileRows").innerHTML = (row.files || []).map(file =>
          "<tr><td><strong>" + esc(file.name) + "</strong></td><td>" + esc(file.relative_path) + "</td><td>" + fmtBytes(file.size) + "</td><td><code>" + esc(file.sha1) + "</code></td></tr>"
        ).join("") || "<tr><td colspan=\"4\">暂无文件</td></tr>";
        $("qshareModal").classList.remove("hidden");
      } catch (err) {
        alert(err.message);
      }
    }
    function closeQshareModal(event) {
      if (event && event.target !== $("qshareModal")) return;
      $("qshareModal").classList.add("hidden");
    }
    async function saveQshareResource() {
      const id = $("qshareEditID").value;
      $("qshareEditMessage").textContent = "";
      try {
        await api("/api/admin/qshare/resources/" + id, {
          method: "PUT",
          body: JSON.stringify({
            title: $("qshareEditTitle").value,
            year: Number($("qshareEditYear").value),
            media_type: $("qshareEditMediaType").value,
            tmdb_id: $("qshareEditTMDBID").value,
            poster_url: $("qshareEditPosterURL").value,
            source_path: $("qshareEditSourcePath").value
          })
        });
        $("qshareEditMessage").textContent = "资源信息已保存";
        loadQshareResources();
      } catch (err) {
        $("qshareEditMessage").textContent = err.message;
      }
    }
    async function unpublishQshareResource() {
      const id = $("qshareEditID").value;
      if (!confirm("确定下架这个 Qshare 资源吗？")) return;
      try {
        await api("/api/admin/qshare/resources/" + id + "/unpublish", { method: "POST" });
        closeQshareModal();
        loadQshareResources();
      } catch (err) {
        $("qshareEditMessage").textContent = err.message;
      }
    }
    function toggleAllQshare() {
      const checked = $("qshareSelectAll").checked;
      document.querySelectorAll(".qshare-checkbox").forEach(cb => cb.checked = checked);
      updateBatchDeleteBtn();
    }
    function updateBatchDeleteBtn() {
      const count = document.querySelectorAll(".qshare-checkbox:checked").length;
      $("batchDeleteQshareBtn").disabled = count === 0;
      $("batchDeleteQshareBtn").textContent = count > 0 ? "批量删除 (" + count + ")" : "批量删除";
    }
    async function batchDeleteQshareResources() {
      const ids = Array.from(document.querySelectorAll(".qshare-checkbox:checked")).map(cb => Number(cb.value));
      if (ids.length === 0) return;
      if (!confirm("确定要永久删除选中的 " + ids.length + " 个 Qshare 资源吗？\n\n此操作不可撤销，相关文件记录也将一并删除。")) return;
      try {
        const data = await api("/api/admin/qshare/resources/batch-delete", {
          method: "POST",
          body: JSON.stringify({ ids })
        });
        alert("成功删除 " + (data.deleted || 0) + " 个资源");
        loadQshareResources();
      } catch (err) {
        alert("删除失败：" + err.message);
      }
    }
    function toggleIPBests() {
      const panel = $("ipBestsPanel");
      panel.classList.toggle("hidden");
      if (!panel.classList.contains("hidden")) loadIPBests();
    }
    function loadIPBestsDebounced() {
      clearTimeout(ipBestTimer);
      ipBestTimer = setTimeout(loadIPBests, 250);
    }
    async function loadIPBests() {
      const params = new URLSearchParams();
      if ($("ipBestSearch").value.trim()) params.set("search", $("ipBestSearch").value.trim());
      params.set("limit", "100");
      try {
        const data = await api("/api/admin/ip-bests?" + params.toString());
        $("ipBestRows").innerHTML = (data.bests || []).map(row => {
          const location = [row.location, row.district, row.street].filter(Boolean).join(" · ");
          const latlng = [row.latitude, row.longitude].filter(v => v != null).join(", ");
          return "<tr>" +
            "<td><code>" + esc(row.ip) + "</code></td>" +
            "<td>" + esc(row.location || "-") + "</td>" +
            "<td>" + esc(row.district || "-") + "</td>" +
            "<td>" + esc(row.street || "-") + "</td>" +
            "<td>" + esc(row.isp || "-") + "</td>" +
            "<td>" + esc(latlng || "-") + "</td>" +
            "<td>" + (row.count || 0) + "</td>" +
            "<td>" + fmtFull(row.updated_at) + "</td>" +
          "</tr>";
        }).join("") || "<tr><td colspan=\"8\">暂无IP归属地数据</td></tr>";
      } catch (err) {
        if (err.message === "unauthorized") { showLogin(); return; }
        $("ipBestRows").innerHTML = "<tr><td colspan=\"8\">" + esc(err.message) + "</td></tr>";
      }
    }
    async function exportOrganizerFailedRecords() {
      try {
        const res = await fetch("/api/admin/organizer/failed-records/export", { headers: { Authorization: authHeader() } });
        if (res.status === 401) {
          showLogin();
          return;
        }
        if (!res.ok) throw new Error(res.statusText);
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        const disposition = res.headers.get("Content-Disposition") || "";
        const match = disposition.match(/filename="([^"]+)"/);
        a.href = url;
        a.download = match ? match[1] : "organizer_failed_records.csv";
        document.body.appendChild(a);
        a.click();
        a.remove();
        URL.revokeObjectURL(url);
      } catch (err) {
        alert(err.message);
      }
    }
    initAuth();
	    if (localStorage.getItem("adminPass")) {
	      login();
	    } else {
	      showLogin();
	    }
	  </script>
</body>
</html>`
