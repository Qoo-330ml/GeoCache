package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdminCodes(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderAdminPage(adminPage{
		Title:         "激活码 - Qmby License Admin",
		BrandTitle:    "Qmby License",
		BrandSubtitle: "激活码管理",
		Heading:       "激活码管理",
		Description:   "生成、查询和禁用邮箱绑定激活码。",
		Actions: `
        <button class="secondary" onclick="location.href='/admin'">返回概览</button>
        <button class="secondary" onclick="refreshCodes()">刷新</button>
        <button class="secondary" onclick="logout()">退出登录</button>`,
		Body: `
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
    </section>`,
		Script: `
    const levelLabels = { trial: "7天试用", yearly: "年费会员", permanent: "永久会员", beta: "内测会员" };
    const statusLabels = { issued: "待激活", active: "已激活", disabled: "已禁用", expired: "已过期" };
    let timer = 0;
    function loadCodesDebounced() {
      clearTimeout(timer);
      timer = setTimeout(loadCodes, 250);
    }
    async function refreshCodes() { await loadCodes(); }
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
        if (err.message === "unauthorized") { showLogin(); return; }
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
        $("message").textContent = data.mail_sent ? "激活码已生成并发送到绑定邮箱" : ("激活码已生成但未发送邮件" + (data.mail_error ? "：" + data.mail_error : ""));
        loadCodes();
      } catch (err) {
        $("message").textContent = err.message;
      }
    }
    async function disableCode(id) {
      if (!confirm("确定禁用这个激活码吗？")) return;
      try {
        await api("/api/admin/codes/" + id + "/disable", { method: "POST" });
        loadCodes();
      } catch (err) {
        alert(err.message);
      }
    }
    async function afterLogin() { await loadCodes(); }`,
		LoginProbe: "/api/admin/codes?limit=1",
	}))
}
