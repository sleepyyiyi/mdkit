# PLAN.md — 安全 Markdown→HTML 转换服务

> 关联目标：实现一个把**不可信** Markdown 转成**经消毒(防 XSS)** HTML 的 Go 后端服务，
> 并提供带 prompt 注入防御与失败降级的 LLM 文档摘要端点。
> 预计耗时：2 天
> AI 信任级：🟢 脚手架/解析器骨架 + 🟡 LLM 编排 + 🔴 消毒器/安全边界

---

## 0. 目标与约束

- **目标**：内部文档系统/AI Agent 提交 Markdown，服务返回可安全嵌入页面的 HTML；并能对文档生成摘要。
- **关键约束**：
  - 输入不可信——输出绝不能含可执行 HTML/JS（XSS 防御是第一目标）
  - 纯标准库，不引第三方 Markdown/消毒库（消毒逻辑必须可审计、可解释威胁模型）
  - 无状态——不持久化任何不可信输入
  - 输入大小有上限，避免资源耗尽
  - 截止 6-24 下班前完成
- **模糊点**：支持哪些 Markdown 语法？→ 决定支持**文档化子集**（标题/强调/行内代码/代码块/链接/列表/引用/段落），范围写进 CLAUDE.md + README，超出部分按已知限制处理。

---

## 1. 风险登记

| 风险 | 影响 | 应对 | 触发条件 |
|---|---|---|---|
| 原始 HTML 直通 → 存储/反射型 XSS | 严重安全事故，用户浏览器执行恶意脚本 | **先转义后格式化**：所有文本先 `html.EscapeString` 再套受控标签 | 输入含 `<script>`/`<img onerror>` 等 |
| `javascript:`/`data:` URL 注入 | 点击链接执行 JS | `sanitizeURL` scheme 白名单，非白名单降级为纯文本 | 输入含 `[x](javascript:...)` |
| ReDoS（正则灾难性回溯） | CPU 耗尽，拒绝服务 | Go `regexp` 是 RE2/线性时间，天然无回溯；**仍**加输入上限兜内存 | 攻击者构造病态输入 |
| 超大输入耗尽内存 | OOM | `MaxInputBytes` + transport 层 `http.MaxBytesReader` 双重校验 | 提交 > 1 MiB 文档 |
| LLM 超时/失败导致摘要不可用 | 功能退化 | 降级为本地抽取式摘要 + `ai_available:false` 标记 | LLM 返回错误或 context 取消 |
| Prompt 注入（文档伪造分隔符操纵 LLM） | LLM 被文档内"指令"劫持 | `buildPrompt` 用分隔符包裹文档并中和伪造的分隔符；文档严格作为数据 | 文档含"忽略上述指令"类内容 |

---

## 2. 任务拆解

> 每个子任务 ≤ 半天。AI 信任级：🟢 AI 主导 / 🟡 AI 主导+人工复核 / 🔴 人工主导。

### Day 1 上午：解析器骨架（受控子集）

**任务 1.1**：定义领域模型 + Markdown 解析器

- **输入上下文**：CLAUDE.md 分层规则与安全模型段；`internal/health/` 作为分层风格参考
- **产出标准**：
  - `model.go`：Convert/Summarize 请求响应体 + `MaxInputBytes`
  - `parser.go`：行级块解析（标题/代码块/引用/列表/段落）+ 行内格式化（粗体/斜体/行内代码/链接），**所有文本先转义**
- **验证方式**：`go build ./...` 通过；`Parse("# x")` 产出 `<h1>x</h1>`
- **人工检查点**：🔴 **确认 `inline()` 一定先 `html.EscapeString` 再套标签，没有任何文本绕过转义直达输出**
- **AI 信任级**：🟡

### Day 1 下午：消毒器（安全核心）

**任务 1.2**：URL scheme 白名单消毒器

