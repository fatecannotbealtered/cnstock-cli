# cnstock-cli

[![CI](https://github.com/fatecannotbealtered/cnstock-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/fatecannotbealtered/cnstock-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fatecannotbealtered/cnstock-cli)](https://goreportcard.com/report/github.com/fatecannotbealtered/cnstock-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![npm version](https://img.shields.io/npm/v/@fatecannotbealtered-/cnstock-cli.svg)](https://www.npmjs.com/package/@fatecannotbealtered-/cnstock-cli)

[English](README.md) | 中文

通过腾讯财经网页端点查询 A 股、港股、美股、指数、基金的实时行情、历史 K 线、分时数据和股票搜索 — 全部在终端完成。

**本项目非腾讯官方产品，与腾讯控股有限公司无关联、无代言、无赞助关系。** 所有商标归其各自所有者。

Go 语言构建，单二进制文件，无需安装运行时。

[安装](#安装) · [命令](#命令) · [JSON 输出](#json-输出) · [安全](#安全) · [贡献](#贡献) · [免责声明](#免责声明)

## 免责声明

**本工具并非基于腾讯财经官方 API。**

此 CLI 使用的端点来自腾讯财经公开网页中观察到的网页端点，它们：

- **没有官方文档** — 腾讯未发布任何开发者文档
- **没有契约保障** — 无正式 SLA、Schema 契约或限频策略
- **没有稳定性保证** — 端点可能随时变更、返回不同数据或停止可用，且不会另行通知
- **不适用于生产环境** — 这些端点面向浏览器人工访问，而非程序化调用

本项目仅供**个人学习、研究和日常使用**。

**请勿将本工具用于：**

- 自动化交易或投资决策
- 商业产品或服务
- 合规敏感的金融报告
- 高频轮询或爬取
- 任何对数据准确性或可用性要求较高的场景

商业用途请使用持牌数据源（如万得、Tushare、AKShare、Bloomberg 等），它们提供正式的 API 文档、SLA 保障。

MIT 许可证仅覆盖本工具源代码。通过端点获取的行情数据仍归其各自权利方所有，本工具不授予对第三方数据的任何权利。用户须自行承担数据使用方式及合规责任。

完整的数据源声明参见 [SECURITY.md](SECURITY.md)。

## 功能特性

| 能力 | 说明 |
|---|---|
| 📈 **实时行情** | 批量查询 A 股/港股/美股，包含价格、涨跌幅、OHLCV、五档盘口 |
| 📊 **历史 K 线** | 日/周/月 K 线，支持前复权、后复权、不复权 |
| ⏱️ **分时数据** | 当日全部分钟级成交明细 |
| 🔍 **名称搜索** | 中文、拼音、英文均可搜索 |
| 🤖 **AI 友好** | `--json`、`--quiet`、`reference` 自描述命令 |
| ⚡ **单文件** | 下载即用，无运行时依赖 |
| 🌈 **美观输出** | 彩色表格，支持 CJK 字符宽度 |
| 🌐 **多市场** | A 股（沪/深/北）、港股、美股、指数、基金/ETF |

## 项目结构

```
cnstock-cli/
├── cmd/
│   ├── cnstock-cli/
│   │   └── main.go                # 入口
│   ├── root.go                    # 根命令 + 全局 flags (--json, --quiet)
│   ├── quote.go                   # quote 子命令
│   ├── kline.go                   # kline 子命令
│   ├── minute.go                  # minute 子命令
│   ├── search.go                  # search 子命令
│   ├── reference.go               # reference (AI 自发现)
│   └── cmd_test.go                # CLI smoke 测试
├── internal/
│   ├── api/
│   │   ├── client.go              # HTTP 客户端 + 端点解析
│   │   ├── symbol.go              # 代码规范化 + 市场检测
│   │   ├── quote.go               # 行情响应解析
│   │   ├── kline.go               # K 线响应解析
│   │   ├── minute.go              # 分时响应解析
│   │   ├── search.go              # 搜索响应解析
│   │   ├── encoding.go            # UTF-8 / GB18030 解码
│   │   ├── types.go               # 共享数据类型
│   │   ├── e2e_test.go            # API 级集成测试 (httptest)
│   │   ├── symbol_test.go
│   │   ├── quote_test.go
│   │   └── kline_test.go
│   └── output/
│       ├── output.go              # ANSI 颜色 + 终端检测
│       ├── json.go                # JSON 输出 + 错误码
│       └── table.go               # 表格输出 (CJK 宽度支持)
├── test/
│   └── e2e/
│       └── binary_test.go         # 二进制级 E2E 测试 (exec + httptest)
├── scripts/
│   ├── install.js                 # npm postinstall (二进制下载 + SHA256 校验)
│   └── run.js                     # npm bin wrapper
├── skills/
│   └── cnstock-cli/
│       ├── SKILL.md               # AI Agent 技能文件
│       └── symbols.json           # 常用股票代码速查表
├── .github/
│   └── workflows/
│       ├── ci.yml                 # CI: 3 OS × Go 1.23/1.24/1.25
│       └── release.yml            # Release: test → goreleaser → npm
├── .goreleaser.yml                # 跨平台构建配置
├── go.mod
├── Makefile
├── package.json                   # npm 包定义
├── README.md
├── README_zh.md
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
└── LICENSE                        # MIT
```

## 安装

### 快速开始

```bash
# 安装 CLI
npm install -g @fatecannotbealtered-/cnstock-cli

# 安装 Agent 技能
npx skills add fatecannotbealtered/cnstock-cli -y -g

# 第一条命令
cnstock-cli quote sh600519
```

### 其他方式: Go install

```bash
go install github.com/fatecannotbealtered/cnstock-cli/cmd/cnstock-cli@latest
```

### 其他方式: 下载二进制

从 [GitHub Releases](https://github.com/fatecannotbealtered/cnstock-cli/releases) 下载并添加到 PATH。

> **注意：** npm 安装依赖系统中的 `curl`（macOS 和大多数 Linux 发行版已预装）。Windows 或精简环境中如缺少 `curl`，请使用 `go install` 或直接下载二进制。

## 命令

### 实时行情

```bash
# 单只股票
cnstock-cli quote sh600519

# 批量查询（逗号分隔）
cnstock-cli quote 600519,hk00700,usAAPL

# JSON 输出
cnstock-cli quote sh600519 --json
```

自动识别市场：

- `600519`、`sh600519`、`sz000858` → A 股
- `00700`、`hk00700` → 港股
- `AAPL`、`usAAPL` → 美股
- `sh000001`、`sz399001` → 指数

### 历史 K 线

```bash
# 日 K 线（默认 20 根，前复权）
cnstock-cli kline sh600519

# 周 K 线，50 根，不复权
cnstock-cli kline sh600519 --period week --limit 50 --adj none
```

| 参数 | 默认値 | 说明 |
|------|--------|------|
| `--period` | `day` | `day`、`week`、`month` |
| `--limit` | `20` | 返回条数 (1-500) |
| `--adj` | `qfq` | `qfq`(前复权)、`hfq`(后复权)、`none`(不复权) |

### 分时数据

```bash
cnstock-cli minute sh600519
```

### 名称搜索

```bash
cnstock-cli search 茅台    # 中文
cnstock-cli search mt      # 拼音
cnstock-cli search apple   # 英文
```

### 自描述

```bash
cnstock-cli reference
```

输出所有命令、参数、JSON Schema 和错误码 — 专为 AI Agent 自发现设计。

## 全局参数

| 参数 | 说明 |
|------|------|
| `--json` | JSON 输出（机器可读） |
| `--quiet` | 抑制非 JSON 输出 |
| `--version` | 打印版本 |
| `--help` | 打印帮助 |

## JSON 输出

所有命令支持 `--json` 输出机器可读格式。示例：

```json
{
  "symbol": "sh600519",
  "market": "A股",
  "name": "贵州茅台",
  "price": 1800.00,
  "change": 15.50,
  "change_pct": 0.87
}
```

完整 Schema 请运行 `cnstock-cli reference`。

## 错误码

| 错误码 | 退出码 | 含义 |
|--------|--------|------|
| `VALIDATION_ERROR` | 2 | 参数无效或缺少必填参数 |
| `NOT_FOUND` | 4 | 代码或资源未找到 |
| `SERVER_ERROR` | 7 | 上游服务器返回错误 |
| `NETWORK_ERROR` | 7 | 网络连接或 HTTP 传输失败 |
| `UNKNOWN_ERROR` | 1 | 未预期错误（如上游响应格式异常） |

## 环境变量

正常无需配置。以下环境变量可用于高级场景（测试、代理、自托管端点）：

| 变量 | 默认値 | 说明 |
|------|--------|------|
| `CNS_QUOTE_ENDPOINT` | `https://qt.gtimg.cn/q=%s` | 行情端点（须含 `%s`） |
| `CNS_KLINE_ENDPOINT` | `https://web.ifzq.gtimg.cn/appstock/app/%s/get?param=%s` | K 线端点 |
| `CNS_MINUTE_ENDPOINT` | `https://web.ifzq.gtimg.cn/appstock/app/minute/query?code=%s` | 分时端点 |
| `CNS_SEARCH_ENDPOINT` | `https://smartbox.gtimg.cn/s3/?v=2&q=%s&t=all&c=1` | 搜索端点 |

## 安全

参见 [SECURITY.md](SECURITY.md)。

## 贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 许可证

[MIT](LICENSE)
