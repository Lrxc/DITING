# 谛听 (DITING) Linux 端移植说明与平台差异对照

## 1. 目标与交付形态

本次移植在既有 `Windows/` Go 模块（`github.com/haoze-evolluling/diting/windows`）内实现 Linux 支持，交付形态为：

- **核心服务 `diting-service`**：单一二进制，内嵌 Vue 3 Web 控制台，由 systemd 托管。
- **DNS 监听**：默认 `127.0.0.1:53` 与 `[::1]:53`，可切换局域网模式监听 `0.0.0.0:53` / `[::]:53`。
- **Web 管理控制台**：默认 `:15353`，带管理员鉴权与 REST/WS 接口。
- **桌面 GUI**：Linux 端暂不提供 Wails 原生窗口（延后为独立阶段），`main_linux.go` 仅作入口提示。

选择该形态的原因：Linux 服务器/容器环境通常无桌面会话与 WebKit 运行时，服务 + 浏览器控制台可覆盖绝大多数使用场景，且仍完整复用 DNS 内核、规则引擎、缓存与 IPC 机制。

---

## 2. 代码组织：单模块 + 构建标签分层

Go 单模块通过 `//go:build` 约束实现跨平台分层，避免导入环：

```
Windows/internal/platform/            # 中性层：DTO、接口、纯逻辑（无构建标签）
├── types.go            AdapterInfo / AdapterState / TakeoverState / PortConflict / ...
├── interfaces.go       CommandExecutor / AdapterScanner / DNSManager / StateStore / PortChecker / ...
├── netutil.go          IsVirtualAdapter / HasValidIPv4Gateway / GetLANAddresses / IsPortBindConflict / ...
├── ports.go            TestUDPBind / TestTCPBind / AreAddressesAvailable
├── state.go            FileStateStore（原子写入 + CheckAndSelfHeal 自愈流程）
├── paths.go            DefaultDataDir / DefaultConfigPath / DefaultFilterDir（支持 DITING_DATA_DIR）
├── paths_{windows,linux}.go        平台专有默认目录
├── env_{windows,linux}.go          lookupEnv / isRoot
├── windows/            # 仅保留 Win32 专有实现（SCM / netsh / iphlpapi / dnsapi / UAC）
└── linux/              # Linux 专有实现（resolvectl / nmcli / /proc / systemd）
```

**依赖规则**（务必遵守，否则会产生导入环）：

- `internal/platform` 不导入任何项目内其它包；
- `internal/platform/windows` 与 `internal/platform/linux` 只导入 `internal/platform`；
- 消费者（`internal/ipc`、`cmd/service`、根包）只导入 `internal/platform`；
- 平台实现的"选择"收敛在 `cmd/service/platform_{windows,linux}.go` 两个工厂文件中，共享代码不含 `runtime.GOOS` 分支。

> 说明：`internal/core` 的 `DefaultFilterConfig()` 直接调用 `platform.DefaultFilterDir()`（core → platform 单向依赖）。若希望内核零平台依赖，可改为默认 `DataDir` 为空、由 `config` 层在加载时填充。

---

## 3. Windows ↔ Linux 架构差异对照

| 维度 | Windows | Linux |
|---|---|---|
| 服务托管 | Windows Service（SCM，`kardianos/service`） | systemd 单元（`kardianos/service` / 自写单元） |
| 服务名 | `DitingDNSService` | `diting-dnsservice` |
| 提权 | UAC 按需提权（`RunElevated`，ShellExecute runas） | 无按需提权，进程启动即固定权限（要求 root） |
| 权限判定 | `HaveAdminRights()`（Token 检查） | `os.Geteuid() == 0` |
| 网卡标识 | 适配器 GUID | 接口名（`eth0`、`wlan0`） |
| 网卡枚举 | `iphlpapi` (GetAdaptersAddresses) | `net.Interfaces()` + `/sys/class/net` + `/proc/net/route` |
| 网卡描述 | 驱动友好名 | `/sys/class/net/<n>/device/uevent` 的 `DRIVER=` |
| DNS 配置读取 | `dnsapi` / 注册表 | 后端命令（`resolvectl` / `nmcli`）或 `/etc/resolv.conf` |
| DNS 接管 | `netsh interface ip[6] set dnsservers` | `resolvectl dns` / `nmcli con mod` / 重写 `resolv.conf` |
| DNS 缓存刷新 | `ipconfig /flushdns`（或 dnsapi 原生） | `resolvectl flush-caches` |
| 53 端口占用识别 | `netsh` + `GetExtendedTcp/UdpTable`（PID 反查） | `/proc/net/{tcp,tcp6,udp,udp6}` + `/proc/<pid>/fd` inode 反查 |
| 系统级占用 | Windows ICS (SharedAccess) | systemd-resolved stub（`127.0.0.53` / `127.0.0.54`） |
| 释放系统级占用 | 停止/禁用 SharedAccess 服务 | 写入 `DNSStubListener=no` drop-in 并重启 resolved |
| 防火墙 | `netsh advfirewall`（命名规则） | `firewalld` / `ufw` / `nftables` / `iptables` 自动探测 |
| 配置目录 | `%ProgramData%\DITING` | `/etc/diting` |
| 数据目录 | `%ProgramData%\DITING` | `/var/lib/diting` |
| 默认路径写入 | 注册表 / ProgramData | 需 root；非 root 时退避 `$HOME/.diting` |
| 桌面 GUI | Wails v2 + WebView2 原生窗口 | 暂不提供（延后阶段），仅 Web 控制台 |
| 窗口截图 / 宽高比锁定 | PrintWindow / Win32 subclass | 不适用（无原生窗口） |

