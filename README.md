# mdkit — 安全 Markdown→HTML 转换服务

把**不可信** Markdown 转成经过消毒(防 XSS)的 HTML，并提供带 prompt 注入防御与失败降级的 LLM 文档摘要。给内部文档系统 / AI Agent 用。

> Go 后端，L2 达标 greenfield 项目（骨架由 l2-init 生成，业务自填）。
> 完整工程约定见 [CLAUDE.md](./CLAUDE.md)，安全模型是核心，务必先读。

## 快速上手 — 命令行（推荐）

```bash
go build -o mdkit ./cmd/mdkit

# Markdown → HTML 输出到终端
mdkit convert README.md

# Markdown → 可浏览器打开的 HTML 文件
mdkit convert README.md -o output.html

# 从 stdin 管道输入
cat doc.md | mdkit convert -

# 文档摘要
mdkit summarize README.md -n 20

# 启动 HTTP API 服务
mdkit serve
mdkit serve -addr :3000

# 帮助
mdkit help
```

## 快速上手 — HTTP API

```bash
go run ./cmd/server        # 或 mdkit serve

# 转换（含 XSS 中和）
curl -X POST localhost:8080/convert -H 'Content-Type: application/json' \
  -d '{"markdown":"# Hi\n\n<script>alert(1)</script>\n\n[ok](https://go.dev)"}'

# 摘要
curl -X POST localhost:8080/summarize -H 'Content-Type: application/json' \
  -d '{"markdown":"The quick brown fox.","max_words":5}'

curl localhost:8080/healthz
```

## 验收（提交前全绿）

```bash
go vet ./...
go test -race ./...
go build ./cmd/mdkit ./cmd/server
go test -bench=BenchmarkParse_Pathological -benchtime=20x ./internal/markdown/
```

## 接口

### CLI

| 命令 | 说明 |
|---|---|
| `mdkit convert <file.md> [-o out.html]` | Markdown→消毒 HTML（stdout 或文件） |
| `mdkit convert -` | 从 stdin 读 |
| `mdkit summarize <file.md> [-n N]` | 文档摘要（默认 40 词） |
| `mdkit serve [-addr :8080]` | 启动 HTTP API |

### HTTP

| 方法 | 路径 | 入参 | 出参 |
|---|---|---|---|
| POST | `/convert` | `{"markdown":"..."}` | `{"html":"...","bytes":N}` |
| POST | `/summarize` | `{"markdown":"...","max_words":N}` | `{"summary":"...","ai_available":bool}` |
| GET | `/healthz` | — | `{"ok":true,"version":"..."}` |

## 安全模型（核心）

输入不可信、输出会被浏览器渲染——这是 XSS 攻击面。三条不可违反的不变量（详见 CLAUDE.md）：

1. **先转义后格式化**：所有文本先 `html.EscapeString`，原始 HTML（`<script>` 等）一律渲染为字面文本。
2. **URL scheme 白名单**：`<a href>` 只放行 `http/https/mailto` + 相对 URL；`javascript:`/`data:`/`vbscript:` 降级为纯文本。
3. **永不输出事件属性**：只产出固定标签集，从不发出 `on*` 属性。

外加：输入上限 `MaxInputBytes`(1 MiB)，service + transport 双层校验；无状态，不落盘。

## 项目结构

```
cmd/
  mdkit/main.go              ★ 统一 CLI 入口(convert/summarize/serve)
  server/main.go               纯 HTTP 入口(向后兼容)
internal/
  markdown/                  ★ 核心业务模块
    model.go                    请求响应体 + MaxInputBytes
    parser.go                   受控子集解析(先转义后格式化)
    parser_test.go              基础格式 + XSS + 边界 + benchmark
    sanitizer.go                URL scheme 白名单(安全核心)
    sanitizer_test.go           16 个 URL 消毒用例
    llm.go                      LLM 接口 + MockLLM + FailingLLM + buildPrompt(注入防御)
    service.go                  编排(转换/摘要/降级)+ 输入上限
    service_test.go             正常/超限/降级/取消/注入防御
    handler.go                  HTTP 编解码(2 路由)+ MaxBytesReader
    handler_test.go             正常/XSS 中和/无效 JSON/超限
  health/                       健康检查(脚手架示例)
.cursor/rules/go.mdc            分层规则摘要(指向 CLAUDE.md)
.claude/skills/sanitized-block/ 自定义 Skill:新增受控渲染块的标准流程
.github/                        issue / PR 模板 + CI 门禁(vet + test + build)
CLAUDE.md                       项目宪法(官方 10 类资产 + 安全模型 + 单一真相)
PLAN.md                         任务规划(2 天,每子任务四要素)
QA_REPORT.md                    质量报告(AI Review 5 条 + 性能/安全/稳定性)
review-checklist.md             Review 四象限清单
prompt-library.md               Prompt 模板库(含 XSS/性能核实正反例)
```

## 已知限制（诚实标注）

- 解析器是**文档化子集**，非完整 CommonMark：不支持表格、任务列表、URL 内嵌套括号（如 `[x](a(b))` 会留尾随字符，但安全无虞）。
- `MockLLM` 是确定性占位实现（取前 N 词），非真实模型；生产替换 `markdown.LLM` 接口即可。
- 生产级消毒建议用久经考验的 `goldmark` + `bluemonday`；本项目手写消毒是为可审计 + 学习威胁模型（见 QA_REPORT §2 Review #1）。

## L2 三维度

- **资产沉淀**：[CLAUDE.md](./CLAUDE.md) 覆盖官方 10 类资产 + 安全模型 + 单一真相；`LLM` 接口抽象；自定义 Skill；prompt-library 含 XSS/性能核实正反例
- **任务规划**：[PLAN.md](./PLAN.md) 2 天拆解，每子任务含输入/产出/验证/人工检查点；风险登记 6 条（XSS/危险 URL/ReDoS/内存/LLM 降级/prompt 注入）
- **质量保障**：19 测试函数/61 用例(`-race`)+ benchmark + [QA_REPORT.md](./QA_REPORT.md)（5 条 AI Review，含 2 条「看似合理但被拒」）+ 性能/安全/稳定性三线检查 + CI 门禁
