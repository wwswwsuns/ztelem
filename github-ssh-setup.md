# GitHub SSH密钥配置指南

## 🔑 SSH密钥已生成

你的SSH公钥已经生成，需要添加到GitHub账户中。

## 📋 配置步骤

### 1. 复制SSH公钥
已为你生成SSH密钥，公钥内容如下（请复制完整内容）：

### 2. 添加到GitHub
1. 登录 [GitHub.com](https://github.com)
2. 点击右上角头像 → Settings
3. 左侧菜单选择 "SSH and GPG keys"
4. 点击 "New SSH key"
5. 填写信息：
   - **Title**: `Telemetry Server Key`
   - **Key**: 粘贴上面的公钥内容
6. 点击 "Add SSH key"

### 3. 验证连接
添加密钥后，执行以下命令验证：
```bash
ssh -T git@github.com
```

### 4. 推送代码
验证成功后，执行：
```bash
git push -u origin main --force
```

## ⚠️ 重要提示
- 请妥善保管私钥文件 `~/.ssh/id_ed25519`
- 公钥可以安全分享，私钥绝对不能泄露
- 如果遇到问题，可以重新生成密钥对

## 🔄 备用方案
如果SSH仍有问题，可以考虑：
1. 使用GitHub CLI工具
2. 直接在GitHub网页上传文件
3. 使用代理或VPN解决网络问题