---

## 4. DNS 接管后端与还原机制

### 4.1 后端自动探测优先级

`detectBackend()` 按以下顺序选择后端（探测只读取运行态，不修改系统配置）：

1. **systemd-resolved**：`resolvectl status` 成功且输出非空；
2. **NetworkManager**：`nmcli -t -f RUNNING general` 返回 `running`；
3. **`/etc/resolv.conf`**：兜底（容器、Alpine、极简服务器）。

### 4.2 各后端接管与还原

| 后端 | 接管（Apply） | 还原（Restore） | 刷新缓存 |
|---|---|---|---|
| systemd-resolved | `resolvectl dns <iface> 127.0.0.1 ::1` + `resolvectl domain <iface> ~.` | 按快照 `resolvectl dns <iface> <servers...>`；快照为空（原为自动获取）时 `resolvectl revert <iface>` | `resolvectl flush-caches` |
| NetworkManager | `nmcli con mod <conn> ipv4.dns 127.0.0.1 ipv6.dns ::1 ipv[46].ignore-auto-dns yes` + `nmcli con up <conn>` | 按快照恢复 `ipv[46].dns` 并还原 `ignore-auto-dns` | `resolvectl flush-caches`（存在时） |
| resolv.conf | 备份原文件后原子重写为 `nameserver 127.0.0.1` / `::1` | 从备份写回；无备份则回退公共解析器 | `resolvectl flush-caches`（存在时） |

### 4.3 安全自愈流程

1. **接管前**：`stateStore.Save(&TakeoverState{...})` 原子落盘快照到 `dns_state.json`，并生成离线 `restore-dns.sh`；
2. **接管失败**：立即 `Restore` 已生效部分并 `Clear` 状态，保证不残留半成品；
3. **正常退出**：`Stop()` 中若接管处于活动状态则先 `Restore` 再关闭；
4. **崩溃残留**：下次启动 `CheckAndSelfHeal` 读取 `dns_state.json`，若 `active=true` 且进程已不存在则自动还原；
5. **残留兜底**：`Restore` 结束时无条件执行 `ResetResidualLoopbackDNS`，扫描所有网卡清理指向 `127.0.0.0/8`、`::1` 的残留 DNS。

> `/etc/resolv.conf` 为软链（systemd-resolved / NetworkManager 托管）时，写入前会先解除链接并备份软链目标，避免写入穿透系统托管文件。

### 4.4 离线还原脚本 `restore-dns.sh`

由 `GenerateRestoreScript` 按探测到的后端生成，位于数据目录（默认 `/var/lib/diting/restore-dns.sh`，`DITING_DATA_DIR` 下为 `<dir>/DITING/restore-dns.sh`）。即使谛听服务无法启动，也可 `sudo sh restore-dns.sh` 独立恢复系统 DNS。

---

## 5. 53 端口冲突与 systemd-resolved stub

`LinuxPortChecker` 遵循 AdGuard Home 的 **Real Socket Check** 原则：

1. 先真实尝试绑定目标 UDP/TCP 地址（`platform.AreAddressesAvailable`）判断可用性；
2. 再解析 `/proc/net/*` 定位占用 53 的套接字，经 `/proc/<pid>/fd`（`socket:[inode]` 反查）映射到进程，读取 `/proc/<pid>/comm` 得进程名；
3. 特别识别 `127.0.0.53` / `127.0.0.54` 上的监听或 `systemd-resolve(d)` 进程，标记为系统级占用（`IsICS=true`，诊断中称为 stub），`CanAutofix=true`。

**一键自动修复**（`AutofixPort53ForAddresses`）仅处理系统级占用：写入 `/etc/systemd/resolved.conf.d/diting.conf`：

```ini
[Resolve]
DNSStubListener=no
```

随后 `systemctl restart systemd-resolved`（无 systemd 单元时回退为向 stub 进程发送 `SIGTERM`）。第三方程序占用不自动处理，返回明确的手动排查建议。

