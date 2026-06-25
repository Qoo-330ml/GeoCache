package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdminPayment(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderAdminPage(adminPage{
		Title:         "支付订单 - Qmby License Admin",
		BrandTitle:    "Qmby License",
		BrandSubtitle: "支付订单",
		Heading:       "支付订单",
		Description:   "配置 XorPay，查看自动支付订单和激活码发放状态。",
		Actions: `
        <button class="secondary" onclick="location.href='/admin'">返回概览</button>
        <button class="secondary" onclick="loadOrders()">刷新订单</button>
        <button class="secondary" onclick="logout()">退出登录</button>`,
		Body: `
    <section class="panel">
      <div class="panel-head"><h2>XorPay 配置</h2></div>
      <div class="panel-body">
        <div class="modal-grid">
          <div><label>公网基础地址</label><input id="publicBaseURL" placeholder="https://license.example.com" /></div>
          <div><label>XorPay AID</label><input id="xorpayAID" /></div>
          <div><label>XorPay Secret</label><input id="xorpaySecret" type="password" placeholder="留空则保持原密钥" /></div>
          <div><label>订单过期秒数</label><input id="orderExpireSeconds" type="number" min="300" value="7200" /></div>
          <div><label>支付宝当面付</label><select id="payAlipay"><option value="1">启用</option><option value="0">停用</option></select></div>
          <div><label>微信 Native</label><select id="payNative"><option value="1">启用</option><option value="0">停用</option></select></div>
        </div>
        <div class="modal-actions">
          <button onclick="saveSettings()">保存配置</button>
        </div>
        <div id="settingsMessage" class="msg"></div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head"><h2>订单</h2></div>
      <div class="panel-body">
        <div class="filters">
          <div><label>搜索</label><input id="search" placeholder="邮箱、订单号、XorPay 单号" oninput="loadOrdersDebounced()" /></div>
          <div><label>状态</label><select id="status" onchange="loadOrders()"><option value="">全部</option><option value="pending">待支付</option><option value="paid">已支付</option><option value="fulfilled">已发码</option><option value="activated">已激活</option><option value="expired">已过期</option><option value="failed">失败</option></select></div>
          <button class="secondary" onclick="loadOrders()">查询</button>
        </div>
        <div class="table-wrap" style="margin-top:14px">
          <table>
            <thead><tr><th>订单号</th><th>邮箱</th><th>套餐</th><th>金额</th><th>支付</th><th>状态</th><th>创建/过期</th><th>错误</th></tr></thead>
            <tbody id="orders"></tbody>
          </table>
        </div>
        <div id="ordersMessage" class="msg"></div>
      </div>
    </section>`,
		Script: `
    const statusLabels = { pending: "待支付", paid: "已支付", fulfilled: "已发码", activated: "已激活", expired: "已过期", failed: "失败" };
    const planLabels = { plus_year: "Plus 年费", plus_lifetime: "Plus 永久", pro_year: "Pro 年费", pro_lifetime: "Pro 永久" };
    let ordersTimer = null;
    function money(cents) { return "¥" + (Number(cents || 0) / 100).toFixed(2); }
    function dateTime(value) { return value ? new Date(value).toLocaleString() : "-"; }
    function enabledTypes() {
      const values = [];
      if ($("payAlipay").value === "1") values.push("alipay");
      if ($("payNative").value === "1") values.push("native");
      return values;
    }
    async function loadSettings() {
      try {
        const data = await api("/api/admin/payment/settings");
        $("publicBaseURL").value = data.public_base_url || "";
        $("xorpayAID").value = data.xorpay_aid || "";
        $("xorpaySecret").placeholder = data.xorpay_configured ? "已保存，留空则保持原密钥" : "XorPay Secret";
        $("orderExpireSeconds").value = data.order_expire_seconds || 7200;
        const types = data.enabled_pay_types || [];
        $("payAlipay").value = types.includes("alipay") ? "1" : "0";
        $("payNative").value = types.includes("native") ? "1" : "0";
        $("settingsMessage").textContent = data.xorpay_configured ? "XorPay 已配置" : "XorPay 未配置";
      } catch (err) {
        if (err.message === "unauthorized") { showLogin(); return; }
        $("settingsMessage").textContent = err.message;
      }
    }
    async function saveSettings() {
      $("settingsMessage").textContent = "";
      try {
        const data = await api("/api/admin/payment/settings", {
          method: "PUT",
          body: JSON.stringify({
            public_base_url: $("publicBaseURL").value,
            xorpay_aid: $("xorpayAID").value,
            xorpay_secret: $("xorpaySecret").value,
            enabled_pay_types: enabledTypes(),
            order_expire_seconds: Number($("orderExpireSeconds").value)
          })
        });
        $("xorpaySecret").value = "";
        $("xorpaySecret").placeholder = data.xorpay_configured ? "已保存，留空则保持原密钥" : "XorPay Secret";
        $("settingsMessage").textContent = data.xorpay_configured ? "支付配置已保存并启用" : "支付配置已保存但未启用";
      } catch (err) {
        $("settingsMessage").textContent = err.message;
      }
    }
    function loadOrdersDebounced() {
      clearTimeout(ordersTimer);
      ordersTimer = setTimeout(loadOrders, 250);
    }
    async function loadOrders() {
      $("ordersMessage").textContent = "";
      const params = new URLSearchParams();
      if ($("search").value) params.set("search", $("search").value);
      if ($("status").value) params.set("status", $("status").value);
      try {
        const data = await api("/api/admin/payment/orders?" + params.toString());
        $("orders").innerHTML = (data.orders || []).map(order =>
          "<tr>" +
          "<td><code>" + esc(order.order_id) + "</code><div class=\"note\">" + esc(order.provider_order_id || "") + "</div></td>" +
          "<td>" + esc(order.email || "") + "<div class=\"note\">" + esc(order.instance_id || "") + "</div></td>" +
          "<td>" + esc(planLabels[order.plan_key] || order.plan_key || "") + "<div class=\"note\">" + esc(order.duration_days ? order.duration_days + " 天" : "永久") + "</div></td>" +
          "<td>" + money(order.amount_cents) + "</td>" +
          "<td>" + esc(order.pay_type || "") + "<div class=\"note\">" + dateTime(order.paid_at) + "</div></td>" +
          "<td><span class=\"badge " + esc(order.status || "") + "\">" + esc(statusLabels[order.status] || order.status || "") + "</span></td>" +
          "<td>" + dateTime(order.created_at) + "<div class=\"note\">" + dateTime(order.expires_at) + "</div></td>" +
          "<td>" + esc(order.last_error || "") + "</td>" +
          "</tr>"
        ).join("");
        $("ordersMessage").textContent = "共 " + (data.total || 0) + " 条订单";
      } catch (err) {
        if (err.message === "unauthorized") { showLogin(); return; }
        $("ordersMessage").textContent = err.message;
      }
    }
    async function afterLogin() { await loadSettings(); await loadOrders(); }`,
		LoginProbe: "/api/admin/payment/settings",
	}))
}