- **输入上下文**：任务 1.1 的 `renderLink`；OWASP XSS 防御要点（scheme 白名单优于黑名单）
- **产出标准**：`sanitizer.go` 的 `sanitizeURL`——白名单 `http/https/mailto` + 相对 URL；其余（`javascript:`/`data:`/`vbscript:`/控制字符混淆）降级
- **验证方式**：`sanitizer_test.go` ≥ 15 个用例（含大小写混淆、Tab 混淆、相对路径、含冒号路径）
- **人工检查点**：🔴 **逐条确认是白名单而非黑名单；确认混淆绕过（大小写/控制字符/前导空格）都被拦**
- **AI 信任级**：🔴 人工主导，安全相关

### Day 2 上午：服务编排 + LLM 摘要 + HTTP 层

**任务 2.1**：Service 编排 + 输入上限

- **输入上下文**：任务 1.1/1.2 产出；CLAUDE.md 接口约定
- **产出标准**：`service.go`（Convert：超限拒绝 + 调 Parse；Summarize：转纯文本 → 调 LLM → 失败降级）
- **验证方式**：`service_test.go` 覆盖正常/超限/降级/context 取消
- **人工检查点**：🔴 **确认超大输入在 service 与 transport 两层都被挡；确认 LLM 失败时一定走 fallback 而非 500**
- **AI 信任级**：🟡

**任务 2.2**：LLM 接口 + prompt 注入防御 + Handler

- **输入上下文**：任务 2.1；考题题型（LLM 集成 + 注入防御）
- **产出标准**：
  - `llm.go`：`LLM` 接口 + `MockLLM`（确定性）+ `FailingLLM`（降级测试）+ `buildPrompt`（分隔符包裹 + 中和伪造分隔符）
  - `handler.go`：`POST /convert`、`POST /summarize`，仅 JSON 编解码 + `MaxBytesReader`
  - `cmd/server/main.go` 注册路由
- **验证方式**：手动 curl 两端点；`handler_test.go` 覆盖正常/XSS 中和/无效 JSON/超限
- **人工检查点**：🔴 **确认 handler 不含业务逻辑；确认 `buildPrompt` 真能中和文档内伪造的分隔符**
- **AI 信任级**：🟡

### Day 2 下午：测试 + Review + 收口

**任务 2.3**：安全/边界/性能测试 + AI Review + QA 报告

- **输入上下文**：全部代码；`review-checklist.md` 四象限
- **产出标准**：
  - `parser_test.go`：XSS 用例（script/img onerror/svg onload/data uri/属性突破）+ 边界（空/未闭合 fence/海量括号星号）+ benchmark
  - QA_REPORT.md 全章节 + ≥ 1 条「AI 建议被拒」真实案例
- **验证方式**：`go test -race ./...` 全绿；`go test -bench=.` 确认线性；`go vet ./...` 零警告
- **人工检查点**：🔴 **确认 XSS 测试覆盖"原始 HTML 被转义"与"危险 URL 被降级"两条核心路径；确认 QA 缺陷案例真实**
- **AI 信任级**：🟢 生成 + 🟡 用例审核

---

## 3. 人工检查点（不可省略）

AI 永远不能代替你做的判断：

- [ ] **Day 1 终点**：消毒是不是白名单？有没有任何文本绕过 `html.EscapeString` 直达输出？
- [ ] **Day 2 上午**：超大输入是否两层都挡？LLM 失败是否一定降级不抛 500？
- [ ] **Day 2 终点**：XSS 测试是否覆盖原始 HTML 转义 + 危险 URL 降级两条线？QA 缺陷案例是否真实？

---

## 4. 验收标准

- [ ] `go test -race ./...` 全绿
- [ ] `go vet ./...` 零警告，`go build ./...` 通过
- [ ] XSS 测试：`<script>`/`onerror`/`javascript:`/`data:` 全部被中和
- [ ] 输入上限：超 `MaxInputBytes` 被拒（service + transport 双层）
- [ ] LLM 失败/取消：降级为抽取式摘要 + `ai_available:false`
- [ ] benchmark 确认病态输入线性时间完成
- [ ] QA_REPORT 含 ≥ 1 个真实「AI 建议被拒」案例
- [ ] 每天有 commit，git log 可追踪进度
