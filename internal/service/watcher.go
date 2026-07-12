package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xiaoxinkeji/forum-watch-bot/internal/config"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/db"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/model"
	tg "github.com/xiaoxinkeji/forum-watch-bot/internal/telegram"
)

type Watcher struct {
	Store    *db.Store
	Bot      *tg.Bot
	Config   *config.Config
	Fetchers []model.SiteTopicFetcher
}

func (w *Watcher) RunOnce() error {
	allSubs, err := w.Store.ListSubscriptions(0)
	if err != nil {
		return err
	}
	for _, fetcher := range w.Fetchers {
		topics, err := fetcher.FetchLatest()
		if err != nil {
			continue
		}
		for _, topic := range topics {
			seen, err := w.Store.IsSeen(string(topic.Site), topic.ExternalID)
			if err == nil && seen {
				continue
			}
			for _, sub := range allSubs {
				if !sub.Enabled || sub.Site != string(topic.Site) {
					continue
				}
				ok, matchText := matches(topic, sub.KeywordExpr)
				if !ok {
					continue
				}
				count, err := w.Store.CountUserPushesToday(sub.UserID)
				if err == nil && count >= w.Config.Runtime.DailyPushLimit && !w.BotIsAdmin(sub.UserID) {
					continue
				}
				msg := fmt.Sprintf("<b>%s</b>\n站点: %s\n订阅标签: %s\n匹配: %s\n%s", topic.Title, topic.Site, sub.Label, matchText, topic.URL)
				w.Bot.SendToUser(sub.UserID, msg)
				_ = w.Store.AddPushLog(sub.UserID, topic.URL)
			}
			channelMsg := fmt.Sprintf("<b>%s</b>\n站点: %s\n%s", topic.Title, topic.Site, topic.URL)
			w.Bot.SendToChannel(channelMsg)
			_ = w.Store.MarkSeen(string(topic.Site), topic.ExternalID, topic.Title, topic.URL, topic.PublishedAt)
		}
	}
	return nil
}

func (w *Watcher) BotIsAdmin(userID int64) bool {
	return w.Bot.IsAdmin(userID)
}

func matches(topic model.Topic, expr string) (bool, string) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, ""
	}
	title := strings.ToLower(topic.Title)
	content := strings.ToLower(topic.Title + " " + topic.Excerpt)
	desc := strings.ToLower(topic.Excerpt)
	items := strings.Split(expr, ",")
	var positive []string
	for _, raw := range items {
		kw := strings.TrimSpace(raw)
		if kw == "" {
			continue
		}
		isBlock := strings.HasPrefix(kw, "-")
		if isBlock {
			kw = strings.TrimSpace(strings.TrimPrefix(kw, "-"))
		}
		scope := "title"
		if strings.HasPrefix(kw, "#t") { kw = strings.TrimSpace(strings.TrimPrefix(kw, "#t")); scope = "title" }
		if strings.HasPrefix(kw, "#c") { kw = strings.TrimSpace(strings.TrimPrefix(kw, "#c")); scope = "content" }
		if strings.HasPrefix(kw, "#a") { kw = strings.TrimSpace(strings.TrimPrefix(kw, "#a")); scope = "all" }
		target := title
		if scope == "content" { target = desc }
		if scope == "all" { target = content }
		matched := false
		if strings.Contains(kw, "*") {
			pattern := ".*" + regexp.QuoteMeta(strings.ToLower(kw)) + ".*"
			pattern = strings.ReplaceAll(pattern, "\\*", ".*")
			re, err := regexp.Compile(pattern)
			matched = err == nil && re.MatchString(target)
		} else {
			matched = strings.Contains(target, strings.ToLower(kw))
		}
		if isBlock && matched {
			return false, "blocked:" + kw
		}
		if !isBlock && matched {
			positive = append(positive, kw)
		}
	}
	if len(positive) == 0 {
		return false, ""
	}
	return true, strings.Join(positive, ",")
}

func Loop(w *Watcher) {
	interval := time.Duration(w.Config.Runtime.PollIntervalSeconds) * time.Second
	for {
		_ = w.RunOnce()
		time.Sleep(interval)
	}
}
