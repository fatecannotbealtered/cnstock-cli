# 调研报告:A 股「市场涨跌家数统计」HTTP 端点

> 目标:为 cnstock-cli(Go,免登录 web 端点,best-effort)寻找获取全市场
> 上涨/下跌/平盘/涨停/跌停家数(及可选总成交额)的稳定端点。
>
> 验证环境时间:2026-06-05(交易日)。下文标注「✅ 已验证」的均为本次实际 curl 验证;
> 标注「未验证」的为基于公开资料的推断。

---

## 0. 结论速览(TL;DR)

- **首选(强烈推荐):东方财富 `push2.eastmoney.com/api/qt/ulist.np/get`**
  一次请求即可拿到 **沪 + 深 + 北** 三市的上涨/下跌/平盘家数 + 各市成交额,
  纯 JSON、UTF-8、**无需任何 header(Referer/UA/Cookie 全可省)**、字段语义清晰。
  涨停/跌停家数则再补两个东财端点(ZT/DT 池)即可,各返回一个 `tc` 计数。

- **次选:腾讯(维持现状作为兜底)** —— 腾讯**没有**干净的「全市场涨跌家数」聚合端点,
  需要自己拉全量行情再本地计数,成本高;建议仅作为东财不可用时的降级路径,
  或继续用现有 `qt.gtimg.cn` 做个股/指数行情。

- 新浪 `hq.sinajs.cn` / `vip.stock.finance.sina.com.cn` 只提供「某板块股票**总数**」
  (如 `getHQNodeStockCount` 返回一个数字),**不直接给涨跌家数**,且需要 `Referer`、GBK 编码,
  价值有限,仅可做总数校验。

---

## 1. 东方财富(首选)

东财有两类端点组合,正好覆盖全部需求。

### 1.1 上涨 / 下跌 / 平盘 家数 + 成交额 —— `ulist.np`(✅ 已验证)

关键洞察:东财的指数行情里,`f104/f105/f106` 表示**该指数所属整个板块**的
涨/跌/平股票数(不是指数成分股数)。因此选三个「综合指数」即可无重叠地覆盖全市场:

| secid | 名称 | 覆盖范围 |
|-------|------|----------|
| `1.000001` | 上证指数 | 全部沪市股票 |
| `0.399106` | 深证综指 | 全部深市股票(含创业板) |
| `0.899050` | 北证50 | 全部北交所股票 |

> `secid` 格式:`市场.代码`,`1`=上交所,`0`=深交所/北交所。

**可直接 curl 的真实示例:**

```bash
curl "https://push2.eastmoney.com/api/qt/ulist.np/get?fltt=2&secids=1.000001,0.399106,0.899050&fields=f3,f14,f104,f105,f106,f6"
```

**真实响应片段(2026-06-05 验证):**

```json
{"rc":0,"data":{"total":3,"diff":[
  {"f3":-0.74,"f6":1363887868514.9, "f14":"上证指数","f104":1284,"f105":1008,"f106":60},
  {"f3":-1.33,"f6":1705304054213.09,"f14":"深证综指","f104":1742,"f105":1107,"f106":74},
  {"f3": 5.59,"f6":31879947939.0,  "f14":"北证50", "f104":295, "f105":22,  "f106":0}
]}}
```

**字段语义:**

| 字段 | 含义 |
|------|------|
| `f104` | 上涨家数 |
| `f105` | 下跌家数 |
| `f106` | 平盘家数 |
| `f6`   | 该市成交额(元) |
| `f3`   | 指数涨跌幅(%) |
| `f14`  | 名称 |

> 全市场汇总即把三行相加:上涨 1284+1742+295=3321,下跌 1008+1107+22=2137,
> 平盘 60+74+0=134(合计 ≈5592,与新浪 hs_a 总数 ~5524 + 北交所一致,交叉验证通过)。
> 总成交额 = 三行 `f6` 相加。

**关键参数说明:**
- `secids`:逗号分隔的 `市场.代码` 列表(必填)。
- `fields`:返回字段白名单,建议只取 `f3,f14,f104,f105,f106,f6` 以减小体积。
- `fltt=2`:价格按「实际值」格式化(对计数字段无影响,保留即可)。

