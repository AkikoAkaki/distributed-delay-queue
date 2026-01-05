# Development Setup Guide (Windows 11)

## 1. Prerequisites (前置要求)

确保你的 Windows 开发环境安装了以下核心工具。

### 1.1 Go Language
- **Version:** Go 1.21+
- **Check:** `go version`
- **Env Config:** 确保 `%USERPROFILE%\go\bin` 添加到了系统的 `PATH` 环境变量中。

### 1.2 Docker Desktop (WSL 2 Backend)
- **Settings:** 在 Docker Desktop 设置中，确保勾选 "Use the WSL 2 based engine"。
- **Why:** WSL 2 文件读写性能远高于 Hyper-V，对于数据库操作至关重要。

### 1.3 Make Tool (Windows)
Windows 默认没有 `make` 命令。为了统一构建脚本，推荐使用 Chocolatey 或 Scoop 安装。
- **Option A (Chocolatey):** 以管理员身份打开 PowerShell 运行 `choco install make`
- **Option B (Scoop):** 运行 `scoop install make`
- **Verification:** 终端输入 `make --version`

---

## 2. Project Configuration (配置说明)

### 2.1 配置文件
项目默认不提交敏感配置。请基于模板创建本地配置：

```powershell
# Windows PowerShell
Copy-Item config/config.example.yaml configs/config.yaml
```

- config/config.yaml 已被加入 .gitignore，可以在本地随意修改（如 Redis 密码或端口），不会影响远程仓库。

### 2.2 基础设施 (Infrastructure)

使用 Docker Compose 启动本地依赖（Redis）：

```bash
make up
# 等同于: docker-compose -f deployments/docker-compose.yaml up -d
```
- Redis 地址: localhost:6379
- 数据持久化: 挂载于 Docker Volume redis_data，容器重启数据不丢失。

---

## 3. Workflow (日常工作流)

| Command | Description |
|---------------|-------------|
| make up | 启动 Redis 容器 |
| make down | 停止并移除容器 |
| make run-server | 启动 API Server (需先 make up) |
| make run-worker | 启动 Worker (需先 make up) |
| make test | 运行所有单元测试 |

---

## Troubleshooting (常见问题)

**Q: make command not found?**

A: 请参考 1.3 节安装 make，或者直接运行 Makefile 中对应的命令（如 `go run cmd/server/main.go`）。

**Q: Git 提示 LF will be replaced by CRLF?**

A: 建议在项目中强制使用 LF 换行符（Go 标准）。运行：

```bash
git config --global core.autocrlf input
```
