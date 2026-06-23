---
name: sanitized-block
description: 在 mdkit 里给 Markdown 解析器新增一种「受控渲染块/行内元素」时使用（如新增 ~~删除线~~、表格、脚注等）。当用户说"支持 XX 语法 / 加一种 Markdown 元素 / 解析器加 XX"时触发，自动套用「先转义后格式化」的安全骨架并补 XSS 测试。不适用于消毒器(sanitizer.go)或 LLM(llm.go)改动。
---

# 新增一个受控渲染块

> 目标：让"给解析器加一种语法"这个高频动作**每次都自动满足安全不变量**，不靠人记。
> 完整约定见 [CLAUDE.md](../../../CLAUDE.md)「安全模型」段，本 skill 只编排步骤，不新增规则。

## 何时用（决定加载）
- 用户要在 `internal/markdown/parser.go` 新增一种块级或行内 Markdown 元素。
- 关键词：支持 XX 语法、加删除线/表格/脚注、解析器加 XX。

## 步骤
1. **先读** [CLAUDE.md](../../../CLAUDE.md) 的「安全模型」三不变量 + 「架构要点」。
2. 在 `parser.go` 新增解析逻辑，**强制遵守**：
   - 块级：文本进入前先 `html.EscapeString`（或经已转义的 `inline()`），**绝不**把原始片段拼进输出。
   - 行内：只能在 **已转义的** 字符串上做正则替换（参考 `reBold`/`reItalic` 的用法）。
   - 若产出带属性的标签（如表格 `<td>`）：属性值只能是固定常量，**禁止**把用户输入塞进任何属性。
   - 若涉及 URL：必须过 `sanitizeURL`，非白名单降级为纯文本（参考 `renderLink`）。
3. 正则一律用 `regexp.MustCompile`（RE2 线性，无需担心 ReDoS），但**不要**放宽 `MaxInputBytes`。
4. **补测试**：在 `parser_test.go` 加
   - 正常渲染用例（基础格式）
   - **至少 1 个 XSS 用例**：把恶意输入（`<script>`/`javascript:`/`on*`）喂进新语法，断言输出被中和。
5. 运行验收命令，全绿才算完成。

## 验收（必须全过）
- [ ] `go test -race ./internal/markdown/` 通过，新语法有正常 + XSS 两类测试
- [ ] `go vet ./...` 零告警
- [ ] 新逻辑在 `parser.go`（解析），未改 `handler.go`/`service.go` 的职责边界
- [ ] 新元素的任何文本/属性都经过转义或白名单，无原始输入直通

## 正例
```go
// 新增 ~~删除线~~：在【已转义】文本上替换，安全
var reStrike = regexp.MustCompile(`~~([^~]+)~~`)

func inline(text string) string {
    esc := html.EscapeString(text)            // ← 先转义
    esc = reStrike.ReplaceAllString(esc, "<del>$1</del>") // ← 在转义后的串上做
    // ... 其余行内规则
    return esc
}
```

## 反例（会被 Review 打回）
```go
// ❌ 在【未转义】的原文上替换 → 原始 HTML 直通 → XSS
func inline(text string) string {
    text = reStrike.ReplaceAllString(text, "<del>$1</del>") // 没先 EscapeString
    return text                                             // <script> 直接输出
}
```
**错在**：跳过 `html.EscapeString`，恶意 `~~<script>~~` 会把 `<script>` 直通到输出，违反安全不变量 1。
