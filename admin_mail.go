package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdminMail(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderAdminPage(adminPage{
		Title:         "邮件配置 - Qmby License Admin",
		BrandTitle:    "Qmby License",
		BrandSubtitle: "邮件配置",
		Heading:       "邮件配置",
		Description:   "配置 SMTP 后，生成激活码会自动发送到绑定邮箱。",
		Actions: `
        <button class="secondary" onclick="location.href='/admin'">返回概览</button>
        <button class="secondary" onclick="logout()">退出登录</button>`,
		Body: `
    <section class="panel">
      <div class="panel-head"><h2>SMTP</h2></div>
      <div class="panel-body">
        <div class="modal-grid">
          <div><label>SMTP Host</label><input id="host" placeholder="smtp.example.com" /></div>
          <div><label>SMTP Port</label><input id="port" placeholder="587" /></div>
          <div><label>账号</label><input id="username" placeholder="mailer@example.com" /></div>
          <div><label>密码</label><input id="password" type="password" placeholder="留空则保持原密码" /></div>
          <div><label>发件人</label><input id="from" placeholder="Qmby License <mailer@example.com>" /></div>
          <div><label>测试收件邮箱</label><input id="testEmail" placeholder="user@example.com" /></div>
        </div>
        <div class="modal-actions">
          <button onclick="saveMailSettings()">保存配置</button>
          <button class="secondary" onclick="sendTestMail()">发送测试邮件</button>
        </div>
        <div id="message" class="msg"></div>
      </div>
    </section>`,
		Script: `
    async function loadMailSettings() {
      try {
        const data = await api("/api/admin/mail-settings");
        $("host").value = data.host || "";
        $("port").value = data.port || "587";
        $("username").value = data.username || "";
        $("from").value = data.from || "";
        $("password").placeholder = data.password_set ? "已保存，留空则保持原密码" : "SMTP 密码";
        $("message").textContent = data.configured ? "邮件配置已启用" : "邮件配置未启用";
      } catch (err) {
        if (err.message === "unauthorized") { showLogin(); return; }
        $("message").textContent = err.message;
      }
    }
    async function saveMailSettings() {
      $("message").textContent = "";
      try {
        const data = await api("/api/admin/mail-settings", {
          method: "PUT",
          body: JSON.stringify({
            host: $("host").value,
            port: $("port").value,
            username: $("username").value,
            password: $("password").value,
            from: $("from").value
          })
        });
        $("password").value = "";
        $("password").placeholder = data.password_set ? "已保存，留空则保持原密码" : "SMTP 密码";
        $("message").textContent = data.configured ? "邮件配置已保存并启用" : "邮件配置已保存但未启用";
      } catch (err) {
        $("message").textContent = err.message;
      }
    }
    async function sendTestMail() {
      $("message").textContent = "";
      try {
        await api("/api/admin/mail-settings/test", {
          method: "POST",
          body: JSON.stringify({ email: $("testEmail").value })
        });
        $("message").textContent = "测试邮件已发送";
      } catch (err) {
        $("message").textContent = err.message;
      }
    }
    async function afterLogin() { await loadMailSettings(); }`,
		LoginProbe: "/api/admin/mail-settings",
	}))
}
