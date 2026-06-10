package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdminFeatures(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderAdminPage(adminPage{
		Title:         "功能权限 - Qmby License Admin",
		BrandTitle:    "Qmby License",
		BrandSubtitle: "功能权限",
		Heading:       "功能权限",
		Description:   "控制各云端功能的可用范围。",
		Actions: `
        <button class="secondary" onclick="location.href='/admin'">返回概览</button>
        <button class="secondary" onclick="loadFeatures()">刷新</button>
        <button class="secondary" onclick="logout()">退出登录</button>`,
		Body: `
    <section class="panel">
      <div class="panel-head"><h2>功能权限</h2></div>
      <div class="panel-body">
        <div id="featureRows" class="features-grid"></div>
        <div style="height:14px"></div>
        <button onclick="saveFeatures()">保存功能权限</button>
        <div id="featureMessage" class="msg"></div>
      </div>
    </section>`,
		Script: `
    const featureAccessLabels = { free: "普通可用", member: "会员可用", disabled: "关闭功能" };
    let features = [];
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
        if (err.message === "unauthorized") { showLogin(); return; }
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
    async function afterLogin() { await loadFeatures(); }`,
		LoginProbe: "/api/admin/features",
	}))
}
