#!/bin/sh
# 将 release tag 移到当前最新提交（确保 go.sum 和 workflow 已提交）
# 用法: ./scripts/retag-release.sh v1.0.0

set -e
TAG="${1:-v1.0.0}"
echo "Moving tag $TAG to current HEAD..."
git tag -d "$TAG" 2>/dev/null || true
git push origin --delete "$TAG" 2>/dev/null || true
git tag "$TAG"
git push origin "$TAG"
echo "Done. Tag $TAG now points to $(git rev-parse HEAD)"
