# Git 分支管理 - Makefile 命令

本项目在 Makefile 中添加了方便的 Git 分支管理命令，用于同步远程分支删除并清理本地陈旧分支。

## 🎯 可用命令

### 1. `make git-fetch` - 快速获取更新

**最基础的命令** - 从远程拉取最新数据并清理陈旧引用。

```bash
make git-fetch
```

**功能**:

- ✅ 从所有远程拉取最新数据（`git fetch --prune --all`）
- ✅ 清理已删除的远程分支引用
- ✅ 检测并提示陈旧分支数量

**示例输出**:

```
🔄 Fetching from remote and pruning stale references...
✅ Fetch complete

✅ No stale branches found
```

或者当有陈旧分支时：

```
🔄 Fetching from remote and pruning stale references...
✅ Fetch complete

⚠️  Found 2 local branch(es) tracking deleted remotes
💡 Run 'make git-clean-branches' to remove them
```

---

### 2. `make git-sync` - 完整同步

**最常用的命令** - 同步远程仓库并显示完整状态。

```bash
make git-sync
```

**功能**:

- ✅ 从所有远程拉取最新数据
- ✅ 清理已删除的远程分支引用
- ✅ 显示当前分支
- ✅ 显示所有本地分支及其跟踪状态
- ✅ 检测并提示陈旧分支（跟踪已删除的远程分支）
- ✅ 显示当前分支的领先/落后状态

**示例输出**:

```
🔄 Syncing with remote repository...

📊 Repository Status:
====================

📍 Current branch:
develop

📋 All local branches:
* develop      6cc5cf4 [origin/develop] Latest commit
  feature/v2   fa358c7 [origin/feature/v2: gone] Old feature
  feature/v3   6cc5cf4 [origin/feature/v3] Active feature

⚠️  Found 1 stale branch(es) tracking deleted remotes:
   - feature/v2

💡 Run 'make git-clean-branches' to clean them up

🔍 Tracking status: ## develop...origin/develop
```

---

### 3. `make git-prune` - 清理远程引用

仅清理远程分支引用，不删除本地分支。

```bash
make git-prune
```

**功能**:

- ✅ 拉取并清理远程分支引用
- ✅ 显示所有本地分支
- ✅ 列出跟踪已删除远程分支的本地分支

**适用场景**:

- 查看哪些本地分支跟踪的远程分支已被删除
- 在删除本地分支前先检查状态

---

### 4. `make git-clean-branches` - 交互式清理

**删除跟踪已删除远程分支的本地分支**（需要确认）。

```bash
make git-clean-branches
```

**功能**:

- ✅ 自动检测陈旧分支
- ✅ 显示将要删除的分支列表
- ✅ 要求用户确认（输入 y/N）
- ✅ 强制删除确认的分支（`git branch -D`）

**示例流程**:

```
🔍 Finding branches to clean up...

🗑️  Found stale branches (tracking deleted remotes):
   - feature/v2
   - old-experiment

Delete these branches? [y/N] y

Deleted branch feature/v2 (was fa358c7).
Deleted branch old-experiment (was 1234567).
✅ Stale branches deleted
```

**安全提示**:

- 该命令使用 `git branch -D`（强制删除）
- 确保重要更改已保存或推送
- 已合并的分支可以安全删除

---

### 5. `make git-show-merged` - 显示已合并分支

显示已合并到当前分支的其他分支。

```bash
make git-show-merged
```

**功能**:

- ✅ 列出已完全合并的本地分支
- ✅ 自动排除 main/master/develop 等主分支
- ✅ 不会删除任何分支，仅显示

**示例输出**:

```
🔍 Branches merged into current branch:
   feature/bug-fix-123
   feature/small-update

💡 These branches are fully merged and may be safe to delete
```

---

### 6. `make git-cleanup` - 综合清理

**交互式综合清理** - 结合已合并分支和陈旧分支。

```bash
make git-cleanup
```

**功能**:

- ✅ 首先显示已合并的分支
- ✅ 询问是否要查看陈旧分支
- ✅ 如果选择是，运行 `git-clean-branches`

**适用场景**:

- 定期清理本地分支
- 项目里程碑完成后的大清理

---

## 📖 使用场景

### 场景 1: 每日同步检查

**工作流开始时**，检查远程更新和分支状态：

```bash
make git-sync
```

如果发现陈旧分支：

```bash
make git-clean-branches
```

---

### 场景 2: PR 合并后清理

当你的 PR 被合并且远程分支被删除后：

```bash
# 1. 同步并查看状态
make git-sync

# 2. 清理陈旧分支
make git-clean-branches

# 输入 'y' 确认删除
```

---

### 场景 3: 项目里程碑后大清理

完成一个大版本或里程碑后：

```bash
# 显示已合并的分支
make git-show-merged

# 手动删除已合并的分支
git branch -d feature/已合并的分支

# 清理陈旧分支
make git-clean-branches

# 或者使用综合清理
make git-cleanup
```

---

### 场景 4: 仅查看不删除

想查看状态但不删除任何分支：

