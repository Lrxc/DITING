# 谛听 (DITING)

谛听（DITING）是一款专注于本地高性能 DNS 解析加速、智能调度优化与网络流量过滤的开源工具。致力于在设备本地实现低延迟解析、灵活规则阻断、防追踪与流量管控，全链路在本地闭环运行，无任何远程数据上传与隐私追踪。

> **当前仓库范围**：仅包含 **Android 客户端**。此前的 Windows 桌面端、Linux 服务端与开发者维护工具箱已从本仓库移除（历史代码仍可在 Git 历史中查阅）。

---

## 平台支持

| 模块 / 模式 | 最低系统要求 | 核心引擎与实现 | 核心特性与适用场景 |
|---|---|---|---|
| ⚡ **极速模式** (Express) | Android 7.0+ (API 24) | **纯 Kotlin 原生**<br>轻量窄路由 TUN | **超轻量、极低功耗**。仅捕获 DNS 流量（53 端口），其余直接走物理网络；提供核心解析加速、4 种调度模式、规则拦截与缓存。适合老旧设备或注重省电的用户。 |
| 🛡️ **普通模式** (Normal) | Android 10.0+ (API 29) | **Go 用户态协议栈**<br>(gVisor netstack + AAR) | **全功能网络管控**。全隧道接管，支持细粒度应用分流、可选 HTTPS 流量解密检查、出站代理联动、实时网速与流量排行。 |
| 🏠 **服务器模式** (Server) | Android 10.0+ (API 29) | **Go 独立 DNS 服务**<br>(监听 `0.0.0.0:1053`) | **局域网共享 DNS**。供同一局域网下的 PC、路由器或电视接入解析，共享本地规则与缓存，不占用 VPN 槽位。 |

- **硬件架构**：`arm64-v8a`
- **当前版本**：`1.2.11`（versionCode `25`）

---

## 核心特性

### 📡 DNS 解析与上游调度
- **多协议支持**：支持标准 DNS（UDP/TCP 53）、DNS-over-HTTPS（DoH）、DNS-over-TLS（DoT）与 DNS-over-QUIC（DoQ）。
- **多策略调度**：提供**单一服务**（Single）、**智能优选**（Smart EWMA 延迟预测与成功率考量）、**最快响应**（Fastest 并发竞速）及**依次尝试**（Sequential 备援容灾）四种调度模式。
- **服务商管理**：内置阿里、腾讯、Cloudflare、Google、DNSPod 等主流公共 DNS，支持自由扩展与编辑自定义节点。
- **Bootstrap 引导与防绕过**：内置与自定义 Bootstrap IP 负责解析加密上游域名，杜绝递归死锁；启用 DDR 防绕过机制（拦截 `_dns.resolver.arpa`），引导流量规范经由本地通道解析。

### ⚡ 智能缓存与容灾防击穿
- **并发安全分片缓存**：采用并发安全 LRU 缓存，提供跟随 TTL、平衡、高命中等多档预设策略，兼顾解析实时性与响应效率。
- **乐观容灾（Stale-While-Revalidate）**：在上游超时或网络抖动时，短暂复用仍处于宽限期内的过期缓存，保障弱网基本可用。
- **并发合并防击穿（Singleflight）**：针对突发同一域名的并发解析请求进行请求合并，减轻上游负载并防止本地缓存击穿。

### 🛡️ 规则引擎与地址覆写
- **高性能多维度匹配**：支持域名黑白名单（兼容 AdGuard 语法、通配符与正则）、Trie 前缀树与 Bloom 过滤器加速、网页元素隐藏（Cosmetic）规则。
- **规则订阅与自动更新**：支持 AdGuard 格式远程规则订阅，支持规则分组、镜像源模板加速（如 GitHub 镜像代理）与后台定时自动拉取更新。
- **静态覆写与重定向**：支持 IPv4 / IPv6 静态地址覆写（A / AAAA 记录）与 CNAME 重定向解析。
- **多样化阻断响应**：支持零地址（`0.0.0.0` / `::`）、NXDOMAIN、NODATA 与 REFUSED 四种拦截响应行为。

### 📱 客户端专属特性
- **细粒度应用分流**：支持排除应用（Bypass 直连物理网络）、禁止联网（丢弃全部外联）与应用白名单访问（仅限访问已解析 IP）。
- **可选 HTTPS 流量检查**：基于 gVisor netstack 与本地 CA 根证书，仅对用户勾选的应用解密与 URL 级过滤；支持证书绑定自适应旁路与 QUIC/H3 引导回退。
- **出站代理联动**：支持将流量转接至本地 SOCKS5（含 UDP ASSOCIATE）或 HTTP CONNECT 代理。
- **全景洞察与诊断工具**：支持悬浮窗与状态栏实时网速、按 UID 流量消耗统计排行、Ping 延迟 / Traceroute / DNS 手动解析诊断与快捷设置磁贴（Quick Settings Tile）。
- **个性化与智能化**：支持 Material 3 动态取色、深浅主题与自定义背景；支持配置导入导出；集成可选的私有 LLM API 助手（BYOK，默认关闭，仅用于域名分析与研判）。

---

## 项目工程结构

