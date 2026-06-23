# mdkit — 项目宪法（CLAUDE.md）

> L2 达标起步骨架（Go 后端）。greenfield 初始化即写满官方 10 类资产的「最小可用」版本，
> 细节随迭代补。本文件是项目唯一完整宪法（见文末「单一真相」）。

## 项目定位　〔官方资产: 项目背景〕
安全的 Markdown→HTML 转换服务：把不可信 Markdown 转成经过消毒(防 XSS)的 HTML，并提供 LLM 文档摘要(带 prompt 注入防御与失败降级)。给内部文档系统/AI Agent 用。

## 技术栈　〔官方资产: 系统架构〕
- 语言/运行时：Go 1.22+
- HTTP：标准库 `net/http`（`go 1.22` 的 `mux` 路由，方法+路径模式 `POST /convert`）
- 存储：**无状态服务**，不持久化任何数据（转换/摘要均为纯函数式处理）。这是有意的安全设计——不可信输入不落盘。
- LLM：接口抽象 `markdown.LLM`，当前 `MockLLM` 确定性实现；生产可替换为 OpenAI 兼容实现，接口不变
- HTML 转义：标准库 `html.EscapeString`（不引第三方），消毒逻辑自管以保证可审计
- 测试框架：Go 内置 `testing` + table-driven + benchmark

## 架构要点　〔官方资产: 系统架构〕
- 分层：`传输(internal/<mod>/handler)` → `逻辑(internal/<mod>/service)` → `领域(parser/sanitizer/llm)`
- 依赖方向：handler → service → parser/sanitizer/llm；**禁止反向**，禁止 handler 直接调解析/消毒
- 装配：`cmd/server/main.go` 只做 wire up（构造 service、注册路由），不写业务
- 依赖注入：`LLM` 接口通过 `NewService(llm)` 注入，测试可替换为 `MockLLM`/`FailingLLM`；禁止包级全局可变状态

## 安全模型（本项目核心）　〔官方资产: 系统架构 / 安全约定〕
> 输入是**不可信的** Markdown，输出会被浏览器渲染——这是一个 XSS 攻击面。三条不可违反的不变量：
1. **先转义后格式化**：所有文本进入 `inline()` 时先 `html.EscapeString`，再套受控标签。原始 HTML（`<script>` 等）一律渲染为字面文本，绝不直通。
2. **URL scheme 白名单**：生成 `<a href>` 前必过 `sanitizeURL`，只放行 `http/https/mailto` 与相对 URL；`javascript:`/`data:`/`vbscript:` 等一律降级为纯文本。
3. **永不输出事件属性**：解析器只产出固定标签集，从不发出 `on*` 属性；用户无法注入属性。
- 输入大小上限 `MaxInputBytes`（1 MiB），transport 层 `http.MaxBytesReader` + service 层双重校验。
- Markdown 解析为**文档化子集**（标题/强调/行内代码/代码块/链接/有序无序列表/引用/段落）；超出范围的语法（如 URL 内嵌套括号）按已知限制处理，见 README。

## 接口约定　〔官方资产: 接口约定〕
- 领域模型与 service 同包（如 `health.Status`）；DTO 用 struct tag 控制 JSON 序列化
- HTTP 错误：对外只给通用信息 + 合适状态码，**不泄露**内部堆栈/SQL；对内 `log` 带上下文
- 返回结构统一：成功返回领域 JSON，失败走 `http.Error` 或统一 error envelope

## 编码规范　〔官方资产: 编码规范〕
- 包名小写无下划线；导出符号 `PascalCase` 且**必须有 doc comment**
- 错误处理显式 `if err != nil`，**禁止**吞错（空 `_ = err`）；错误用 `fmt.Errorf("...: %w", err)` 包装
- 提交前 `gofmt` 格式化；提交信息：`type(scope): 摘要`（type ∈ feat/fix/refactor/test/chore）

## 测试要求　〔官方资产: 测试要求〕
- 核心逻辑（service 业务、handler 编解码与状态码）必须有 table-driven 单测
- 边界：空输入、错误分支、并发安全（涉及共享状态时）
- 跑：`go test ./...`；新增逻辑分支需配套测试方可合并

## 工具与命令约束　〔官方资产: 工具约束〕
- 允许：`go run ./cmd/server` / `go build ./...` / `go test ./...` / `go vet ./...` / `gofmt`
- **禁止**：handler 内直接访问数据库/外部 IO（走 service/repository）；吞错；
  包级全局可变状态；提交 `.env` / 密钥 / 编译产物（见 `.gitignore`）

## Review 标准（合并前自检）　〔官方资产: Review 标准〕
- [ ] 关键逻辑 / 边界条件已覆盖（含测试）
- [ ] **性能风险**已看（循环内重 IO、未关闭的 `resp.Body`/连接、无界 goroutine/内存）
- [ ] **安全风险**已看（SQL 注入用参数化、鉴权校验、错误不泄露内部、密钥不入日志/仓库）
- [ ] 架构分层未被破坏（无反向依赖、handler 未直碰存储）
- [ ] 改代码同时改了对应资产（防漂移）
- 详见 [review-checklist.md](./review-checklist.md)

## 任务流程　〔官方资产: 任务流程〕
- 分支：`feat/<简述>` / `fix/<简述>`，从最新主干切；commit 引用 issue 号（`#NN`）
- 改任意模块前：先读本文件相关段；动接口/领域模型先想清依赖方向

## Prompt 模板库　〔官方资产: Prompt 模板〕
- 见 [prompt-library.md](./prompt-library.md)（高频任务可复用 Prompt + 正反例）

## 典型示例　〔官方资产: 典型示例〕
- 正例：文本进入输出前**先转义再格式化**（安全不变量 1）：
  ```go
  // internal/markdown/parser.go
  func inline(text string) string {
      esc := html.EscapeString(text)        // ← 先转义，原始 HTML 变字面量
      esc = reBold.ReplaceAllString(esc, "<strong>$1</strong>")
      esc = reLink.ReplaceAllStringFunc(esc, renderLink) // ← URL 过白名单
      return esc
  }
  ```
- 反例（**禁止**）：直接拼接用户输入到 HTML / 凭正则"剥离" script 标签：
  ```go
  // 错误：原始 HTML 直通 → 存储型/反射型 XSS
  out := "<p>" + userInput + "</p>"
  // 错误：正则解析 HTML 不可靠，且诱导用 .*? 回溯
  re := regexp.MustCompile(`<script.*?>.*?</script>`) // 既漏又招黑
  ```

---

## 单一真相（最高优先级）　← 防漂移的关键，务必保留
- **本文件 `CLAUDE.md` 为项目唯一完整宪法。**
- `.cursor/rules/*.mdc` 等只放**按路径摘要 + 指针**，不写新规则；新要求一律先更新本文件。
- 各类 Agent / IDE 插件若与本文冲突，**以本文件为准**，并提 PR 修正歧义。
