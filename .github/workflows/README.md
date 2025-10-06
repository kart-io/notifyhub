# GitHub Workflows

本目录包含项目的 GitHub Actions 工作流配置。

## 📋 可用工作流

### 1. CI (`ci.yml`)

**持续集成工作流**

- 运行测试套件（Go 1.21 & 1.22）
- 代码质量检查（vet、fmt、golangci-lint）
- 安全扫描（gosec）
- 性能基准测试
- 覆盖率报告（上传到 Codecov）

**触发**: PR 和推送到 main/develop 分支

**包含的检查**:

- ✅ 依赖验证
- ✅ 代码格式检查
- ✅ 静态分析
- ✅ 单元测试（带竞态检测）
- ✅ 性能基准测试
- ✅ 安全扫描
- ✅ 代码构建

---

### 2. Auto Tag (`auto-tag.yml`)

**简单自动标签**

- 自动计算版本号
- 创建 Git 标签
- 生成简单变更日志
- 创建 GitHub Release

**触发**:

- 自动: 推送到 main/master 分支
- 手动: 通过 workflow_dispatch

**版本规则**:

- `feat!:` 或 `BREAKING CHANGE:` → Major (v2.0.0)
- `feat:` 或 `feature:` → Minor (v1.1.0)
- 其他提交 → Patch (v1.0.1)

---

### 3. Semantic Release (`semantic-release.yml`)

**高级语义化版本发布**

- 智能提交分析和分类
- 详细分类变更日志（Breaking/Features/Fixes/Docs等）
- 支持手动指定版本号
- 支持预发布版本
- 自动跳过无意义发布

**触发**:

- 自动: 推送到 main/master 分支
- 手动: GitHub UI 触发

**提交规范**:

```bash
feat: 新功能 → Minor
fix: 修复 → Patch
feat!: 破坏性变更 → Major
docs/chore/style/refactor: 分类但不触发版本
```

**特色功能**:

- 📊 提交统计（Breaking/Features/Fixes）
- 📝 详细分类的变更日志
- 🔗 自动生成对比链接
- ⏭️ 智能跳过无语义化提交

---

## 🚀 快速开始

### 自动打标签

1. **简单使用**:

```bash
git commit -m "feat: add new feature"
git push origin main
# ✅ 自动创建 v1.1.0
```

2. **破坏性变更**:

```bash
git commit -m "feat(api)!: redesign API"
git push origin main
# ✅ 自动创建 v2.0.0
```

3. **Bug 修复**:

```bash
git commit -m "fix: resolve timeout issue"
git push origin main
# ✅ 自动创建 v1.0.1
```

### 手动触发发布

1. 进入 **Actions** → **Semantic Release**
2. 点击 **Run workflow**
3. 填写参数（可选）:
   - `version`: v3.0.0（留空自动计算）
   - `prerelease`: true/false
4. 点击 **Run workflow**

## 📖 详细文档

完整的使用指南和最佳实践，请查看:

- [自动打标签使用指南](../docs/AUTO_TAG_GUIDE.md)

## 🔧 提交信息规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>
```

**类型**:

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档
- `style`: 格式
- `refactor`: 重构
- `perf`: 性能
- `test`: 测试
- `chore`: 构建/工具

**示例**:

```bash
feat(email): add HTML template support
fix(feishu): resolve webhook timeout
docs: update API documentation
chore: update dependencies
```

## ⚙️ 权限配置

所有工作流已配置必要权限:

- `contents: write` - 创建标签和发布
- `pull-requests: read` - 读取 PR 信息

## 🔗 相关资源

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
