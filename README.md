# FileRenaming

文件重命名工具 - 基于 Wails + Vue 3 + Naive UI 开发的跨平台桌面应用。

## 关于项目

这是一个使用 Wails 框架构建的桌面应用程序，提供了便捷的文件重命名功能。

## 技术栈

- **后端**: Go 1.23 + Wails v2
- **前端**: Vue 3 + Naive UI + UnoCSS
- **包管理**: pnpm

## 开发环境设置

### 前置要求

- Go 1.23 或更高版本
- Node.js 20 或更高版本
- pnpm
- Wails CLI

### 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 安装前端依赖

```bash
cd frontend
pnpm install
```

## 开发模式

在项目目录下运行 `wails dev` 启动开发模式。这将启动一个 Vite 开发服务器，提供快速的前端热重载。

如果你想在浏览器中开发并访问 Go 方法，还有一个运行在 http://localhost:34115 的开发服务器。在浏览器中连接到此地址，你可以在开发者工具中调用 Go 代码。

```bash
wails dev
```

## 构建

### 本地构建

要构建可重新分发的生产模式包，使用 `wails build`：

```bash
# 构建当前平台
wails build

# 构建 Windows 版本
wails build -platform windows/amd64

# 构建 macOS 版本
wails build -platform darwin/amd64
```

### 自动构建（CI/CD）

项目配置了 GitHub Actions 工作流，在以下情况下会自动构建：

- 推送到 `main` 分支
- 创建 Pull Request 到 `main` 分支
- 手动触发工作流

工作流会自动构建 Windows 和 macOS 版本，并将构建产物上传为 Artifacts。

#### 获取构建产物

1. 前往 GitHub 仓库的 Actions 页面
2. 选择最新的工作流运行
3. 在 Artifacts 部分下载对应平台的构建产物：
   - `FileRenaming-windows`: Windows 版本（.exe 文件）
   - `FileRenaming-macos`: macOS 版本（可执行文件）

#### 创建 Release

当创建 Git 标签时，工作流会自动创建 Release 并上传构建产物：

```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

## 项目配置

可以通过编辑 `wails.json` 来配置项目。更多关于项目设置的信息可以在这里找到：
https://wails.io/docs/reference/project-config

## 作者

- **BluerAngala** - 1107117864@qq.com
