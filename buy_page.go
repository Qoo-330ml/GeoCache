package main

const buyHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>购买 Qmby 激活码</title>
  <style>
    :root { --bg:#f7fafc; --text:#132033; --muted:#66758a; --card:#ffffffd9; --line:#dce5ef; --primary:#0ea5e9; --teal:#0fbaa7; --warn:#f59e0b; }
    @media (prefers-color-scheme: dark) { :root { --bg:#101722; --text:#eef4fb; --muted:#9caabc; --card:#162033d9; --line:#2a3548; } }
    * { box-sizing: border-box; }
    body { margin:0; min-height:100vh; font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; color:var(--text); background:radial-gradient(circle at 18% 16%,rgba(14,165,233,.22),transparent 30rem),radial-gradient(circle at 86% 28%,rgba(15,186,167,.18),transparent 28rem),var(--bg); }
    .wrap { max-width:1080px; margin:0 auto; padding:34px 18px 54px; }
    .hero { display:flex; justify-content:space-between; gap:20px; align-items:flex-end; margin-bottom:24px; }
    h1 { font-size:32px; margin:0 0 8px; letter-spacing:0; }
    p { margin:0; color:var(--muted); }
    .plans { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:14px; margin-bottom:18px; }
    .plan { border:1px solid var(--line); background:var(--card); border-radius:10px; padding:18px; cursor:pointer; box-shadow:0 18px 50px rgba(15,23,42,.08); backdrop-filter:blur(14px); transition:.15s ease; }
    .plan:hover, .plan.active { border-color:rgba(14,165,233,.65); transform:translateY(-1px); }
    .plan h2 { font-size:18px; margin:0 0 8px; }
    .price { font-size:28px; font-weight:800; margin:14px 0 0; }
    .price small { font-size:14px; color:var(--muted); font-weight:500; }
    .panel { border:1px solid var(--line); background:var(--card); border-radius:10px; padding:18px; box-shadow:0 18px 50px rgba(15,23,42,.08); backdrop-filter:blur(14px); }
    .form { display:grid; grid-template-columns:1fr auto; gap:12px; align-items:end; }
    label { display:block; color:var(--muted); font-size:13px; margin-bottom:6px; }
    input { width:100%; height:44px; border:1px solid var(--line); border-radius:8px; background:rgba(255,255,255,.65); color:var(--text); padding:0 12px; font-size:15px; outline:none; }
    @media (prefers-color-scheme: dark) { input { background:rgba(15,23,42,.58); } }
    button { height:44px; border:0; border-radius:8px; background:var(--primary); color:white; padding:0 20px; font-weight:750; cursor:pointer; min-width:128px; }
    button:disabled { opacity:.58; cursor:default; }
    .msg { min-height:22px; margin-top:12px; color:var(--muted); font-size:14px; }
    .warn { color:var(--warn); }
    @media (max-width: 820px) { .hero { display:block; } .plans, .form { grid-template-columns:1fr; } button { width:100%; } }
  </style>
</head>
<body>
  <main class="wrap">
    <section class="hero">
      <div>
        <h1>购买 Qmby 激活码</h1>
        <p>填写邮箱并完成支付宝付款后，激活码会发送到你的邮箱并自动绑定。</p>
      </div>
    </section>
    <section id="plans" class="plans"></section>
    <section class="panel">
      <div class="form">
        <div>
          <label>接收激活码的邮箱</label>
          <input id="email" type="email" placeholder="user@example.com" autocomplete="email">
        </div>
        <button id="payButton" onclick="createOrder()">支付宝付款</button>
      </div>
      <div id="message" class="msg"></div>
    </section>
  </main>
  <script>
    let products = [];
    let selectedLevel = "";
    function $(id) { return document.getElementById(id); }
    function esc(value) { return String(value || "").replace(/[&<>"']/g, s => ({ "&":"&amp;", "<":"&lt;", ">":"&gt;", '"':"&quot;", "'":"&#39;" }[s])); }
    async function loadProducts() {
      const res = await fetch("/api/public/products");
      const data = await res.json();
      products = data.products || [];
      selectedLevel = products[1] ? products[1].level : (products[0] ? products[0].level : "");
      renderProducts();
    }
    function renderProducts() {
      $("plans").innerHTML = products.map(item =>
        "<article class=\"plan " + (item.level === selectedLevel ? "active" : "") + "\" onclick=\"selectPlan('" + esc(item.level) + "')\">" +
        "<h2>" + esc(item.label) + "</h2>" +
        "<p>" + esc(item.description) + "</p>" +
        "<div class=\"price\">¥" + esc(item.amount) + " <small>CNY</small></div>" +
        "</article>"
      ).join("");
    }
    function selectPlan(level) {
      selectedLevel = level;
      renderProducts();
    }
    async function createOrder() {
      $("message").textContent = "";
      $("message").className = "msg";
      $("payButton").disabled = true;
      try {
        const res = await fetch("/api/public/orders", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email: $("email").value, level: selectedLevel })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "创建订单失败");
        document.open();
        document.write(data.pay_form_html);
        document.close();
      } catch (err) {
        $("message").textContent = err.message;
        $("message").className = "msg warn";
        $("payButton").disabled = false;
      }
    }
    loadProducts().catch(err => { $("message").textContent = err.message; });
  </script>
</body>
</html>`

const payReturnHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>付款处理中</title>
  <style>
    body { margin:0; min-height:100vh; display:grid; place-items:center; font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; background:#f7fafc; color:#172033; }
    .card { width:min(420px,calc(100% - 32px)); border:1px solid #dce5ef; border-radius:10px; padding:24px; background:white; box-shadow:0 18px 50px rgba(15,23,42,.08); }
    h1 { margin:0 0 10px; font-size:22px; }
    p { margin:0; color:#66758a; line-height:1.7; }
    a { color:#0ea5e9; }
  </style>
</head>
<body>
  <section class="card">
    <h1>付款结果处理中</h1>
    <p>如果付款成功，系统会在收到支付宝通知后生成激活码并发送到你的邮箱。你可以稍后查看邮箱，包括垃圾邮件箱。</p>
    <p style="margin-top:12px"><a href="/buy">返回购买页</a></p>
  </section>
</body>
</html>`
