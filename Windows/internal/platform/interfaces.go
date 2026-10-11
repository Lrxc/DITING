package platform

import "context"

// CommandExecutor 定义系统命令执行接口，便于测试与多平台扩展。
type CommandExecutor interface {
	RunCommand(ctx context.Context, name string, args ...string) (string, error)
}

// AdapterScanner 定义网卡枚举与扫描接口。
type AdapterScanner interface {
	ScanAll(ctx context.Context) ([]AdapterInfo, error)
	GetActivePhysicalAdapters(ctx context.Context) ([]AdapterInfo, error)
}

// DNSManager 定义系统 DNS 接管与还原管理接口。
//
// 各平台实现需保证：
//   - Takeover* 失败时必须回滚已生效的改动并清理残留状态文件；
//   - Restore* 必须原子还原接管前快照，并在还原后清理状态文件；
//   - ResetResidualLoopbackDNS 作为兜底，扫描并重置任何残留指向本机回环的 DNS。
type DNSManager interface {
	Takeover(ctx context.Context, adapters []AdapterInfo) error
	TakeoverSingle(ctx context.Context, adapter AdapterInfo) error
	Restore(ctx context.Context) error
	RestoreSingle(ctx context.Context, adapterID string) error
	RestoreAdapters(ctx context.Context, adapters []AdapterState) error
	ResetResidualLoopbackDNS(ctx context.Context) error
	IsTakeoverActive() bool
	GetTakenOverAdapters() []AdapterState
	FlushDNSCache(ctx context.Context) error
}

// Restorer 抽象网卡还原能力接口，便于解耦自愈逻辑与执行逻辑。
type Restorer interface {
	RestoreAdapters(ctx context.Context, adapters []AdapterState) error
}

// StateStore 定义接管状态存取接口。
type StateStore interface {
	Save(state *TakeoverState) error
	Load() (*TakeoverState, error)
	Clear() error
	GetFilePath() string
	CheckAndSelfHeal(ctx context.Context, restorer Restorer) (bool, error)
}

// PortChecker 定义端口冲突探测接口。
type PortChecker interface {
	CheckPort53(ctx context.Context) (*PortCheckResult, error)
	CheckPort53ForAddresses(ctx context.Context, udpAddrs, tcpAddrs []string) (*PortCheckResult, error)
	AutofixPort53ForAddresses(ctx context.Context, udpAddrs, tcpAddrs []string) (*PortCheckResult, error)
}

// PrivilegedExecutor 特权命令执行接口。
//
// Windows 通过 UAC (ShellExecuteEx runas) 弹窗提权；Linux 无等价机制，
// 实现直接以当前身份执行，要求调用方已具备 root 权限。
type PrivilegedExecutor interface {
	RunElevated(ctx context.Context, exe string, args ...string) error
	IsElevated() bool
}
