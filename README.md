# BESTSUB



![Version](https://img.shields.io/badge/Status-开发中-blue)
[![GitHub stars](https://img.shields.io/github/stars/bestruirui/BestSub.svg?style=social&label=Star)](https://github.com/bestruirui/BestSub)

## 📜 旧版 (Legacy Version)

> **[点击这里](https://github.com/bestruirui/BestSub/blob/master/README_zh.md)** 跳转至经典命令行版本
>
> **[点击这里](https://github.com/bestruirui/BestSub/releases/latest)** 下载稳定的命令行版本的应用程序
>
> 旧版已趋于成熟，功能完善且运行稳定

## 📢 征集图标

- 要求：SVG格式

## 🚀 功能规划

### ✅ 核心功能

| 状态 | 功能 | 描述 |
|------|------|------|
| 🔄 | **自定义前端** | 开放api[文档](https://bestsub.apifox.cn)，后端使用标准openapi格式 |
| 🔄 | **插件系统** | 扩展性强的插件架构 |
| 🔄 | **订阅管理** | 全面的订阅配置与控制 |
| 🔄 | **节点检测** | 可靠的节点状态监测 |
| 🔄 | **节点重命名** | 灵活的节点命名管理 |
| 🔄 | **通知系统** | 多渠道的提醒服务 |


### 🔌 插件系统

- 订阅保存
- 节点检测
- 节点重命名
- 自定义通知渠道

### 📋 订阅管理

- 根据节点存活数量自动管理订阅状态，多次没有存活节点后，禁用或者删除此订阅
- 每个订阅可自定义 `cron` 获取时间
- 自定义检测项目
- 自动识别订阅类型：`mihomo`/`base64`/`v2ray`
- 可用节点缓存
- 多种订阅导出选项：
  - 永久分享链接
  - 带过期信息的临时分享链接
  - 内置云端保存（Gist、WebDAV等）
  - 支持通过插件自定义导出方式

### 🔍 节点检测

- 节点去重
  - 严格去重：落地IP相同则视为同一节点
  - 宽松去重：不判断落地IP，仅通过配置文件去重
- 通过插件形式自定义检测项目
- 内置流媒体、OpenAI等服务检测

### ✏️ 节点重命名

- 前端可视化预览重命名效果
- 自定义重命名模板
- 支持自定义重命名插件

### 📱 通知系统

- 内置主流通知渠道
- 支持自定义通知插件

### 💫 未来展望

- 多用户支持
- 节点缓存功能，按国家缓存，保证每个地区都至少有一个可用节点

## ❤️ 支持项目

点亮 Star ⭐ 来支持项目开发！