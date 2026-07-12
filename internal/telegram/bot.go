package telegram

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/config"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/db"
)

type subWizard struct {
	Step       string
	Site       string
	CategoryID int
	Label      string
}

type Bot struct {
	API       *tgbotapi.BotAPI
	Store     *db.Store
	Config    *config.Config
	AdminSet  map[int64]struct{}
	ChannelID int64
	mu        sync.Mutex
	wizards   map[int64]*subWizard
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
	return &Bot{API: api, Store: store, Config: cfg, AdminSet: admins, ChannelID: cfg.Telegram.PushChannelID, wizards: map[int64]*subWizard{}}, nil
}

func (b *Bot) IsAdmin(userID int64) bool {
	_, ok := b.AdminSet[userID]
	return ok
}

func mainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("/help"), tgbotapi.NewKeyboardButton("/site")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("/quota"), tgbotapi.NewKeyboardButton("/list")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("/sub"), tgbotapi.NewKeyboardButton("/admin")),
	)
	kb.ResizeKeyboard = true
	return kb
}

func (b *Bot) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	kb := mainKeyboard()
	msg.ReplyMarkup = kb
	_, _ = b.API.Send(msg)
}

func (b *Bot) SendToUser(userID int64, text string) { b.send(userID, text) }
func (b *Bot) SendToChannel(text string) {
	msg := tgbotapi.NewMessage(b.ChannelID, text)
	msg.ParseMode = "HTML"
	_, _ = b.API.Send(msg)
}

func (b *Bot) helpText() string {
	return "<b>forum-watch-bot 命令</b>\n" +
		"/start - 欢迎信息\n/help - 查看帮助\n/site - 查看当前支持站点\n/quota - 查看今日剩余推送额度\n/sub - 交互式添加订阅\n/sub site categoryID label keywordExpr - 直接添加订阅\n/list - 查看自己的订阅\n/del id - 删除订阅\n/admin - 查看管理员信息与推送频道"
}
func (b *Bot) siteText() string {
	return "<b>当前站点</b>\nlinuxdo -> https://linux.do/latest.rss\nnodeseek -> https://rss.nodeseek.com\nnodeloc -> https://www.nodeloc.com/latest.rss"
}
func (b *Bot) quotaText(userID int64) string {
	if b.IsAdmin(userID) { return "你是管理员，不受每日推送次数限制。" }
	n, err := b.Store.CountUserPushesToday(userID)
	if err != nil { return "查询额度失败: " + err.Error() }
	remain := b.Config.Runtime.DailyPushLimit - n
	if remain < 0 { remain = 0 }
	return fmt.Sprintf("今日已推送: %d\n每日上限: %d\n剩余次数: %d", n, b.Config.Runtime.DailyPushLimit, remain)
}
func (b *Bot) adminText(userID int64) string {
	base := fmt.Sprintf("推送频道: <code>%d</code>\n管理员数量: %d", b.ChannelID, len(b.AdminSet))
	if !b.IsAdmin(userID) { return base }
	var ids []string
	for id := range b.AdminSet { ids = append(ids, fmt.Sprintf("%d", id)) }
	return base + "\n管理员ID: " + strings.Join(ids, ", ")
}

func (b *Bot) startWizard(userID int64) { b.mu.Lock(); defer b.mu.Unlock(); b.wizards[userID] = &subWizard{Step: "site"} }
func (b *Bot) getWizard(userID int64) *subWizard { b.mu.Lock(); defer b.mu.Unlock(); return b.wizards[userID] }
func (b *Bot) clearWizard(userID int64) { b.mu.Lock(); defer b.mu.Unlock(); delete(b.wizards, userID) }

