package web

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiaoxinkeji/forum-watch-bot/internal/config"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/db"
)

type Server struct {
	Store  *db.Store
	Config *config.Config
}

type pageData struct { Title string; Body template.HTML }

func New(store *db.Store, cfg *config.Config) *Server { return &Server{Store: store, Config: cfg} }

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != s.Config.Web.Username || p != s.Config.Web.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="forum-watch-bot"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func render(w http.ResponseWriter, title string, body template.HTML) {
	tpl := `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title><style>:root{--bg:#0b1220;--panel:#121a2b;--muted:#94a3b8;--text:#e5e7eb;--line:#243041;--accent:#3b82f6;--good:#10b981}*{box-sizing:border-box}body{font-family:Inter,Arial,sans-serif;max-width:1100px;margin:0 auto;padding:24px 16px;background:var(--bg);color:var(--text)}table{border-collapse:collapse;width:100%;background:var(--panel);border-radius:12px;overflow:hidden}th,td{border-bottom:1px solid var(--line);padding:10px;text-align:left}th{background:#182235}input,textarea,select{padding:10px;margin:4px 0 12px;width:100%;background:#0f172a;color:var(--text);border:1px solid var(--line);border-radius:10px}a,button{padding:8px 12px;border-radius:10px;border:0;background:var(--accent);color:white;text-decoration:none;display:inline-block;cursor:pointer}nav{display:flex;gap:10px;flex-wrap:wrap;margin-bottom:18px}.card{padding:14px;border:1px solid var(--line);background:var(--panel);margin:12px 0;border-radius:14px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px}.muted{color:var(--muted)}.ok{color:var(--good)}</style></head><body><nav><a href="/">首页</a><a href="/subscriptions">订阅</a><a href="/subscriptions/new">新增订阅</a><a href="/credentials">站点登录</a><a href="/test-push">测试推送</a></nav><h1>{{.Title}}</h1>{{.Body}}</body></html>`
	t := template.Must(template.New("page").Parse(tpl))
	_ = t.Execute(w, pageData{Title: title, Body: body})
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.auth(s.handleIndex))
	mux.HandleFunc("/subscriptions", s.auth(s.handleSubscriptions))
	mux.HandleFunc("/subscriptions/new", s.auth(s.handleNewSubscription))
	mux.HandleFunc("/subscriptions/delete", s.auth(s.handleDeleteSubscription))
	mux.HandleFunc("/credentials", s.auth(s.handleCredentials))
	mux.HandleFunc("/test-push", s.auth(s.handleTestPush))
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	subs, _ := s.Store.ListSubscriptions(0)
	stats, _ := s.Store.GetDashboardStats()
	body := template.HTML(fmt.Sprintf(`<div class="card"><b>订阅数:</b> %d</div><div class="card"><b>已记录主题:</b> %d</div><div class="card"><b>今日推送:</b> %d</div><div class="card"><b>推送频道:</b> %d</div><div class="card"><b>每日限额:</b> %d</div><div class="card"><b>支持站点:</b><ul><li>linuxdo</li><li>nodeseek</li><li>nodeloc</li></ul></div>`, len(subs), stats.SeenTopicCount, stats.TodayPushCount, s.Config.Telegram.PushChannelID, s.Config.Runtime.DailyPushLimit))
	render(w, "forum-watch-bot 后台", body)
}

func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs, _ := s.Store.ListSubscriptions(0)
	var sb strings.Builder
	sb.WriteString(`<table><tr><th>ID</th><th>UserID</th><th>Site</th><th>CategoryID</th><th>Label</th><th>Keyword</th><th>Enabled</th><th>Action</th></tr>`)
	for _, sub := range subs { sb.WriteString(fmt.Sprintf(`<tr><td>%d</td><td>%d</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>%t</td><td><a href="/subscriptions/delete?id=%d&user_id=%d">删除</a></td></tr>`, sub.ID, sub.UserID, sub.Site, sub.CategoryID, template.HTMLEscapeString(sub.Label), template.HTMLEscapeString(sub.KeywordExpr), sub.Enabled, sub.ID, sub.UserID)) }
	sb.WriteString(`</table>`)
	render(w, "订阅列表", template.HTML(sb.String()))
}

