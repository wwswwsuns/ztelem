#!/bin/bash

# GitHub上传命令脚本
# 请将 YOUR_USERNAME 替换为你的GitHub用户名
# 请将 REPOSITORY_NAME 替换为你创建的仓库名

echo "=== GitHub上传步骤 ==="
echo "1. 在GitHub上创建新仓库后，复制仓库URL"
echo "2. 执行以下命令："
echo ""

echo "# 添加远程仓库（替换为你的仓库URL）"
echo "git remote add origin https://github.com/YOUR_USERNAME/REPOSITORY_NAME.git"
echo ""

echo "# 设置主分支名称"
echo "git branch -M main"
echo ""

echo "# 推送到GitHub"
echo "git push -u origin main"
echo ""

echo "=== 示例命令（请替换URL） ==="
echo "git remote add origin https://github.com/yourusername/telemetry-system.git"
echo "git branch -M main"
echo "git push -u origin main"
echo ""

echo "=== 验证上传 ==="
echo "git remote -v  # 查看远程仓库配置"
echo "git status     # 查看仓库状态"