# forum-watch-bot

一个使用 Go 编写的 Telegram 论坛新帖监控机器人，支持三站 RSS 监控、Telegram 订阅和内置 Web 管理后台。

## 当前已完成
- Linux.do RSS 监控（可用）
- NodeSeek 官方 RSS 监控（可用）
- NodeLoc RSS 监控（可用）
- Telegram 用户自助订阅
- 交互式订阅向导（/sub 后逐步录入）
- 支持站点 + 分类ID(备注) + 标签 + 关键词表达式订阅
- 普通用户每日推送次数限制
- 推送到固定 Telegram 频道
- SQLite 本地存储
- 代理支持（HTTP / SOCKS5）
- 交互式 TG 主菜单按钮
- Dockerfile
- systemd service 示例
- 内置 Web 管理后台（Basic Auth）

## 当前实测可用 RSS 源
- `https://linux.do/latest.rss`
- `https://rss.nodeseek.com`
- `https://www.nodeloc.com/latest.rss`

## Telegram 命令
```bash
/start
/help
/site
/quota
/admin
/sub
/sub site categoryID label keywordExpr
/list
/del id
/cancel
```

## Web 后台
启用配置：
```json
"web": {
  "enabled": true,
  "listen": ":8080",
  "username": "admin",
  "password": "change-me"
}
```

功能：
- 首页状态面板
- 全部订阅列表
- Web 新增订阅
- Web 删除订阅
- Basic Auth 保护
- 站点登录配置页（保存 Cookie / Headers JSON）
- Web 测试推送页面

访问：
```bash
http://服务器IP:8080
```

站点登录态说明：
- 在 `/credentials` 页面为 `linuxdo` / `nodeseek` / `nodeloc` 保存 Cookie
- 如需要自定义请求头，可填写 `Headers JSON`
- RSS 请求会自动携带已保存的 Cookie/Header

## 交互式订阅
直接发送：
```bash
/sub
```
机器人会依次询问：
1. site
2. categoryID（RSS 模式下仅作备注，通常填 0）
3. label
4. keywordExpr

任意步骤可发送：
```bash
/cancel
```
取消当前向导。

## 直接订阅示例
```bash
/sub linuxdo 0 社区热帖 #t甲骨文,ARM,-已出
/sub nodeseek 0 NS优惠 #t杜甫,9929,-已出
/sub nodeloc 0 NL热帖 #t香港,9929,-已出
```

## Docker
```bash
docker build \
  --build-arg ALL_PROXY=socks5://127.0.0.1:20170 \
  --build-arg HTTP_PROXY=socks5://127.0.0.1:20170 \
  --build-arg HTTPS_PROXY=socks5://127.0.0.1:20170 \
  -t forum-watch-bot .

docker run -d --name forum-watch-bot \
  -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json \
  -v $(pwd)/data:/app \
  forum-watch-bot
```

## systemd
示例文件：
- `forum-watch-bot.service`

部署示例：
```bash
cp forum-watch-bot /opt/forum-watch-bot/
cp config.json /opt/forum-watch-bot/
cp forum-watch-bot.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now forum-watch-bot
```

## 构建
```bash
go build -ldflags "-X main.version=v0.9.0 -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%FT%TZ)" -o forum-watch-bot ./cmd/forum-watch-bot
```

## 说明
本项目为定制实现，功能设计参考了：
- https://github.com/IonRh/TGBot_RSS

该项目许可证：Boost Software License 1.0。
本版本已包含可用的最小 Web 后台，并保留 TG 与 Docker/systemd 部署链。
