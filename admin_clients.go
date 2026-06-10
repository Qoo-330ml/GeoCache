package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdminClients(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderAdminPage(adminPage{
		Title:         "联网客户端 - Qmby License Admin",
		BrandTitle:    "Qmby License",
		BrandSubtitle: "联网客户端",
		Heading:       "联网客户端",
		Description:   "查看客户端实例、活跃状态、版本和最近联网信息。",
		Actions: `
        <button class="secondary" onclick="location.href='/admin'">返回概览</button>
        <button class="secondary" onclick="loadClients()">刷新</button>
        <button class="secondary" onclick="logout()">退出登录</button>`,
		Body: `
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
    </section>`,
		Script: `
    let clientTimer = 0;
    function loadClientsDebounced() {
      clearTimeout(clientTimer);
      clientTimer = setTimeout(loadClients, 250);
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
        if (err.message === "unauthorized") { showLogin(); return; }
        $("clientRows").innerHTML = "<tr><td colspan=\"8\">" + esc(err.message) + "</td></tr>";
      }
    }
    async function afterLogin() { await loadClients(); }`,
		LoginProbe: "/api/admin/clients?limit=1",
	}))
}