---

## 6. `DITING_DATA_DIR` 调试用法

设置 `DITING_DATA_DIR` 后，配置目录与数据目录统一收敛至 `${DITING_DATA_DIR}/DITING`，无需写入系统目录即可运行：

```bash
sudo DITING_DATA_DIR=/tmp/diting ./build/bin/diting-service -run -port 5353
```

- 状态快照：`/tmp/diting/DITING/dns_state.json`
- 离线还原脚本：`/tmp/diting/DITING/restore-dns.sh`
- 规则源缓存：`/tmp/diting/DITING/filters`

非 root 环境下，Linux 默认目录依次退避为 `$HOME/.diting`（`paths_linux.go`）。注意：非 root 无法绑定 53 端口或写入 `/etc`，此时 `HaveAdminRights()` 会返回 false 并给出可读错误，而非静默失败。

---

## 7. 与 Windows 端的功能差异清单

| 能力 | Windows | Linux | 说明 |
|---|---|---|---|
| 原生桌面 GUI | ✅ Wails | ❌ | Linux Wails GUI 延后（需 GTK/WebKit2GTK） |
| Web 控制台 | ✅ | ✅ | 功能一致 |
| 物理网卡双栈 DNS 接管 | ✅ | ✅ | 后端不同，语义一致 |
| 崩溃自愈 / 应急还原 | ✅ | ✅ | 一致，另附离线脚本 |
| 53 端口冲突检测与识别 | ✅ | ✅ | 系统级占用从 ICS 换成 resolved stub |
| 一键自动修复 | ✅ 停用 ICS | ✅ 关闭 stub | — |
| LAN 局域网 DNS | ✅ | ✅ | 防火墙后端自动探测，无防火墙时降级提示 |
| 开机自启 | ✅ 注册表/服务 | ✅ systemd enable | — |
| UAC / 提权弹窗 | ✅ | ❌ | Linux 无按需提权，需以 root 启动 |
| 界面截图 / 窗口宽高比 | ✅ | ❌ | 无原生窗口，不适用 |
| ICS 概念 | ✅ | ❌ | Linux 无 ICS |

---

## 8. 构建与验证

```bash
# 双端编译
cd Windows
go build ./...                             # Windows
GOOS=linux GOARCH=amd64 go build ./...     # Linux 交叉编译

# 双端测试（Linux 侧建议在 WSL 内执行）
go test ./...
GOOS=linux go test ./...                   # 需在 Linux 环境；仅纯逻辑可跨编译运行
```

- Linux 单二进制构建：`./build_linux.sh`（支持 `--package` 生成 `.deb` / `.tar.gz`）。
- 维护工具箱：构建目标选择 `linux`，架构可选 `amd64` / `arm64` / `arm`。
- systemd 安装：`Windows/build/linux/install.sh` / `uninstall.sh`。

### WSL 实测要点（本次验证记录）

- WSL2 中 `53` 由 `systemd-resolved` 的 `127.0.0.53` / `127.0.0.54` stub 占用，`/etc/resolv.conf` 指向 `10.255.255.254` 且由 WSL 生成；服务可正常绑定 `127.0.0.1:53` / `[::1]:53`（不与 stub 地址冲突）。
- 验证顺序：启动服务 → `curl /api/v1/health`、`/api/v1/status` → UDP 查询命中缓存 → `POST /api/v1/takeover/enable` 写入 `resolvectl dns eth0 127.0.0.1 ::1` → `kill -9` 模拟崩溃 → 重启触发 `CheckAndSelfHeal` 自动还原 → `POST /api/v1/takeover/disable` 正常还原。
- 回退手段：若 WSL 网络异常，`wsl --shutdown` 后重进会由 WSL 重新生成 `/etc/resolv.conf`。

---

## 9. 风险与注意事项

- **WSL 网络易断**：WSL2 的 `/etc/resolv.conf` 由 WSL 生成（`/mnt/wsl/resolv.conf`），接管测试务必在临时 `DITING_DATA_DIR` 下进行并备好还原脚本。
- **systemd-resolved stub 占 53**：`DNSStubListener=no` 属侵入性操作，UI 上必须给出明确确认提示；卸载后需手动删除 drop-in 并重启 resolved。
- **无提权弹窗**：服务必须由 root/systemd 启动，非 root 运行需给出可读错误。
- **空防火墙后端**：无 `ufw`/`firewalld`/`nftables`/`iptables` 时 `ConfigureFirewall*` 走降级分支返回诊断提示，不阻塞启动。
- **`//go:embed all:dist`**：`Windows/frontend/dist/` 虽被 gitignore 但必须存在于工作区，清理构建产物时不要删除。
- **单文件行数**：所有代码文件须遵循不超过 600 行的项目规范（`AGENTS.md`）。
