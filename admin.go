package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdmin(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderAdminPage(adminPage{
		Title:         "Qmby License Admin",
		BrandTitle:    "Qmby License",
		BrandSubtitle: "授权服务管理后台",
		Heading:       "Qmby License Admin",
		Description:   "独立授权服务：生成邮箱绑定激活码，Qmby 联网后按邮箱校验会员。",
		Actions: `
        <button class="secondary" onclick="location.href='/admin/features'">功能权限</button>
        <button class="secondary" onclick="location.href='/admin/mail'">邮件配置</button>
        <button class="secondary" onclick="location.href='/admin/codes'">激活码</button>
        <button class="secondary" onclick="location.href='/admin/clients'">联网客户端</button>
        <button class="secondary" onclick="location.href='/admin/ip-bests'">IP归属地</button>
        <button class="secondary" onclick="location.href='/admin/qshare'">Qshare</button>
        <button class="secondary" onclick="exportOrganizerFailedRecords()">导出整理失败记录</button>
        <button class="secondary" onclick="loadStats()">刷新</button>
        <button class="secondary" onclick="logout()">退出登录</button>`,
		Body: `
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
      <div class="panel-head"><h2>功能入口</h2></div>
      <div class="panel-body">
        <div class="nav-grid">
          <a class="nav-card" href="/admin/features"><strong>功能权限</strong><span>控制各功能普通用户、会员或关闭状态。</span></a>
          <a class="nav-card" href="/admin/mail"><strong>邮件配置</strong><span>配置 SMTP，生成激活码后自动发送到绑定邮箱。</span></a>
          <a class="nav-card" href="/admin/codes"><strong>激活码</strong><span>生成、查询和禁用邮箱绑定激活码。</span></a>
          <a class="nav-card" href="/admin/clients"><strong>联网客户端</strong><span>查看客户端实例、版本、活跃和会员状态。</span></a>
          <a class="nav-card" href="/admin/ip-bests"><strong>IP归属地</strong><span>查询客户端上报过的 IP 归属地记录。</span></a>
          <a class="nav-card" href="/admin/qshare"><strong>Qshare</strong><span>管理云端共享内容及文件元数据。</span></a>
        </div>
      </div>
    </section>`,
		Script: `
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
    async function afterLogin() { await loadStats(); }`,
		LoginProbe: "/api/admin/stats",
	}))
}

type adminPage struct {
	Title         string
	BrandTitle    string
	BrandSubtitle string
	Heading       string
	Description   string
	Actions       string
	Body          string
	Script        string
	LoginProbe    string
}

func renderAdminPage(page adminPage) string {
	return adminHTMLStart(page) + page.Body + adminHTMLMiddle(page) + page.Script + adminHTMLEnd(page.LoginProbe)
}

func adminHTMLStart(page adminPage) string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>` + page.Title + `</title>
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
    .login-screen { min-height: 100vh; display: grid; place-items: center; padding: 24px; }
    .login-card {
      width: min(420px, 100%);
      background: var(--panel);
      border: 1px solid rgba(148,163,184,.24);
      border-radius: 10px;
      box-shadow: 0 24px 80px rgba(15,23,42,.14);
      backdrop-filter: blur(16px);
      padding: 28px;
    }
    .brand { display: flex; align-items: center; gap: 12px; margin-bottom: 24px; }
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
    .login-form input, .login-form button { height: 44px; }
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
    .filters { display: grid; grid-template-columns: 1fr 150px 150px auto; gap: 12px; align-items: end; }
    .client-filters { display: grid; grid-template-columns: 1fr 130px 130px auto; gap: 12px; align-items: end; }
    .ip-bests-filters { display: grid; grid-template-columns: 1fr auto; gap: 12px; align-items: end; }
    .qshare-filters { display: grid; grid-template-columns: 1fr 150px auto auto; gap: 12px; align-items: end; }
    .stats-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 12px; }
    .features-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 12px; }
    .nav-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 12px; }
    .nav-card, .feature-card, .stat {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 14px;
      background: rgba(255,255,255,.38);
    }
    .nav-card { display: grid; gap: 6px; color: var(--text); text-decoration: none; }
    .nav-card span { color: var(--muted); font-size: 13px; line-height: 1.45; }
    .feature-card { display: grid; gap: 10px; }
    .feature-title { font-weight: 700; }
    .stat-value { font-size: 25px; font-weight: 780; line-height: 1.1; }
    .stat-label { color: var(--muted); font-size: 12px; margin-top: 6px; }
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
    .header-actions { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; justify-content: flex-end; }
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
      .grid, .filters, .client-filters, .stats-grid, .features-grid, .ip-bests-filters, .qshare-filters, .modal-grid { grid-template-columns: 1fr; }
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
          <div class="brand-title">` + page.BrandTitle + `</div>
          <div class="brand-subtitle">` + page.BrandSubtitle + `</div>
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
        <h1>` + page.Heading + `</h1>
        <p>` + page.Description + `</p>
      </div>
      <div class="header-actions">` + page.Actions + `
      </div>
    </header>
`
}

func adminHTMLMiddle(page adminPage) string {
	return `
  </div>

  <script>
    const adminLoginProbe = "` + page.LoginProbe + `";

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
        await api(adminLoginProbe);
        showApp();
        await afterLogin();
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
    function esc(value) {
      return String(value || "").replace(/[&<>"']/g, s => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[s]));
    }
    function fmt(value, permanent) {
      if (!value) return permanent ? "永久" : "-";
      return new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
    }
    function fmtFull(value) {
      if (!value) return "-";
      return new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" });
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
`
}

func adminHTMLEnd(loginProbe string) string {
	return `
    initAuth();
    if (localStorage.getItem("adminPass")) {
      login();
    } else {
      showLogin();
    }
  </script>
</body>
</html>`
}
