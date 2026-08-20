# OpenList Desktop

<div align="center">
  <img src="./app-icon.png" alt="OpenList Desktop" width="128" height="128" />
  
  **跨平台的 OpenList 桌面应用程序，集成云存储功能**

  [![License](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](./LICENSE)
  [![Vue](https://img.shields.io/badge/Vue-3.5.17-green.svg)](https://vuejs.org/)
  [![Tauri](https://img.shields.io/badge/Tauri-2.6.0-orange.svg)](https://tauri.app/)
  [![Rust](https://img.shields.io/badge/Rust-2024-red.svg)](https://www.rust-lang.org/)
  
  [English](./README_en.md) | [中文](./README.md)
</div>

## 注意

该项目处于早期开发阶段，版本号0.x.x阶段，可能会有重大更改和不稳定的功能。

## 🔍 概述

OpenList Desktop 是一个跨平台桌面应用程序，为管理 OpenList 和通过 Rclone 进行本地挂载提供用户友好的界面。

该应用程序是一个全面的解决方案，用于：

- 管理 OpenList 文件管理服务
- 挂载和管理云存储（WebDAV）
- 监控服务运行状态
- 提供系统托盘集成以进行后台操作

## ✨ 功能特性

### 🚀 核心功能

- **OpenList 服务管理**：启动、停止和监控 OpenList 核心
- **本地挂载**：通过 Rclone 挂载至本地文件系统
- **实时监控**：跟踪服务状态、运行时间和性能指标
- **系统托盘**：带系统托盘通知的后台操作

### ⚙️ 管理功能

- **服务控制**：启动/停止/重启 OpenList 和 Rclone 服务
- **配置管理**：所有服务的基于 GUI 的配置
- **日志监控**：实时日志查看和管理
- **更新管理**：自动更新检查和安装
- **自动启动**：配置应用程序与软件同时启动

## 📸 应用截图

![主页仪表板](./screenshot/homepage.png)
![挂载管理](./screenshot/mountpage.png)
![日志监控](./screenshot/logpage.png)
![设置](./screenshot/settingpage.png)
![更新管理](./screenshot/update.png)

## 📦 安装

### 系统要求

- **操作系统**：Windows 10+、macOS 10.15+ 或 Linux（Ubuntu 18.04+）

### 下载选项

#### 1. GitHub 发布版（推荐）

从 [GitHub Releases](https://github.com/OpenListTeam/openlist-desktop/releases) 下载最新版本：

- **Windows**：`OpenList-Desktop_x.x.x_{arch}-setup.exe`
- **macOS**：`OpenList-Desktop_x.x.x_{arch}.dmg`
- **Linux**：`OpenList-Desktop_x.x.x_{arch}.deb` 或 `OpenList-Desktop_x.x.x_{arch}.rpm`

#### 2. 从源码构建

```bash
# 克隆仓库
git clone https://github.com/OpenListTeam/openlist-desktop.git
cd openlist-desktop

# 安装依赖
yarn install

# 准备开发环境
yarn run prebuild:dev

# 构建应用程序
yarn run build
yarn run tauri build
```

### 安装步骤

#### Windows

##### 使用安装程序

1. 下载 `.exe` 安装程序
2. 按照安装向导进行操作
3. 从开始菜单或桌面快捷方式启动

##### 使用Winget

```bash
winget install OpenListTeam.OpenListDesktop
```

#### macOS

1. 下载 `.dmg` 文件
2. 打开 DMG 并将 OpenList Desktop 拖到应用程序文件夹
3. 右键单击并选择"打开"（仅首次）
4. 在提示时授予必要权限

#### Linux

1. 下载 `.deb` 或 `.rpm` 包
2. 使用包管理器安装：

   ```bash
   sudo dpkg -i OpenList-Desktop_x.x.x_amd64.deb
   # 或者
   sudo rpm -i OpenList-Desktop_x.x.x_amd64.rpm
   ```

## 🚀 使用说明

### 首次启动

1. **初始设置**：首次启动时，应用程序将指导您完成初始配置
2. **服务安装**：在提示时安装 OpenList 服务
3. **存储配置**：配置您的第一个云存储连接

### 基本操作

#### 启动服务

```bash
仪表板 → 快速操作 → 启动 OpenList 核心
```

#### 添加云存储

1. 导航到 **挂载** 选项卡
2. 点击 **添加远程** 按钮
3. 配置存储设置：
   - **名称**：存储的唯一标识符
   - **类型**：存储提供商（WebDAV）
   - **URL**：存储端点 URL
   - **凭据**：用户名和密码
   - **挂载点**：本地目录路径
4. 点击 **保存** 和 **挂载**

#### 监控操作

- **服务状态**：检查仪表板上的服务健康指示器
- **日志**：使用日志选项卡监控系统操作
- **性能**：在仪表板上查看运行时间和响应指标

### 高级功能

#### 自定义 Rclone 标志

添加自定义 Rclone 标志以获得最佳性能：

- `--vfs-cache-mode=full`：启用完整 VFS 缓存
- `--buffer-size=256M`：增加缓冲区大小
- `--transfers=10`：并发传输限制

#### 系统托盘操作

- **右键单击托盘图标** 进行快速操作
- **双击** 显示/隐藏主窗口

## 🤝 贡献

我们欢迎社区贡献！

## 📄 许可证

本项目在 **GNU 通用公共许可证 v3.0** 下许可 - 详情请参见 [LICENSE](./LICENSE) 文件。

---

<div align="center">
  <p>由 OpenList 团队用 ❤️ 制作</p>
  <p>
    <a href="https://github.com/OpenListTeam/openlist-desktop">GitHub</a> •
    <a href="https://doc.oplist.org/">网站</a> •
    <a href="https://t.me/OpenListTeam">Telegram</a>
  </p>
</div>