```
DITING/
├── Android/                       # Android 客户端工程
│   ├── app/                       # 应用层源码（Compose UI、Room 数据库、极速模式、服务器模式）
│   │   ├── src/main/java/com/haoze/diting/
│   │   │   ├── express/           # 极速模式：纯 Kotlin 窄路由 TUN 与 DNS 数据面
│   │   │   ├── normal/            # 普通模式：VpnService + Go 全隧道
│   │   │   ├── server/            # 服务器模式：局域网 DNS 服务
│   │   │   ├── core/              # DNS 工具、规则解析、缓存、日志、诊断
│   │   │   ├── data/              # Room 实体、DAO、仓库
│   │   │   └── ui/                # Compose 界面、设置、主题、本地化
│   │   ├── libs/tunnel.aar        # Go 隧道模块产物（gomobile bind）
│   │   └── src/test/              # 单元测试
│   ├── tunnel/                    # Go 用户态网络栈与 DNS 隧道模块源码（编译为 tunnel.aar）
│   ├── gradle/                    # 版本目录与 Wrapper
│   └── build_apk.bat              # 交互式构建、版本归档与 ADB 安装脚本
├── docs/                          # 项目技术规范与维护文档
│   ├── development/               # 架构文档（AAR 构建、名单云控）
│   │   └── Archive/               # 历史设计方案与归档技术规范
│   └── assets/                    # 文档静态资源（收款码图片等）
├── avatars/                       # 赞助者与共建者头像资源（云控展示）
├── recognition_members.json       # 赞助者与共建者名单配置（云端热更新）
└── AGENTS.md                      # 项目开发与代码规范定义
```

> `avatars/` 与 `recognition_members.json` 由客户端通过 GitHub Raw 云端拉取，用于「关于」页面的赞助者与共建者名单展示。

---

## 环境要求与构建指南

### 运行与构建要求
- **系统要求**：极速模式 Android 7.0+ (API 24+)；普通模式 / 服务器模式 Android 10.0+ (API 29+)。
- **硬件架构**：`arm64-v8a`。
- **构建环境**：JDK 17 或更高版本（AGP 9.x 要求；推荐使用 Android Studio 自带 JBR）、Android SDK（compileSdk 36）。构建 Go 隧道模块另需 Go 1.23+、`gomobile` 与 Android NDK。

### 构建命令
在 `Android/` 目录下使用 Gradle Wrapper：

```bash
cd Android

# 编译 Debug APK
./gradlew :app:assembleDebug --console=plain    # Windows 下使用 gradlew.bat

# 编译 Release APK
./gradlew :app:assembleRelease --console=plain

# 运行单元测试
./gradlew :app:testDebugUnitTest --console=plain
```

亦可在 Windows 下直接运行 `Android/build_apk.bat`：支持一键选择编译模式、自动归档产物，并可通过 ADB（支持无线调试）快速部署至真机。

构建产物路径：
- 标准 APK：`Android/app/build/outputs/apk/<buildType>/app-<buildType>.apk`
- 归档 APK：`Android/app/build/outputs/apk/versioned/<buildType>/DITING-<buildType>-v<versionName>.apk`

### 重新生成 Go 隧道模块（`tunnel.aar`）
修改 `Android/tunnel/` 下的 Go 源码后，需要重新编译 AAR 才能生效：

```bash
cd Android/tunnel
gomobile bind -target=android/arm64 -androidapi 24 -trimpath -ldflags="-s -w" -o ../app/libs/tunnel.aar .
```

详细的环境变量、NDK 配置与常见问题见 [`docs/development/aar-build-notes.md`](docs/development/aar-build-notes.md)。

### 签名配置
在 `Android/keystore.properties`（或环境变量 `DITING_KEYSTORE_FILE` / `DITING_KEYSTORE_PASSWORD` / `DITING_KEY_ALIAS` / `DITING_KEY_PASSWORD`）中提供签名信息，示例见 `Android/keystore.properties.example`。密钥文件不入库。

---

## 技术栈一览

- **客户端**：Kotlin / Jetpack Compose / Material 3 / Coroutines & Flow / Room / DataStore / VpnService / OkHttp / WorkManager
- **网络内核**：Go / gVisor netstack / miekg/dns / quic-go / Singleflight / gomobile
- **极速模式数据面**：纯 Kotlin 窄路由 TUN + DNS 报文编解码
- **测试**：JUnit（Kotlin 单元测试）+ Go `testing`

---

## 开发与参考文档

- [Go AAR 构建记录](docs/development/aar-build-notes.md) — `tunnel.aar` 的编译环境、构建参数与交叉编译细节。
- [云控贡献者名单维护说明](docs/development/recognition-members.md) — 赞助者与共建者名单的云控机制、JSON 规范与头像维护。
- [极速模式架构设计与技术规范（归档）](docs/development/Archive/express-mode-architecture.md) — 窄路由数据面、DNS 劫持清单与取舍。
- [极速模式分阶段实施计划（归档）](docs/development/Archive/express-mode-plan.md)
- [AdGuard 规则修饰符语法参考（归档）](docs/development/Archive/adguard-rule-modifier-syntax-reference.md)
- [Android 签名证书管理与发布签名规范（归档）](docs/development/Archive/android-signing-certificate-management.md)

代码规范见 [`AGENTS.md`](AGENTS.md)：单个代码文件不得超过 600 行；提交信息使用中文并遵循 Conventional Commits。

---

<a id="sponsorship"></a>

## 赞助支持

如果您觉得谛听对您有所帮助，欢迎赞助支持本项目的开发与维护。付款时请备注您的昵称，以便同步加入项目赞助者名单。

| 支付宝付款码 | 微信付款码 |
| :---: | :---: |
| ![支付宝付款码](docs/assets/alipay_code.png) | ![微信付款码](docs/assets/wechatpay_code.png) |

关于赞助者与共建者名单的动态展示机制与头像提交规范，请参阅 [云控贡献者名单维护说明](docs/development/recognition-members.md)。

---

## 作者与开源协议

- **作者**：[haoze-evolluling](https://github.com/haoze-evolluling)
- **开源协议**：本项目基于 [GNU General Public License v3.0 (GPL-3.0)](https://www.gnu.org/licenses/gpl-3.0.html) 协议开源。
