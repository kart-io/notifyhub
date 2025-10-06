# GitHub 自动打 Tag 工作流使用指南

本项目提供了两个自动化的 GitHub Workflow 来管理版本标签和发布。

## 📋 工作流概述

### 1. Auto Tag (`auto-tag.yml`)

**简单的自动标签工作流**

- 📌 **触发时机**: 代码推送到 `main` 或 `master` 分支时自动触发
- 🎯 **版本策略**: 基于提交信息的关键词自动确定版本号
- 🚀 **功能**:
  - 自动计算下一个版本号
  - 生成简单的变更日志
  - 创建 Git 标签
  - 创建 GitHub Release

**版本规则**:

- `feat!:` 或 `BREAKING CHANGE:` → Major 版本 (v1.x.x → v2.0.0)
- `feat:` 或 `feature:` → Minor 版本 (v1.0.x → v1.1.0)
- 其他提交 → Patch 版本 (v1.0.0 → v1.0.1)

### 2. Semantic Release (`semantic-release.yml`)

**高级语义化版本发布工作流**

- 📌 **触发时机**:
  - 代码推送到 `main` 或 `master` 分支
  - 手动触发（支持自定义版本号）
- 🎯 **版本策略**: 遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范
- 🚀 **功能**:
  - 智能分析提交类型
  - 生成详细的分类变更日志
  - 支持手动指定版本号
  - 支持预发布版本
  - 自动跳过没有语义化提交的发布

**提交类型识别**:

- `feat!:` / `BREAKING CHANGE:` → Major 版本
- `feat:` → Minor 版本
- `fix:` → Patch 版本
- `docs:` → 文档更新（变更日志单独分类）
- `chore:` → 构建/工具更新（变更日志单独分类）
- 其他类型会被分类但不触发版本更新

## 🎯 使用方法

### 方法一：通过提交信息自动触发

#### 使用 Auto Tag

只需将代码推送到 main 分支，工作流会自动运行：

```bash
git commit -m "feat: add new notification platform"
git push origin main
```

#### 使用 Semantic Release

使用规范的提交信息：

```bash
# 新功能 (Minor 版本)
git commit -m "feat(email): add HTML template support"

# Bug 修复 (Patch 版本)
git commit -m "fix(feishu): resolve webhook timeout issue"

# 破坏性变更 (Major 版本)
git commit -m "feat(api)!: redesign notification API"
# 或
git commit -m "feat(api): redesign notification API

BREAKING CHANGE: The old API is no longer supported"

# 文档更新 (不触发版本)
git commit -m "docs: update README with new examples"

# 工具更新 (不触发版本)
git commit -m "chore: update dependencies"

git push origin main
```

### 方法二：手动触发（仅 Semantic Release）

1. 进入 GitHub 仓库的 **Actions** 页面
2. 选择 **Semantic Release** 工作流
3. 点击 **Run workflow**
4. 填写参数：
   - **version**: 留空自动计算，或指定版本如 `v3.1.0`
   - **prerelease**: 是否为预发布版本
5. 点击 **Run workflow** 执行

### 方法三：命令行手动创建标签

如果你想手动创建标签：

```bash
# 创建带注释的标签
git tag -a v3.0.0 -m "Release v3.0.0

Features:
- Add new feature X
- Improve performance Y

Fixes:
- Fix bug Z"

# 推送标签到远程
git push origin v3.0.0

# 推送所有标签
git push origin --tags
```

## 📝 提交信息规范

### Conventional Commits 格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

**示例**:

```
feat(email): add attachment support

Implement attachment functionality for email platform.
Supports multiple file types including PDF, images, and documents.

Closes #123
```

### 提交类型 (type)

| 类型 | 说明 | 版本影响 |
|------|------|----------|
| `feat` / `feature` | 新功能 | Minor |
| `fix` | Bug 修复 | Patch |
| `feat!` / `fix!` | 破坏性变更 | Major |
| `docs` | 文档更新 | 无 |
| `style` | 代码格式（不影响代码运行） | 无 |
| `refactor` | 重构 | 无 |
| `perf` | 性能优化 | 无 |
| `test` | 测试相关 | 无 |
| `chore` | 构建/工具/依赖更新 | 无 |

### 作用域 (scope) - 可选

指定提交影响的范围，例如：

- `email` - 邮件平台
- `feishu` - 飞书平台
- `api` - API 接口
- `config` - 配置相关

### 破坏性变更标记

两种方式标记破坏性变更：

1. 在类型后加 `!`：

```
feat(api)!: redesign notification interface
```

2. 在 footer 中添加 `BREAKING CHANGE:`：

```
feat(api): redesign notification interface

BREAKING CHANGE: The old sendNotification method is removed.
Use the new send() method instead.
```

## 🔍 工作流执行结果

### 成功执行后

