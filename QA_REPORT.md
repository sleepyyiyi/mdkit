# QA_REPORT.md — 质量保障报告

> 对齐 L2 维度③「AI 交付质量保障」：测试覆盖关键路径 / AI Review 记录 /
> **性能** / **安全** / 系统稳定性可复验。

---

## 1. 测试

- **策略**：安全是第一目标，测试重点放在三条线——(1) XSS 中和（原始 HTML 转义 + 危险 URL 降级）、(2) 边界稳定性（空/未闭合 fence/海量符号不 panic）、(3) LLM 降级（失败/取消走 fallback）。解析器、消毒器、service、handler 各层 table-driven 覆盖。
- **现状**：
  - 测试文件：5 个（`parser_test.go` / `sanitizer_test.go` / `service_test.go` / `handler_test.go` / `health/handler_test.go`）
  - markdown 模块：**19 个测试函数 / 61 个用例（含子用例）**，全绿
  - 含 1 个 benchmark（`BenchmarkParse_Pathological`）验证病态输入线性时间
- **命令**：`go test -race -v ./...`
- [x] 关键路径有测试　[x] 边界/异常有测试　[x] 安全（XSS/注入）逻辑覆盖　[x] 降级路径覆盖

---

## 2. AI 审查记录（Review）

> 按 [review-checklist.md](./review-checklist.md) 四象限逐项审查 markdown 模块。

| # | AI 发现 | 位置 | 判断 | 理由 |
|---|---|---|---|---|
| 1 | 建议改用成熟三方库 `goldmark` + `bluemonday` 做解析与消毒，"手写消毒器易出漏洞" | 整个 markdown 模块 | **部分采纳** | CLAUDE.md 明确约束纯标准库，且本练习的**学习目标**就是理解 XSS 威胁模型——黑盒库会把它藏起来。对受控子集，输出标签白名单 + 先转义的方案是可审计的。**但生产环境确实应换 goldmark+bluemonday（久经考验）**，已作为「已知限制」写入 README。 |
| 2 | 警告 `parser.go` 的正则有 **ReDoS** 风险，建议重写避免 `.*` 回溯 | `parser.go` 正则 | **拒绝** | Go 的 `regexp` 是 **RE2 实现，线性时间、无回溯**——经典 ReDoS（PCRE/JS 那种灾难性回溯）在 Go 标准库里**不可能发生**。这是从其他语言误迁移的担忧。已用 `BenchmarkParse_Pathological` 实测病态输入线性完成佐证。**这是「AI 看似合理但错」的典型**：仍保留输入上限是为了兜内存，不是兜 ReDoS。 |
| 3 | 建议用正则 `<script.*?>.*?</script>` 剥离危险标签 | （建议新增） | **拒绝** | 正则解析 HTML 不可靠（嵌套/变形绕过），且黑名单天然漏报。正确做法是**白名单 + 先转义**：原始 HTML 一律转义为字面量，根本不存在"剥离"。采纳此建议反而会引入虚假安全感。 |
| 4 | `renderLink` 对 URL 内嵌套括号（`[x](javascript:alert(1))`）会留下尾随 `)` 文本 | `parser.go:renderLink` | **接受（标注限制）** | 安全上无问题（`javascript:` 已被降级，无脚本泄漏），仅残留一个 `)` 字符。受控子集不支持 URL 内嵌套括号，已在 README「已知限制」标注。 |
| 5 | `writeJSON` 在 Encode 部分写出后再 `http.Error` 会 superfluous WriteHeader | `handler.go:writeJSON` | **接受** | 响应体已开始写时再写状态码会告警。生产应改为 log-only。当前 Encode 目标是内存 struct，失败概率极低；保留兜底但注释说明。 |

**总结**：5 项发现中，1 部分采纳、2 拒绝、2 接受。两条拒绝（ReDoS 误迁移、正则剥标签）都是体现人工判断价值的关键案例。

---

## 3. 性能风险

- [x] 已过 review-checklist「性能」象限
- **检查项与佐证**：
  - **ReDoS / 正则回溯**：Go `regexp` 为 RE2，线性时间，**无灾难性回溯**。`BenchmarkParse_Pathological`（32000 个行内元素）实测 ~9ms/op，确认线性。✅
  - **字符串拼接复杂度**：解析器全程用 `strings.Builder`，避免 O(n²) 拼接。✅
  - **输入上限**：`MaxInputBytes`(1 MiB) 在 service 层校验 + transport 层 `http.MaxBytesReader` 兜底，防内存耗尽。✅
  - **资源泄漏**：无 goroutine、无文件/连接句柄；纯函数式处理。✅
  - **复验**：`go test -bench=BenchmarkParse_Pathological -benchtime=20x ./internal/markdown/`

## 4. 安全风险

- [x] 已过 review-checklist「安全」象限
- **检查项与佐证**：
  - **XSS（核心）**：先 `html.EscapeString` 再格式化——原始 `<script>`/`<img onerror>`/`<svg onload>` 全部转义为字面量，`TestParse_XSS` 覆盖。✅
  - **危险 URL**：`sanitizeURL` scheme 白名单（http/https/mailto + 相对），`javascript:`/`data:`/`vbscript:`/大小写与 Tab 混淆全部降级，`TestSanitizeURL` 16 用例覆盖。✅
  - **属性突破**：URL 内的 `"` 在整行转义阶段已变 `&#34;`，无法突破 href 属性，`TestParse_XSS/attribute_breakout` 覆盖。✅
  - **Prompt 注入**：`buildPrompt` 用分隔符包裹文档并中和伪造分隔符，文档严格作为数据，`TestBuildPrompt_NeutralizesDelimiterForgery` 覆盖。✅
  - **错误不泄露内部**：handler 错误只回固定常量字符串，不含 err 细节、不含用户输入。✅
  - **无状态**：不可信输入不落盘，无持久化攻击面。✅
  - **已知限制**：受控子集不覆盖全部 CommonMark 语法；生产建议 goldmark+bluemonday。

## 5. 系统稳定性 / 可复验

- [x] CI 门禁就位（vet + test + build，见 `.github/workflows/ci.yml`）
- [x] 稳定性：`TestParse_EdgeCases` + `TestParse_UnclosedFenceNoPanic` 覆盖空输入/未闭合 fence/海量括号星号/超长列表标记，均不 panic
- [x] `go test -race ./...` 全绿，无 race
- [x] 环境可复现：仅 Go 标准库，`go.mod` 无第三方依赖，README 有完整运行命令

**复验方式**：
```bash
git clone <repo> && cd mdkit
go vet ./... && go test -race ./... && go build ./...
go test -bench=BenchmarkParse_Pathological -benchtime=20x ./internal/markdown/

go run ./cmd/server &
# XSS 中和
curl -s -X POST localhost:8080/convert -H 'Content-Type: application/json' \
  -d '{"markdown":"<script>alert(1)</script> [evil](javascript:alert(1)) [ok](https://go.dev)"}'
# 摘要 + 降级
curl -s -X POST localhost:8080/summarize -H 'Content-Type: application/json' \
  -d '{"markdown":"The quick brown fox jumps.","max_words":3}'
```
预期：`convert` 输出含 `&lt;script&gt;`、不含 `<script>`、不含 `javascript:`，`ok` 链接为合法 `<a href="https://go.dev">`。
