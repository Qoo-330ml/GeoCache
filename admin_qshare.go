package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveAdminQshare(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderAdminPage(adminPage{
		Title:         "Qshare - Qmby License Admin",
		BrandTitle:    "Qshare Admin",
		BrandSubtitle: "共享内容管理",
		Heading:       "Qshare 管理",
		Description:   "查看、编辑、下架或批量删除云端共享内容。",
		Actions: `
        <button class="secondary" onclick="location.href='/admin'">返回概览</button>
        <button class="secondary" onclick="loadQshareResources()">刷新</button>
        <button class="secondary" onclick="logout()">退出登录</button>`,
		Body: `
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
    </div>`,
		Script: `
    let qshareTimer = 0;
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
    async function afterLogin() { await loadQshareResources(); }`,
		LoginProbe: "/api/admin/qshare/resources?limit=1",
	}))
}