func (b *Bot) handleWizard(chatID, userID int64, text string) bool {
	w := b.getWizard(userID)
	if w == nil { return false }
	trimmed := strings.TrimSpace(text)
	if trimmed == "/cancel" || strings.EqualFold(trimmed, "cancel") || trimmed == "取消" {
		b.clearWizard(userID)
		b.send(chatID, "已取消当前订阅向导。")
		return true
	}
	switch w.Step {
	case "site":
		site := strings.TrimSpace(text)
		if site != "linuxdo" && site != "nodeseek" && site != "nodeloc" {
			b.send(chatID, "站点只能是：linuxdo / nodeseek / nodeloc\n或发送 /cancel 取消")
			return true
		}
		w.Site = site
		w.Step = "category"
		b.send(chatID, "请输入 categoryID，RSS 模式下通常填 0；仅作备注保留：")
		return true
	case "category":
		n, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil { b.send(chatID, "categoryID 必须是数字，请重新输入，或发送 /cancel 取消："); return true }
		w.CategoryID = n
		w.Step = "label"
		b.send(chatID, "请输入订阅标签：")
		return true
	case "label":
		label := strings.TrimSpace(text)
		if label == "" {
			b.send(chatID, "标签不能为空，请重新输入，或发送 /cancel 取消：")
			return true
		}
		w.Label = label
		w.Step = "keyword"
		b.send(chatID, "请输入关键词表达式：")
		return true
	case "keyword":
		kw := strings.TrimSpace(text)
		if kw == "" {
			b.send(chatID, "关键词不能为空，请重新输入，或发送 /cancel 取消：")
			return true
		}
		err := b.Store.AddSubscription(db.UserSubscription{UserID: userID, Site: w.Site, CategoryID: w.CategoryID, Label: w.Label, KeywordExpr: kw, Enabled: true})
		b.clearWizard(userID)
		if err != nil { b.send(chatID, "订阅失败: "+err.Error()); return true }
		b.send(chatID, "订阅已添加")
		return true
	}
	return false
}

func (b *Bot) HandleUpdates() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.API.GetUpdatesChan(u)
	for upd := range updates {
		if upd.Message == nil || upd.Message.From == nil { continue }
		chatID := upd.Message.Chat.ID
		userID := upd.Message.From.ID
		text := strings.TrimSpace(upd.Message.Text)
		if text == "" { continue }
		if !strings.HasPrefix(text, "/") {
			if b.handleWizard(chatID, userID, text) { continue }
		}
		switch {
		case strings.HasPrefix(text, "/start"):
			b.send(chatID, "欢迎使用论坛新帖监控机器人。\n\n"+b.helpText())
		case text == "/help":
			b.send(chatID, b.helpText())
		case text == "/site":
			b.send(chatID, b.siteText())
		case text == "/quota":
			b.send(chatID, b.quotaText(userID))
		case text == "/admin":
			b.send(chatID, b.adminText(userID))
		case text == "/sub":
			b.startWizard(userID)
			b.send(chatID, "开始添加订阅。\n请输入站点名：linuxdo / nodeseek / nodeloc")
		case strings.HasPrefix(text, "/sub "):
			parts := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(text, "/sub")), " ", 4)
			if len(parts) < 4 { b.send(chatID, "用法：/sub site categoryID label keywordExpr"); continue }
			categoryID, err := strconv.Atoi(parts[1])
			if err != nil { b.send(chatID, "categoryID 必须是数字"); continue }
			err = b.Store.AddSubscription(db.UserSubscription{UserID: userID, Site: parts[0], CategoryID: categoryID, Label: parts[2], KeywordExpr: parts[3], Enabled: true})
			if err != nil { b.send(chatID, "订阅失败: "+err.Error()); continue }
			b.send(chatID, "订阅已添加")
		case text == "/list":
			subs, err := b.Store.ListSubscriptions(userID)
			if err != nil { b.send(chatID, "查询失败: "+err.Error()); continue }
			if len(subs) == 0 { b.send(chatID, "你还没有订阅"); continue }
			var sb strings.Builder
			for _, s := range subs { sb.WriteString(fmt.Sprintf("ID:%d | %s | cat=%d | %s | %s\n", s.ID, s.Site, s.CategoryID, s.Label, s.KeywordExpr)) }
			b.send(chatID, sb.String())
		case strings.HasPrefix(text, "/del "):
			idStr := strings.TrimSpace(strings.TrimPrefix(text, "/del"))
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil { b.send(chatID, "id 不正确"); continue }
			if err := b.Store.DeleteSubscription(userID, id); err != nil { b.send(chatID, "删除失败: "+err.Error()); continue }
			b.send(chatID, "删除成功")
		default:
			b.send(chatID, "未知命令。发送 /help 查看帮助。")
		}
	}
	return nil
}