```bash
# 方式 1: 完整状态
make git-sync

# 方式 2: 只看已合并的
make git-show-merged

# 方式 3: 只看陈旧的
make git-prune
```

---

## 🔍 理解分支状态

### 什么是"陈旧分支"？

陈旧分支（Stale Branch）是指：

- 本地分支跟踪的远程分支已被删除
- 在 `git branch -vv` 输出中显示 `[origin/xxx: gone]`
- 通常发生在 PR 合并后，远程分支被删除

**示例**:

```bash
$ git branch -vv
  feature/v2  fa358c7 [origin/feature/v2: gone] Old commit
                                      ^^^^
                                      表示远程分支已删除
```

### 分支清理安全性

| 分支状态 | 删除命令 | 安全性 | 说明 |
|---------|---------|--------|------|
| 已合并 | `git branch -d` | ✅ 安全 | 必须完全合并才能删除 |
| 未合并 | `git branch -D` | ⚠️ 小心 | 强制删除，可能丢失更改 |
| 陈旧但未合并 | `git branch -D` | ⚠️ 检查 | 确保重要更改已保存 |

**Makefile 命令使用 `-D`（强制删除），使用前请确认！**

---

## 💡 最佳实践

### 1. 每日工作流

```bash
# 开始工作
make git-sync
git checkout -b feature/new-feature

# ... 开发 ...

# 提交前再次同步
make git-sync
```

### 2. PR 合并后

```bash
# 切换到主分支
git checkout main

# 拉取最新代码
git pull

# 清理陈旧分支
make git-clean-branches
```

### 3. 定期维护（每周/每月）

```bash
# 完整清理
make git-cleanup

# 或分步进行
make git-show-merged  # 查看已合并的
make git-sync         # 查看陈旧的
make git-clean-branches  # 确认后删除
```

### 4. 谨慎删除

**在删除前，确保**:

- ✅ 重要更改已提交
- ✅ 功能分支的工作已合并或推送
- ✅ 实验性分支的代码已备份（如果需要）

**检查未推送的提交**:

```bash
# 查看分支是否有未推送的提交
git log origin/feature/xxx..feature/xxx

# 如果有输出，说明有本地提交未推送
```

---

## 🛡️ 恢复误删的分支

如果不小心删除了重要分支：

### 方法 1: 使用 reflog（最近删除）

```bash
# 查看最近的操作
git reflog

# 找到删除前的提交 hash (例如 fa358c7)
git checkout -b feature/recovered fa358c7
```

### 方法 2: 查找悬空提交

```bash
# 列出所有悬空的提交
git fsck --lost-found

# 查看某个提交
git show <commit-hash>

# 恢复
git checkout -b recovered <commit-hash>
```

### 方法 3: 从远程恢复（如果曾推送）

```bash
# 如果远程还有（即使显示 gone）
git checkout -b feature/recovered origin/feature/xxx
```

---

## 📋 命令速查表

| 命令 | 用途 | 删除分支 | 需要确认 |
|------|------|---------|---------|
| `make git-fetch` | 快速获取更新 | ❌ | ❌ |
| `make git-sync` | 完整同步状态 | ❌ | ❌ |
| `make git-prune` | 清理远程引用 | ❌ | ❌ |
| `make git-clean-branches` | 删除陈旧分支 | ✅ | ✅ |
| `make git-show-merged` | 显示已合并分支 | ❌ | ❌ |
| `make git-cleanup` | 综合清理 | ✅ | ✅ |

---

## 🔗 相关 Git 命令

如果你想手动操作：

```bash
# 拉取并清理远程引用
git fetch --prune --all

# 查看所有分支及跟踪状态
git branch -vv

# 查看已合并的分支
git branch --merged

# 删除单个分支（安全）
git branch -d branch-name

# 删除单个分支（强制）
git branch -D branch-name

# 批量删除陈旧分支
git branch -vv | grep ': gone]' | awk '{print $1}' | xargs git branch -D
```

---

## 🆘 故障排查

### 问题 1: 命令没有找到陈旧分支

**检查**:

- 确保已运行 `git fetch --prune`
- 检查远程分支是否真的被删除：`git branch -r`

### 问题 2: 不能删除当前分支

**解决**:

```bash
# 切换到其他分支
git checkout main

# 然后运行清理命令
make git-clean-branches
```

### 问题 3: 权限错误

**检查**:

- 确保对仓库有写权限
- 检查分支是否被保护

### 问题 4: 误删重要分支

**立即恢复**:

```bash
# 查看最近操作
git reflog

# 恢复（使用正确的 hash）
git checkout -b recovered <commit-hash>
```

---

## 📚 延伸阅读

- [Git Branching - Branch Management](https://git-scm.com/book/en/v2/Git-Branching-Branch-Management)
- [Git Fetch --prune](https://git-scm.com/docs/git-fetch#Documentation/git-fetch.txt---prune)
- [Git Reflog](https://git-scm.com/docs/git-reflog)

---

**提示**: 将这些命令加入你的日常工作流，保持仓库清洁有序！ ✨