func (s *Server) handleNewSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		body := template.HTML(`<form method="post"><label>User ID</label><input name="user_id"><label>Site</label><select name="site"><option>linuxdo</option><option>nodeseek</option><option>nodeloc</option></select><label>Category ID</label><input name="category_id" value="0"><label>Label</label><input name="label"><label>KeywordExpr</label><input name="keyword_expr"><button type="submit">保存</button></form>`)
		render(w, "新增订阅", body); return }
	_ = r.ParseForm(); userID, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64); categoryID, _ := strconv.Atoi(r.FormValue("category_id"))
	err := s.Store.AddSubscription(db.UserSubscription{UserID: userID, Site: r.FormValue("site"), CategoryID: categoryID, Label: r.FormValue("label"), KeywordExpr: r.FormValue("keyword_expr"), Enabled: true})
	if err != nil { render(w, "新增订阅失败", template.HTML(template.HTMLEscapeString(err.Error()))); return }
	http.Redirect(w, r, "/subscriptions", http.StatusFound)
}

func (s *Server) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64); userID, _ := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	_ = s.Store.DeleteSubscription(userID, id)
	http.Redirect(w, r, "/subscriptions", http.StatusFound)
}

func (s *Server) handleCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		site := r.FormValue("site")
		cookie := r.FormValue("cookie")
		headersJSON := r.FormValue("headers_json")
		if headersJSON == "" { headersJSON = "{}" }
		if err := s.Store.UpsertSiteCredential(site, cookie, headersJSON); err != nil {
			render(w, "保存登录态失败", template.HTML(template.HTMLEscapeString(err.Error())))
			return
		}
		http.Redirect(w, r, "/credentials", http.StatusFound)
		return
	}
	sites := []string{"linuxdo","nodeseek","nodeloc"}
	var sb strings.Builder
	sb.WriteString(`<div class="card"><b>说明：</b><span class="muted">在这里填站点 Cookie / Headers JSON，RSS 请求会自动携带。</span></div>`)
	for _, site := range sites {
		cred, _ := s.Store.GetSiteCredential(site)
		status := "未配置"
		if strings.TrimSpace(cred.Cookie) != "" || strings.TrimSpace(cred.HeadersJSON) != "{}" { status = "已配置" }
		sb.WriteString(fmt.Sprintf(`<div class="card"><h3>%s <span class="ok">%s</span></h3><form method="post"><input type="hidden" name="site" value="%s"><label>Cookie</label><textarea name="cookie" rows="5">%s</textarea><label>Headers JSON</label><textarea name="headers_json" rows="5">%s</textarea><button type="submit">保存</button></form></div>`, site, status, site, template.HTMLEscapeString(cred.Cookie), template.HTMLEscapeString(cred.HeadersJSON)))
	}
	render(w, "站点登录配置", template.HTML(sb.String()))
}

func (s *Server) handleTestPush(w http.ResponseWriter, r *http.Request) {
	body := template.HTML(`<form method="post"><label>测试消息</label><textarea name="message" rows="5">forum-watch-bot 测试推送</textarea><button type="submit">发送到频道</button></form>`)
	if r.Method == http.MethodGet { render(w, "测试推送", body); return }
	_ = r.ParseForm()
	msg := r.FormValue("message")
	if msg == "" { msg = "forum-watch-bot 测试推送" }
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.Config.Telegram.BotToken)
	resp, err := http.PostForm(url, map[string][]string{"chat_id": {fmt.Sprintf("%d", s.Config.Telegram.PushChannelID)}, "text": {msg}, "parse_mode": {"HTML"}})
	if err != nil { render(w, "测试推送失败", template.HTML(template.HTMLEscapeString(err.Error()))); return }
	defer resp.Body.Close()
	if resp.StatusCode >= 300 { render(w, "测试推送失败", template.HTML(fmt.Sprintf("telegram status: %d", resp.StatusCode))); return }
	render(w, "测试推送", template.HTML(`<div class="card ok">测试推送已发送，请检查频道。</div>`+string(body)))
}
