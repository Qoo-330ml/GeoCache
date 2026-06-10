package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdminQshare(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, adminQshareHTML)
}

const adminQshareHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Qshare Admin</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #f6f8fb;
      --panel: rgba(255,255,255,.82);
      --text: #172033;
      --muted: #617086;
      --line: #dfe6ef;
      --primary: #0ea5e9;
      --danger: #ef4444;
      --success: #4fbd31;
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
    .hidden { display: none !important; }
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
      background: linear-gradient(135deg, var(--primary), #0fbaa7);
      box-shadow: 0 12px 28px rgba(14,165,233,.25);
    }
    .brand-title { font-size: 19px; font-weight: 750; line-height: 1.2; }
    .brand-subtitle { color: var(--muted); font-size: 13px; margin-top: 3px; }
    .login-form { display: grid; gap: 14px; }
    .login-form input, .login-form button { height: 44px; }
    .login-error { min-height: 20px; color: var(--danger); font-size: 13px; }
    .wrap { max-width: 1180px; margin: 0 auto; padding: 28px 18px 48px; }
    header { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 22px; }
    h1 { font-size: 26px; margin: 0 0 6px; letter-spacing: 0; }
    p { color: var(--muted); margin: 0; }
    .header-actions { display: flex; gap: 10px; align-items: center; }
    .panel {
      background: var(--panel);
      border: 1px solid rgba(148,163,184,.22);
      border-radius: 8px;
      box-shadow: 0 18px 60px rgba(15,23,42,.10);
      backdrop-filter: blur(14px);
      margin-bottom: 18px;
    }
    .panel-head { padding: 18px 18px 0; }
    .panel h2 { font-size: 16px; margin: 0; }
    .panel-body { padding: 18px; }
    .qshare-filters { display: grid; grid-template-columns: 1fr 150px auto auto; gap: 12px; align-items: end; }
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
    table { width: 100%; border-collapse: collapse; min-width: 920px; font-size: 14px; }
    th, td { padding: 12px 14px; border-top: 1px solid var(--line); text-align: left; vertical-align: top; }
    th { color: var(--muted); font-size: 12px; font-weight: 650; background: rgba(100,116,139,.08); }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-weight: 700; overflow-wrap: anywhere; }
    .table-wrap { overflow-x: auto; border: 1px solid var(--line); border-radius: 8px; }
    .badge { display: inline-flex; padding: 4px 8px; border-radius: 7px; font-size: 12px; }
    .issued { background: rgba(14,165,233,.12); color: var(--primary); }
    .active { background: rgba(79,189,49,.12); color: var(--success); }
    .disabled { background: rgba(100,116,139,.16); color: var(--muted); }
    .note { color: var(--muted); font-size: 12px; margin-top: 3px; }
    .msg { margin-top: 12px; color: var(--muted); font-size: 14px; min-height: 20px; }
    .modal-backdrop { position: fixed; inset: 0; z-index: 20; display: grid; place-items: center; padding: 20px; background: rgba(15,23,42,.58); }
    .modal { width: min(860px, 100%); max-height: calc(100vh - 40px); overflow: auto; background: var(--bg); border: 1px solid var(--line); border-radius: 8px; box-shadow: 0 24px 80px rgba(15,23,42,.35); }
    .modal-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 18px; border-bottom: 1px solid var(--line); }
    .modal-body { padding: 18px; }
    .modal-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
    .modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }
    .file-list { margin-top: 16px; max-height: 260px; overflow: auto; }
    @media (max-width: 820px) {
      header { align-items: stretch; flex-direction: column; }
      .header-actions { flex-direction: column; align-items: stretch; }
      .qshare-filters, .modal-grid { grid-template-columns: 1fr; }
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
          <div class="brand-title">Qshare Admin</div>
          <div class="brand-subtitle">共享内容管理</div>
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
        <h1>Qshare 管理</h1>
        <p>查看、编辑、下架或批量删除云端共享内容。</p>
      </div>
      <div class="header-actions">
        <button class="secondary" onclick="location.href='/admin'">返回后台</button>
        <button class="secondary" onclick="loadQshareResources()">刷新</button>
        <button class="secondary" onclick="logout()">退出登录</button>
      </div>
    </header>

    <section class="panel">
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
          <thead><tr><th>文件 / 文件夹</th><th>相对路径</th><th>大小</th><th>SHA1</th></tr></thead>
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
        await api("/api/admin/qshare/resources?limit=1");
        showApp();
        await loadQshareResources();
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
    function loadQshareDebounced() {
      clearTimeout(qshareTimer);
      qshareTimer = setTimeout(loadQshareResources, 250);
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
        $("qshareRows").innerHTML = "<tr><td colspan=\"8\">" + esc(err.message) + "</td></tr>";
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
        $("qshareFileRows").innerHTML = (row.files || []).map(file => {
          const name = (file.is_dir ? "文件夹 · " : "") + (file.name || "");
          return "<tr><td><strong>" + esc(name) + "</strong></td><td>" + esc(file.relative_path) + "</td><td>" + fmtBytes(file.size) + "</td><td><code>" + esc(file.sha1 || "-") + "</code></td></tr>";
        }).join("") || "<tr><td colspan=\"4\">暂无文件</td></tr>";
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
    initAuth();
    if (localStorage.getItem("adminPass")) {
      login();
    } else {
      showLogin();
    }
  </script>
</body>
</html>`