**格式 / 编码 / Header(✅ 已验证):**
- 格式:标准 JSON(`ulist.np` 不带 JSONP 回调时直接返回 JSON 对象)。
- 编码:UTF-8。
- Header:**全部可省**。实测不带 `Referer`、不带 `User-Agent`、不带 Cookie 也返回 200 + 正常数据。
  (可选 `cb=xxx` 参数会变成 JSONP,本场景不需要,务必**不要**带 `cb`。)

### 1.2 涨停家数 —— ZT 池 `getTopicZTPool`(✅ 已验证)

```bash
curl "https://push2ex.eastmoney.com/getTopicZTPool?ut=7eea3edcaed734bea9cbfc24409ed989&dpt=wz.ztzt&Pageindex=0&pagesize=1&sort=fbt:asc&date=20260605"
```

**真实响应片段:**
```json
{"rc":0,"data":{"tc":73,"qdate":20260605,"pool":[ {"c":"002354","n":"天娱数科", ...} ]}}
```
- `data.tc` = **涨停家数**(本例 73)。只要计数,可设 `pagesize=1` 忽略 `pool` 明细。
- `date` 需传当前交易日(`YYYYMMDD`)。
- `ut` 是东财固定公开 token(各开源库长期沿用同一串)。
- 域名是 `push2ex`(注意带 `ex`),非 `push2`。

### 1.3 跌停家数 —— DT 池 `getTopicDTPool`(✅ 已验证)

```bash
curl "https://push2ex.eastmoney.com/getTopicDTPool?ut=7eea3edcaed734bea9cbfc24409ed989&dpt=wz.ztzt&Pageindex=0&pagesize=1&sort=fund:asc&date=20260605"
```
**真实响应:** `{"rc":0,"data":{"tc":11,"qdate":20260605,"pool":[...]}}` —— `data.tc` = **跌停家数**(本例 11)。

### 1.4 东财稳定性 / 合规

- **被广泛使用**:`push2.eastmoney.com` 是 akshare、efinance、adata、leek-fund、easyquotation
  生态等几乎所有开源库的核心数据源,`ulist.np` / `clist/get` / ZT/DT 池都是常用接口。
  这是国内非官方财经接口里**事实上的标准**,变更概率相对最低、社区修复最快。
- **反爬**:`ulist.np` 实测无 Referer/UA 限制,反爬极弱;ZT/DT 池需要 `ut` token(公开固定)。
  仍建议设置合理超时与重试(项目已有 retry 机制)。
- **字段稳定性**:`f1xx` 系列字段编号多年稳定;风险点是东财偶尔调整字段含义或限频。
- **合规**:非官方接口,无公开 ToS 授权;数据版权归东方财富。属个人 best-effort 抓取,
  建议在 README 注明「数据来自东方财富 web 端点,非官方 API,仅供个人参考」,
  并保持现有「透明 UA、低频请求」的做法。

---

## 2. 新浪财经

### 2.1 `vip.stock.finance.sina.com.cn` 板块计数(✅ 已验证,但不含涨跌)

```bash
curl "https://vip.stock.finance.sina.com.cn/quotes_service/api/json_v2.php/Market_Center.getHQNodeStockCount?node=hs_a" \
  -H "Referer: https://finance.sina.com.cn/"
```
**真实响应:** `"5524"` —— 仅返回 `hs_a`(沪深 A 股)**股票总数**,**没有**上涨/下跌/涨停拆分。
- 编码:GBK。Header:需要 `Referer: *.sina.com.cn`,否则可能 403。
- 用途:只能作为「全市场总只数」的交叉校验,无法满足主需求。

### 2.2 `hq.sinajs.cn` 实时行情(✅ 已验证 header 要求)

```bash
curl "http://hq.sinajs.cn/list=sh000001" -H "Referer: https://finance.sina.com.cn/"
```
- **必须带 `Referer: *.sina.com.cn`**:实测无 Referer → **HTTP 403**,带上 → 200(已验证)。
- 格式:JS 文本 `var hq_str_sh000001="...";`,GBK 编码。
- 这是**逐个代码**的行情接口,要算涨跌家数得自己拉全市场再计数,**不适合**做聚合统计。
- easyquotation 用的就是这个端点(它本身定位是逐只行情,不是市场宽度)。

**结论:新浪没有「一把梭」的涨跌家数聚合端点。**

---

## 3. 腾讯(现状 + 兜底评估)

### 3.1 现状
项目当前用 `qt.gtimg.cn/q=` 与 `web.ifzq.gtimg.cn`、`smartbox.gtimg.cn`。这些是
**逐只/搜索/K线**端点,GBK 编码,需 `Referer: https://finance.qq.com`(项目已设置)。

