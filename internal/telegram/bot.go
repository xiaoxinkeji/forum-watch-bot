package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/config"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/db"
)

type Bot struct {
	API       *tgbotapi.BotAPI
	Store     *db.Store
	Config    *config.Config
	AdminSet  map[int64]struct{}
	ChannelID int64
}

func New(token string, store *db.Store, cfg *config.Config) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	admins := map[int64]struct{}{}
	for _, id := range cfg.Admin.AdminUserIDs {
		admins[id] = struct{}{}
	}
	return &Bot{API: api, Store: store, Config: cfg, AdminSet: admins, ChannelID: cfg.Telegram.PushChannelID}, nil
}

func (b *Bot) IsAdmin(userID int64) bool {
	_, ok := b.AdminSet[userID]
	return ok
}

func (b *Bot) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	_, _ = b.API.Send(msg)
}

func (b *Bot) SendToUser(userID int64, text string) {
	b.send(userID, text)
}

func (b *Bot) SendToChannel(text string) {
	b.send(b.ChannelID, text)
}

func (b *Bot) HandleUpdates() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.API.GetUpdatesChan(u)
	for upd := range updates {
		if upd.Message == nil || upd.Message.From == nil {
			continue
		}
		chatID := upd.Message.Chat.ID
		userID := upd.Message.From.ID
		text := strings.TrimSpace(upd.Message.Text)
		if text == "" {
			continue
		}
		switch {
		case strings.HasPrefix(text, "/start"):
			b.send(chatID, "欢迎使用论坛新帖监控机器人。\n命令：\n/sub site categoryID label keywordExpr\n/list\n/del id")
		case strings.HasPrefix(text, "/sub "):
			parts := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(text, "/sub")), " ", 4)
			if len(parts) < 4 {
				b.send(chatID, "用法：/sub site categoryID label keywordExpr")
				continue
			}
			categoryID, err := strconv.Atoi(parts[1])
			if err != nil {
				b.send(chatID, "categoryID 必须是数字")
				continue
			}
			err = b.Store.AddSubscription(db.UserSubscription{UserID: userID, Site: parts[0], CategoryID: categoryID, Label: parts[2], KeywordExpr: parts[3], Enabled: true})
			if err != nil {
				b.send(chatID, "订阅失败: "+err.Error())
				continue
			}
			b.send(chatID, "订阅已添加")
		case text == "/list":
			subs, err := b.Store.ListSubscriptions(userID)
			if err != nil {
				b.send(chatID, "查询失败: "+err.Error())
				continue
			}
			if len(subs) == 0 {
				b.send(chatID, "你还没有订阅")
				continue
			}
			var sb strings.Builder
			for _, s := range subs {
				sb.WriteString(fmt.Sprintf("ID:%d | %s | cat=%d | %s | %s\n", s.ID, s.Site, s.CategoryID, s.Label, s.KeywordExpr))
			}
			b.send(chatID, sb.String())
		case strings.HasPrefix(text, "/del "):
			idStr := strings.TrimSpace(strings.TrimPrefix(text, "/del"))
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				b.send(chatID, "id 不正确")
				continue
			}
			if err := b.Store.DeleteSubscription(userID, id); err != nil {
				b.send(chatID, "删除失败: "+err.Error())
				continue
			}
			b.send(chatID, "删除成功")
		default:
			b.send(chatID, "未知命令。可用：/sub /list /del")
		}
	}
	return nil
}
