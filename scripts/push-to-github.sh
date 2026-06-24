#!/usr/bin/env bash
# 将本地 mdkit 推送到 https://github.com/sleepyyiyi/mdkit
# 用法: bash scripts/push-to-github.sh
set -euo pipefail

cd "$(dirname "$0")/.."

# 清除可能失效的环境变量 token（会导致 gh 401）
unset GITHUB_TOKEN GH_TOKEN 2>/dev/null || true

if ! gh auth status >/dev/null 2>&1; then
  echo "请先登录 GitHub（推荐浏览器方式，不要用失效 token）："
  echo "  gh auth login"
  echo "然后执行: gh auth setup-git"
  exit 1
fi

gh auth setup-git

git remote remove origin 2>/dev/null || true
git remote add origin https://github.com/sleepyyiyi/mdkit.git
git branch -M main

echo "▶ 推送 main 到 origin（覆盖远端初始 README 提交）..."
git push -u origin main --force-with-lease

echo ""
echo "✅ 完成: https://github.com/sleepyyiyi/mdkit"
echo "下一步: 在网页创建 Issue → https://github.com/sleepyyiyi/mdkit/issues/new"
