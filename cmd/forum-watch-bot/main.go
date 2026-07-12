package main

import (
	"log"
	"net/http"
	"os"

	"github.com/xiaoxinkeji/forum-watch-bot/internal/config"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/db"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/model"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/service"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/sites"
	tg "github.com/xiaoxinkeji/forum-watch-bot/internal/telegram"
	webui "github.com/xiaoxinkeji/forum-watch-bot/internal/web"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	cfgPath := "config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	store, err := db.Open(cfg.Runtime.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	bot, err := tg.New(cfg.Telegram.BotToken, store, cfg)
	if err != nil {
		log.Fatal(err)
	}
	client, err := sites.NewHTTPClient(cfg.Runtime.ProxyURL)
	if err != nil {
		log.Fatal(err)
	}
	linuxRSS := "https://linux.do/latest.rss"
	nodeSeekRSS := "https://rss.nodeseek.com"
	nodeLocRSS := "https://www.nodeloc.com/latest.rss"
	for _, s := range cfg.Sites {
		if s.Options == nil {
			continue
		}
		switch s.ID {
		case "linuxdo":
			if s.Options["rss_url"] != "" { linuxRSS = s.Options["rss_url"] }
		case "nodeseek":
			if s.Options["rss_url"] != "" { nodeSeekRSS = s.Options["rss_url"] }
		case "nodeloc":
			if s.Options["rss_url"] != "" { nodeLocRSS = s.Options["rss_url"] }
		}
	}
	fetchers := []model.SiteTopicFetcher{
		sites.NewGenericRSSFetcher(client, model.SiteLinuxDO, linuxRSS),
		sites.NewGenericRSSFetcher(client, model.SiteNodeSeek, nodeSeekRSS),
		sites.NewGenericRSSFetcher(client, model.SiteNodeLoc, nodeLocRSS),
	}
	watcher := &service.Watcher{Store: store, Bot: bot, Config: cfg, Fetchers: fetchers}
	go service.Loop(watcher)
	if cfg.Web.Enabled {
		go func() {
			websrv := webui.New(store, cfg)
			log.Printf("web ui listening on %s", cfg.Web.Listen)
			if err := http.ListenAndServe(cfg.Web.Listen, websrv.Routes()); err != nil {
				log.Printf("web ui stopped: %v", err)
			}
		}()
	}
	log.Printf("forum-watch-bot version=%s commit=%s buildTime=%s started", version, commit, buildTime)
	if err := bot.HandleUpdates(); err != nil {
		log.Fatal(err)
	}
}
