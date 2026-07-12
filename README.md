# forum-watch-bot

一个使用 Go 编写的 Telegram 论坛新帖监控机器人。

## 当前已完成
- NodeLoc 新帖监控（可用）
- Telegram 用户自助订阅
- 支持站点 + 分类ID + 标签 + 关键词表达式订阅
- 普通用户每日推送次数限制
- 推送到固定 Telegram 频道
- SQLite 本地存储
- Linux.do / NodeSeek 适配器接口预留

## 为什么 Linux.do / NodeSeek 现在没有默认启用
当前构建环境下，这两个站点都被 Cloudflare `403 Just a moment...` 挡住，无法做出“已验证可用”的默认抓取实现。
因此本版本：
- **NodeLoc 可直接工作**
- **Linux.do / NodeSeek 提供适配器占位与配置说明**
- 后续只要补 cookie / 代理 / 可用 JSON 源即可接上

## 订阅命令
```bash
/sub site categoryID label keywordExpr
/list
/del id
```

示例：
```bash
/sub nodeloc 6 VPS优惠 #t香港,9929,-已出
```

## 关键词语法
参考了 `IonRh/TGBot_RSS` 项目的关键词思路，并按本项目需求裁剪：
- 英文逗号分隔多个关键词
- `-关键词` 表示屏蔽词
- `*` 通配任意字符
- `#t` 只匹配标题
- `#c` 只匹配摘要/内容
- `#a` 匹配标题和内容

## 配置
复制 `config.example.json` 为 `config.json` 并填写：
- Telegram bot token
- 推送频道 ID
- 管理员用户 ID

## 运行
```bash
./forum-watch-bot ./config.json
```

## 构建
```bash
go build -ldflags "-X main.version=v0.1.0 -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%FT%TZ)" -o forum-watch-bot ./cmd/forum-watch-bot
```

## 说明
本项目为定制实现，功能设计参考了：
- https://github.com/IonRh/TGBot_RSS

该项目许可证：Boost Software License 1.0。
本仓库未原样复制其代码结构作为最终成品说明页，而是基于你的需求做了论坛新帖监控的独立实现与裁剪。
