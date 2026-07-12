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

type pageData struct {
	Title string
	Body  template.HTML
}

func New(store *db.Store, cfg *config.Config) *Server {
	return &Server{Store: store, Config: cfg}
}

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
	tpl := `<!doctype html><html><head><meta charset="utf-8"><title>{{.Title}}</title><style>body{font-family:Arial,sans-serif;max-width:1000px;margin:40px auto;padding:0 16px}table{border-collapse:collapse;width:100%}th,td{border:1px solid #ddd;padding:8px}input,select{padding:8px;margin:4px 0;width:100%}a,button{padding:6px 10px}nav a{margin-right:12px}.card{padding:12px;border:1px solid #ddd;margin:12px 0}</style></head><body><nav><a href="/">首页</a><a href="/subscriptions">订阅</a><a href="/subscriptions/new">新增订阅</a></nav><h1>{{.Title}}</h1>{{.Body}}</body></html>`
	t := template.Must(template.New("page").Parse(tpl))
	_ = t.Execute(w, pageData{Title: title, Body: body})
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.auth(s.handleIndex))
	mux.HandleFunc("/subscriptions", s.auth(s.handleSubscriptions))
	mux.HandleFunc("/subscriptions/new", s.auth(s.handleNewSubscription))
	mux.HandleFunc("/subscriptions/delete", s.auth(s.handleDeleteSubscription))
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
	for _, sub := range subs {
		sb.WriteString(fmt.Sprintf(`<tr><td>%d</td><td>%d</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>%t</td><td><a href="/subscriptions/delete?id=%d&user_id=%d">删除</a></td></tr>`, sub.ID, sub.UserID, sub.Site, sub.CategoryID, template.HTMLEscapeString(sub.Label), template.HTMLEscapeString(sub.KeywordExpr), sub.Enabled, sub.ID, sub.UserID))
	}
	sb.WriteString(`</table>`)
	render(w, "订阅列表", template.HTML(sb.String()))
}

func (s *Server) handleNewSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		body := template.HTML(`<form method="post"><label>User ID</label><input name="user_id"><label>Site</label><select name="site"><option>linuxdo</option><option>nodeseek</option><option>nodeloc</option></select><label>Category ID</label><input name="category_id" value="0"><label>Label</label><input name="label"><label>KeywordExpr</label><input name="keyword_expr"><button type="submit">保存</button></form>`)
		render(w, "新增订阅", body)
		return
	}
	_ = r.ParseForm()
	userID, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	categoryID, _ := strconv.Atoi(r.FormValue("category_id"))
	err := s.Store.AddSubscription(db.UserSubscription{UserID: userID, Site: r.FormValue("site"), CategoryID: categoryID, Label: r.FormValue("label"), KeywordExpr: r.FormValue("keyword_expr"), Enabled: true})
	if err != nil {
		render(w, "新增订阅失败", template.HTML(template.HTMLEscapeString(err.Error())))
		return
	}
	http.Redirect(w, r, "/subscriptions", http.StatusFound)
}

func (s *Server) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	userID, _ := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	_ = s.Store.DeleteSubscription(userID, id)
	http.Redirect(w, r, "/subscriptions", http.StatusFound)
}