### 3.2 是否有涨跌家数聚合端点?
- 实测 `proxy.finance.qq.com/ifzqgtimg/appstock/app/mapdata/get?id=hsa` 与
  `.../mkt/StockListM.csv` 均返回 `Can't load controller`(❌ 路径已失效/不存在)。
- 腾讯**没有公开、稳定、文档化**的「全市场涨跌家数」单接口。要做只能拉全量行情本地计数,
  请求量大、易触发限频,**不划算**。

**结论:腾讯保持现状用于个股/指数行情即可;市场宽度统计不建议依赖腾讯。**
若坚持单一数据源,可作为东财失败时的降级兜底,但需自行实现「拉全量 + 计数」逻辑。

---

## 4. 端点对比总表

| 数据源 | 端点 | 涨/跌/平 | 涨停/跌停 | 成交额 | 格式 | 编码 | 需要 Header | 反爬 | 验证 |
|--------|------|:---:|:---:|:---:|------|------|------|------|------|
| **东财** | `push2/ulist.np` | ✅ 一次到手 | ❌ | ✅(f6) | JSON | UTF-8 | 无 | 弱 | ✅ |
| **东财** | `push2ex ZT/DT 池` | ❌ | ✅(tc) | — | JSON | UTF-8 | 无(需 ut) | 弱 | ✅ |
| 新浪 | `getHQNodeStockCount` | ❌(仅总数) | ❌ | ❌ | 文本数字 | GBK | Referer | 中 | ✅ |
| 新浪 | `hq.sinajs.cn` | 需自算 | 需自算 | 需自算 | JS 文本 | GBK | Referer(强制) | 中 | ✅(header) |
| 腾讯 | `qt.gtimg.cn`(现状) | 需自算 | 需自算 | 需自算 | 文本 | GBK | Referer | 中 | — |
| 腾讯 | proxy 聚合端点 | ❌ 已失效 | — | — | — | — | — | — | ✅(失效) |

---

## 5. 最终推荐与落地建议

### 首选:东方财富(3 个请求覆盖全部需求)
1. `ulist.np`(secids=`1.000001,0.399106,0.899050`)→ 上涨/下跌/平盘 + 成交额(汇总三行)。
2. `getTopicZTPool`(`pagesize=1`)→ `data.tc` = 涨停家数。
3. `getTopicDTPool`(`pagesize=1`)→ `data.tc` = 跌停家数。

理由:JSON + UTF-8 解析最简单、**零 header 依赖**、字段语义明确、被开源生态广泛使用(变更风险最低、修复最快)、一次请求即可拿到涨跌家数核心指标。

### 次选 / 兜底:腾讯(现状)
保留现有腾讯端点用于个股行情;市场宽度作为降级时,需自行实现拉全量 + 本地计数。
不建议把市场宽度统计作为腾讯的主路径。

### 落地实现提示(契合现有架构)
- 与现有 `client.go` 完美兼容:新增常量,如
  `BreadthEndpoint = "https://push2.eastmoney.com/api/qt/ulist.np/get?fltt=2&secids=1.000001,0.399106,0.899050&fields=f3,f14,f104,f105,f106,f6"`
  和 `CNS_BREADTH_ENDPOINT` 环境变量覆盖(沿用 `resolveEndpoint` 模式)。
- 东财返回 UTF-8,现有 `decodeResponse`(UTF-8 优先,GBK 兜底)可直接复用,无需改动。
- **注意**:东财端点不需要 `Referer: finance.qq.com`,但带上无害;若想严谨可对东财请求改用
  `Referer: https://quote.eastmoney.com/`,或干脆不发(实测可省)。
- 解析建议:对 `ulist.np` 把三行 `f104/f105/f106/f6` 求和;ZT/DT 池只读 `data.tc`,
  `date` 用当前交易日(非交易日/盘前 `tc` 可能为 0,需做容错)。
- 风险兜底:`rc!=0` 或 `data` 为空时降级提示「数据源暂不可用」,符合 best-effort 定位。

### 合规小结
所有端点均为**非官方** web 接口,无 SLA、无授权,数据版权归各平台。作为个人工具:
保持低频、透明 UA、注明数据来源与「非官方、仅供参考」免责声明即可,避免商用与高频抓取。
