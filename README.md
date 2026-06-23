# mdkit

安全的 Markdown→HTML 转换服务：把不可信 Markdown 转成经过消毒(防 XSS)的 HTML，并提供 LLM 文档摘要(带 prompt 注入防御与失败降级)。给内部文档系统/AI Agent 用。

> Go 后端，L2 达标起步骨架（由 l2-init 生成）。完整工程约定见 [CLAUDE.md](./CLAUDE.md)。

## 运行

```bash
go mod tidy
go run ./cmd/server        # 起服务,默认 :8080
curl localhost:8080/healthz
```

## 验收（提交前全绿）

```bash
go vet ./...
go test ./...
go build ./...
```

## 结构

```
cmd/server/main.go          入口:装配 service + 注册路由
internal/health/            示例业务模块(按真实领域复制改名)
  service.go                领域模型 + 业务逻辑
  handler.go                HTTP 编解码(只编解码,不写业务)
  handler_test.go           table-driven 单测
.cursor/rules/go.mdc        分层规则摘要(指向 CLAUDE.md)
.github/                    issue / PR 模板 + CI 门禁
PLAN.md / QA_REPORT.md      任务规划 / 质量报告(维度②③)
review-checklist.md         合并前 Review 清单
```

## L2 三维度

- **资产沉淀**：[CLAUDE.md](./CLAUDE.md) 覆盖官方 10 类资产 + 单一真相
- **任务规划**：写 issue（目标/约束/风险）→ 填 [PLAN.md](./PLAN.md) → 分支 → PR（Closes #）
- **质量保障**：测试 + [review-checklist.md](./review-checklist.md)（含性能/安全）+ CI 门禁 + [QA_REPORT.md](./QA_REPORT.md)