1. **创建 Git 标签**
   - 标签名: `v3.0.0`
   - 包含详细的变更日志

2. **创建 GitHub Release**
   - Release 标题: `Release v3.0.0`
   - Release 描述: 自动生成的变更日志
   - 包含对比链接

3. **GitHub Actions 摘要**
   - 显示版本信息
   - 显示提交统计
   - 显示完整变更日志

### 查看结果

1. **标签列表**: `https://github.com/你的用户名/notifyhub/tags`
2. **发布列表**: `https://github.com/你的用户名/notifyhub/releases`
3. **工作流运行**: `https://github.com/你的用户名/notifyhub/actions`

## ⚙️ 配置选项

### 修改触发分支

编辑 `.github/workflows/auto-tag.yml` 或 `semantic-release.yml`:

```yaml
on:
  push:
    branches:
      - main
      - master
      - production  # 添加其他分支
```

### 修改版本规则

在 `Determine version bump` 步骤中修改逻辑：

```bash
# 自定义规则
if echo "$COMMIT_MSG" | grep -qiE "^hotfix:"; then
  BUMP_TYPE="patch"
fi
```

### 自定义变更日志格式

修改 `Generate changelog` 步骤中的模板：

```bash
echo "## 🎉 What's New" > CHANGELOG.md
# 自定义格式...
```

## 🚨 注意事项

1. **权限要求**
   - 工作流需要 `contents: write` 权限来创建标签和发布
   - 已在工作流中配置，无需额外设置

2. **标签冲突**
   - 如果标签已存在，工作流会跳过并显示消息
   - 不会覆盖现有标签

3. **初始版本**
   - 如果仓库没有任何标签，会从 `v0.0.0` 开始
   - 首次运行会创建 `v0.0.1` 或 `v0.1.0` 或 `v1.0.0`（取决于提交类型）

4. **合并提交**
   - Merge commits 会被自动过滤
   - 只分析实际的功能提交

5. **预发布版本**
   - 仅 Semantic Release 支持
   - 通过手动触发时选择 `prerelease: true`

## 📖 示例场景

### 场景 1: 发布新功能

```bash
# 开发新功能
git checkout -b feature/user-groups
# ... 开发 ...

# 提交（使用规范格式）
git add .
git commit -m "feat(target): add user group resolution support"

# 合并到主分支
git checkout main
git merge feature/user-groups
git push origin main

# 🎉 自动创建 v1.1.0（Minor 版本）
```

### 场景 2: 修复紧急 Bug

```bash
# 修复 Bug
git checkout -b hotfix/email-timeout

# 提交修复
git add .
git commit -m "fix(email): resolve SMTP connection timeout

The connection timeout was too short for slow networks.
Increased to 30 seconds."

# 合并并推送
git checkout main
git merge hotfix/email-timeout
git push origin main

# 🎉 自动创建 v1.0.1（Patch 版本）
```

### 场景 3: 破坏性变更

```bash
# 重大 API 重构
git checkout -b refactor/api-v2

# 提交破坏性变更
git add .
git commit -m "feat(api)!: redesign notification client API

BREAKING CHANGE:
- Removed old Send() method
- New unified Send() with fluent interface
- Configuration structure changed

Migration guide available in MIGRATION.md"

# 合并并推送
git checkout main
git merge refactor/api-v2
git push origin main

# 🎉 自动创建 v2.0.0（Major 版本）
```

### 场景 4: 手动指定版本

```bash
# 重要里程碑版本
# 通过 GitHub UI 手动触发 Semantic Release
# 填入版本: v3.0.0
# 选择 prerelease: false

# 🎉 创建 v3.0.0
```

## 🔗 相关资源

- [Conventional Commits 规范](https://www.conventionalcommits.org/)
- [语义化版本规范](https://semver.org/lang/zh-CN/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Git 标签文档](https://git-scm.com/book/zh/v2/Git-基础-打标签)

## 🆘 故障排查

### 问题 1: 工作流没有触发

**检查**:

- 分支名称是否为 `main` 或 `master`
- 工作流文件是否在 `.github/workflows/` 目录
- GitHub Actions 是否已启用

### 问题 2: 标签创建失败

**检查**:

- 是否有权限问题
- 标签是否已存在
- 查看 Actions 日志获取详细错误信息

### 问题 3: 版本号不符合预期

**检查**:

- 提交信息是否符合规范格式
- 查看工作流日志中的 "Commit Analysis" 部分
- 确认使用的是正确的工作流

### 问题 4: Release 创建失败

**检查**:

- `GITHUB_TOKEN` 是否有足够权限
- 变更日志是否过大（GitHub API 限制）
- 网络连接是否正常

## 📧 支持

如有问题，请：

1. 查看 [GitHub Actions 运行日志](../../actions)
2. 查看 [Issues](../../issues)
3. 创建新的 Issue 描述问题
