package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdminIPBests(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderAdminPage(adminPage{
		Title:         "IP归属地 - Qmby License Admin",
		BrandTitle:    "Qmby License",
		BrandSubtitle: "IP归属地",
		Heading:       "IP归属地数据库",
		Description:   "查询客户端上报过的 IP、地址和运营商信息。",
		Actions: `
        <button class="secondary" onclick="location.href='/admin'">返回概览</button>
        <button class="secondary" onclick="loadIPBests()">刷新</button>
        <button class="secondary" onclick="logout()">退出登录</button>`,
		Body: `
    <section class="panel">
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
    </section>`,
		Script: `
    let ipBestTimer = 0;
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
    async function afterLogin() { await loadIPBests(); }`,
		LoginProbe: "/api/admin/ip-bests?limit=1",
	}))
}
