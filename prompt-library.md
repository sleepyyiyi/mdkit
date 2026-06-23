# prompt-library.md — Prompt 模板库

> 对齐 L2 维度①「AI 协作资产沉淀」中的核心子项:可复用 Prompt 模板。
> 把项目里高频的 AI 协作动作沉淀成模板,带变量说明 + 正反例,团队复用、口径一致。

---

## 0. 占位符与安全约定
- `{{user_input}}` — 来自用户/外部,**不可信**,渲染/拼接前必须转义或参数化
- `{{ctx}}` — 来自系统上下文(数据库/配置),相对可信,外部来源仍需 sanitize
- 不同用户/租户的上下文**绝不混用**

## 1. 新增模块 / 组件
```
参照 CLAUDE.md 的「架构要点 / 编码规范 / 工具约束」,在 {{layer}} 新增 {{name}}。
要求:遵守依赖方向(禁止反向)、类型来自单一来源、禁止 as any/吞错、配套测试。
给出实现 + 测试,并说明放在哪个目录。
```

## 2. 跑 Code Review
```
按 review-checklist.md 四象限逐项 review 以下 diff。
每项给:是否命中 / 位置(文件:行) / 严重等级 / 修复建议。最后给"是否建议 merge"。
diff:
@<相关文件或粘 diff>
```

## 3. 拆任务 / 写 PLAN
```
基于这个 issue(目标/约束/风险见正文),按 PLAN.md 模板拆成 ≤ 半天的子任务,
每个子任务给:输入上下文 / 产出标准 / 验证方式 / 人工检查点 / AI 信任级。
issue:
<粘 issue 正文>
```

## 4. 补测试
```
为 {{target}} 补 table-driven/单元测试,覆盖:核心路径 + 空输入/错误分支/边界。
不改被测逻辑;若发现 bug 先报告再问是否修。
```

---

## 5. 审查 XSS 防御（本项目特定）
```
审查 {{file}} 是否违反 CLAUDE.md「安全模型」三不变量:
1. 所有文本是否先 html.EscapeString 再套标签(原始 HTML 不得直通)
2. 生成 <a href> 前是否过 sanitizeURL scheme 白名单
3. 是否输出了任何 on* 事件属性
逐条给:是否命中/位置/修复。
```
- ✅ 正例（先转义后格式化）：
  ```go
  esc := html.EscapeString(text)
  esc = reBold.ReplaceAllString(esc, "<strong>$1</strong>")
  ```
- ❌ 反例（原始输入直通 / 黑名单剥标签）：
  ```go
  out := "<p>" + userInput + "</p>"                       // 存储型 XSS
  re := regexp.MustCompile(`<script.*?>.*?</script>`)     // 黑名单不可靠
  ```

## 6. 判断性能担忧是否成立（本项目特定）
```
有人提出 {{concern}}(如 ReDoS/内存)。结合本项目技术栈核实:
- Go regexp 是 RE2(线性,无回溯) → 经典 ReDoS 不适用,不要照搬其他语言结论
- 真实风险是无界输入内存 → 看是否有 MaxInputBytes + MaxBytesReader
给出"成立/不成立 + 依据(实测 benchmark 或代码佐证)"。
```
- ✅ 正例（有依据地拒绝）：用 `BenchmarkParse_Pathological` 实测线性，证明 ReDoS 担忧不成立
- ❌ 反例（无依据地照搬）：因为正则里有 `.*` 就断言 ReDoS，不核实 RE2 语义

---

> 新增高频 Prompt 时往这里加,并标注变量与至少 1 个正/反例。完整工程约定以 [CLAUDE.md](./CLAUDE.md) 为准。
