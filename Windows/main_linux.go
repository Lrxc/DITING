//go:build linux

package main

import (
	"fmt"
	"os"
)

// Version 信息（与 Windows 端保持一致，可在编译期通过 -ldflags 注入）
var (
	Version   = "1.3.5"
	BuildTime = "dev"
	GitCommit = "dev"
)

// Linux 端暂不提供 Wails 原生窗口，统一以 diting-service + 内嵌 Web 控制台
// 的形式交付，因此该入口仅作提示并返回非零以外的状态码。
func main() {
	fmt.Printf("diting (Linux) version %s (built: %s, commit: %s)\n", Version, BuildTime, GitCommit)
	fmt.Println("Linux 端请使用 diting-service 与 Web 控制台：")
	fmt.Println("  启动服务：sudo diting-service -run")
	fmt.Println("  管理地址：http://127.0.0.1:15353")
	os.Exit(0)
}
