#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
谛听 (DITING) Windows 端构建与安装包打包引擎
纯 Python 驱动，支持特权服务、Wails 前端与 NSIS 安装包流水线编译及环境预检。
"""

import argparse
import datetime
import os
import shutil
import subprocess
import sys
from typing import Callable, Dict, List, Optional

if hasattr(sys.stdout, "reconfigure"):
    try:
        if sys.stdout.isatty():
            sys.stdout.reconfigure(errors="replace")
        else:
            sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except Exception:
        pass
if hasattr(sys.stderr, "reconfigure"):
    try:
        if sys.stderr.isatty():
            sys.stderr.reconfigure(errors="replace")
        else:
            sys.stderr.reconfigure(encoding="utf-8", errors="replace")
    except Exception:
        pass


ROOT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
if ROOT_DIR not in sys.path:
    sys.path.insert(0, ROOT_DIR)

from maintenance.core.runner import stream_process_output


def get_root_dir() -> str:
    return ROOT_DIR


def get_current_project_version() -> str:
    """自动获取当前项目的核心最新版本号（消除硬编码）"""
    try:
        from maintenance.core.version_manager import get_current_version
        v = get_current_version()
        if v and v != "0.0.0":
            return v
    except Exception:
        pass

    try:
        wails_json = os.path.join(ROOT_DIR, "Desktop", "wails.json")
        if os.path.exists(wails_json):
            import json
            with open(wails_json, "r", encoding="utf-8") as f:
                data = json.load(f)
                v = data.get("info", {}).get("productVersion")
                if v:
                    return v
    except Exception:
        pass
    return "0.0.0"


def stop_running_processes(on_log: Optional[Callable[[str, str], None]] = None):
    """构建前安全终止旧进程，防止文件占用锁导致编译或打包失败"""
    if sys.platform == "win32":
        for exe in ("diting-gui.exe", "diting-service.exe"):
            try:
                res = subprocess.run(
                    ["taskkill", "/F", "/T", "/IM", exe],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                    check=False,
                )
                if res.returncode == 0 and on_log:
                    on_log("info", f"已安全终止正在运行的旧进程: {exe}")
            except Exception:
                pass
        return

    # 类 Unix（Linux/macOS）：以 pkill 结束同名进程，未运行时 pkill 返回非零属正常
    for name in ("diting-gui", "diting-service"):
        if not shutil.which("pkill"):
            break
        try:
            res = subprocess.run(
                ["pkill", "-x", name],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
            )
            if res.returncode == 0 and on_log:
                on_log("info", f"已安全终止正在运行的旧进程: {name}")
        except Exception:
            pass


def find_tool(name: str, extra_paths: Optional[List[str]] = None) -> Optional[str]:
    cmd = shutil.which(name)
    if cmd:
        return cmd
    if extra_paths:
        for p in extra_paths:
            candidate = os.path.join(p, f"{name}.exe" if sys.platform == "win32" else name)
            if os.path.exists(candidate):
                return candidate
    return None


def check_toolchain() -> Dict[str, dict]:
    nsis_extra = ["C:\\Program Files (x86)\\NSIS", "C:\\Program Files\\NSIS"]
    tools = {
        "go": {"found": False, "version": "", "cmd": shutil.which("go")},
        "git": {"found": False, "version": "", "cmd": shutil.which("git")},
        "wails": {"found": False, "version": "", "cmd": shutil.which("wails")},
        "node": {"found": False, "version": "", "cmd": shutil.which("node")},
        "nsis": {"found": False, "version": "", "cmd": find_tool("makensis", nsis_extra)},
    }

    if tools["go"]["cmd"]:
        try:
            out = subprocess.check_output([tools["go"]["cmd"], "version"], text=True).strip()
            tools["go"]["found"] = True
            tools["go"]["version"] = out.replace("go version ", "")
        except Exception:
            pass

    if tools["git"]["cmd"]:
        try:
            out = subprocess.check_output([tools["git"]["cmd"], "--version"], text=True).strip()
            tools["git"]["found"] = True
            tools["git"]["version"] = out
        except Exception:
            pass

    if tools["wails"]["cmd"]:
        try:
            out = subprocess.check_output([tools["wails"]["cmd"], "version"], text=True, stderr=subprocess.STDOUT).splitlines()
            tools["wails"]["found"] = True
            tools["wails"]["version"] = out[0] if out else "已安装"
        except Exception:
            tools["wails"]["found"] = True
            tools["wails"]["version"] = "已安装"

    if tools["node"]["cmd"]:
        try:
            out = subprocess.check_output([tools["node"]["cmd"], "-v"], text=True).strip()
            tools["node"]["found"] = True
            tools["node"]["version"] = out
        except Exception:
            pass

    if tools["nsis"]["cmd"]:
        try:
            out = subprocess.check_output([tools["nsis"]["cmd"], "/VERSION"], text=True).strip()
            tools["nsis"]["found"] = True
            tools["nsis"]["version"] = f"v{out}"
        except Exception:
            tools["nsis"]["found"] = True
            tools["nsis"]["version"] = "已安装"

    return tools


def format_size(bytes_num: int) -> str:
    if bytes_num >= 1024 * 1024 * 1024:
        return f"{bytes_num / (1024**3):.2f} GB"
    elif bytes_num >= 1024 * 1024:
        return f"{bytes_num / (1024**2):.2f} MB"
    elif bytes_num >= 1024:
        return f"{bytes_num / 1024:.2f} KB"
    return f"{bytes_num} B"


def run_cmd_live(cmd: list, cwd: str, on_log: Callable[[str, str], None], is_cancelled: Callable[[], bool], env: Optional[dict] = None) -> bool:
    return stream_process_output(cmd=cmd, cwd=cwd, on_log=on_log, is_cancelled=is_cancelled, env=env) == 0


def build_pipeline(
    target: str = "all",
    version: Optional[str] = None,
    build_type: str = "release",
    clean: bool = False,
    skip_check: bool = False,
    on_log: Optional[Callable[[str, str], None]] = None,
    is_cancelled: Optional[Callable[[], bool]] = None,
    arch: str = "amd64",
) -> dict:
    if not on_log:
        on_log = lambda lvl, msg: print(f"[{lvl.upper()}] {msg}")
    if not is_cancelled:
        is_cancelled = lambda: False
    if not version:
        version = get_current_project_version()

    root = get_root_dir()
    desktop_dir = os.path.join(root, "Desktop")
    bin_dir = os.path.join(desktop_dir, "build", "bin")
    service_bin = os.path.join(bin_dir, "diting-service.exe")
    gui_bin = os.path.join(bin_dir, "diting-gui.exe")

    on_log("info", "=" * 50)
    on_log("info", f"谛听 (DITING) 构建任务启动: Target={target}, Ver={version}, Type={build_type}")
    on_log("info", "=" * 50)

    # 1. 终止可能正在运行的实例，防止文件写锁
    stop_running_processes(on_log)

    # 2. 前置依赖检查
    tools = check_toolchain()
    if target in ("linux", "deb"):
        if not tools["go"]["found"] and sys.platform != "win32":
            on_log("error", "未找到 Go 环境，无法继续编译。")
            return {"success": False, "message": "缺少 Go 环境"}
        if target == "deb" and sys.platform == "win32" and not shutil.which("wsl"):
            on_log("error", "未检测到 WSL 环境。在 Windows 上打包 deb 安装包需要 WSL 支持。")
            return {"success": False, "message": "缺少 WSL 环境"}
    elif not skip_check:
        if not tools["go"]["found"]:
            on_log("error", "未找到 Go 环境，无法继续编译。")
            return {"success": False, "message": "缺少 Go 环境"}
        if target in ("all", "gui", "installer") and not tools["wails"]["found"]:
            on_log("error", "未找到 Wails CLI，请执行: go install github.com/wailsapp/wails/v2/cmd/wails@latest")
            return {"success": False, "message": "缺少 Wails CLI"}
        if target in ("all", "gui", "installer") and not tools["node"]["found"]:
            on_log("error", "未找到 Node.js，前端客户端编译需要 Node.js。")
            return {"success": False, "message": "缺少 Node.js"}
        if target in ("all", "installer") and not tools["nsis"]["found"]:
            on_log("error", "未找到 makensis，无法打包 NSIS 安装包。")
            return {"success": False, "message": "缺少 NSIS"}

    # 3. 清理旧产物
    if clean and os.path.exists(bin_dir):
        on_log("info", f"清理旧构建产物: {bin_dir}")
        shutil.rmtree(bin_dir, ignore_errors=True)
    os.makedirs(bin_dir, exist_ok=True)

    git_commit = "unknown"
    try:
        git_commit = subprocess.check_output(["git", "-C", root, "rev-parse", "--short", "HEAD"], text=True).strip()
    except Exception:
        pass
    build_time = datetime.datetime.now().astimezone().isoformat()

    is_rel = build_type.lower() == "release"
    ldflags = f"-s -w -X 'main.Version={version}' -X 'main.GitCommit={git_commit}' -X 'main.BuildTime={build_time}'" if is_rel else f"-X 'main.Version={version}' -X 'main.GitCommit={git_commit}' -X 'main.BuildTime={build_time}'"

    built_artifacts = []

    # 3.1 Linux 目标：交叉编译单一二进制（Windows 宿主亦可执行）
    if target == "linux":
        if arch not in ("amd64", "arm64", "arm"):
            return {"success": False, "message": f"不支持的 Linux 架构: {arch}"}
        linux_bin = os.path.join(bin_dir, "diting-service")
        on_log("info", f">>> 正在交叉编译 Linux 特权服务 (linux/{arch})...")
        linux_ldflags = (
            f"-s -w -X main.Version={version} -X main.GitCommit={git_commit} -X main.BuildTime={build_time}"
            if is_rel
            else f"-X main.Version={version} -X main.GitCommit={git_commit} -X main.BuildTime={build_time}"
        )
        cmd_linux = ["go", "build", "-trimpath", "-ldflags", linux_ldflags, "-o", linux_bin, "./cmd/service"]
        env_linux = {"GOOS": "linux", "GOARCH": arch, "CGO_ENABLED": "0"}
        ok = run_cmd_live(cmd_linux, cwd=desktop_dir, on_log=on_log, is_cancelled=is_cancelled, env=env_linux)
        if not ok:
            return {"success": False, "message": "Linux 特权服务编译失败"}
        if os.path.exists(linux_bin):
            sz = os.path.getsize(linux_bin)
            on_log("info", f"[OK] Linux 特权服务编译成功: {linux_bin} ({format_size(sz)})")
            built_artifacts.append(linux_bin)
        else:
            return {"success": False, "message": "未生成预期的 Linux 二进制产物"}

        on_log("info", f"=== 构建流水线执行完毕，生成 {len(built_artifacts)} 个产物 ===")
        return {
            "success": True,
            "bin_dir": bin_dir,
            "artifacts": built_artifacts,
            "target": target,
            "version": version,
            "arch": arch,
        }

    # 3.2 Linux 发行包 (deb / tar.gz)
    if target == "deb":
        if arch not in ("amd64", "arm64", "arm"):
            return {"success": False, "message": f"不支持的 Linux 架构: {arch}"}
        on_log("info", f">>> 正在打包 Linux 发行包 (.deb / .tar.gz, linux/{arch})...")
        if sys.platform == "win32":
            if not shutil.which("wsl"):
                on_log("error", "未检测到 WSL 环境。在 Windows 上打包 deb 安装包需要 WSL 支持。")
                return {"success": False, "message": "缺少 WSL 环境"}
            wsl_dir = desktop_dir.replace("\\", "/")
            cmd_pkg = ["wsl.exe", "-e", "bash", "-c", f"cd \"$(wslpath '{wsl_dir}')\" && ./build_linux.sh --package -a {arch} -v {version} -b {build_type}"]
            ok = run_cmd_live(cmd_pkg, cwd=desktop_dir, on_log=on_log, is_cancelled=is_cancelled)
        else:
            cmd_pkg = ["./build_linux.sh", "--package", "-a", arch, "-v", version, "-b", build_type]
            ok = run_cmd_live(cmd_pkg, cwd=desktop_dir, on_log=on_log, is_cancelled=is_cancelled)

        if not ok:
            return {"success": False, "message": "Linux 发行包打包失败"}

        deb_name = f"diting_{version}_linux_{arch}.deb"
        deb_file = os.path.join(bin_dir, deb_name)
        if os.path.exists(deb_file):
            sz = os.path.getsize(deb_file)
            on_log("info", f"[OK] Debian 安装包生成成功: {deb_file} ({format_size(sz)})")
            built_artifacts.append(deb_file)

        tar_name = f"diting_{version}_linux_{arch}.tar.gz"
        tar_file = os.path.join(bin_dir, tar_name)
        if os.path.exists(tar_file):
            sz = os.path.getsize(tar_file)
            on_log("info", f"[OK] Linux 压缩包生成成功: {tar_file} ({format_size(sz)})")
            built_artifacts.append(tar_file)

        on_log("info", f"=== 构建流水线执行完毕，生成 {len(built_artifacts)} 个产物 ===")
        return {
            "success": True,
            "bin_dir": bin_dir,
            "artifacts": built_artifacts,
            "target": target,
            "version": version,
            "arch": arch,
        }

    # 4. 编译特权服务
    need_service = target in ("all", "service") or (target == "installer" and not os.path.exists(service_bin))
    if need_service:
        on_log("info", ">>> 正在编译 Windows 特权服务 (diting-service.exe)...")
        cmd_svc = ["go", "build", "-ldflags", f"-s -w -H windowsgui -X main.Version={version}", "-o", service_bin, "./cmd/service"]
        ok = run_cmd_live(cmd_svc, cwd=desktop_dir, on_log=on_log, is_cancelled=is_cancelled)
        if not ok:
            return {"success": False, "message": "特权服务编译失败"}
        if os.path.exists(service_bin):
            sz = os.path.getsize(service_bin)
            on_log("info", f"[OK] 特权服务编译成功: {service_bin} ({format_size(sz)})")
            built_artifacts.append(service_bin)

    # 5. 编译 Wails GUI
    need_gui = target in ("all", "gui") or (target == "installer" and not os.path.exists(gui_bin))
    if need_gui:
        on_log("info", ">>> 正在使用 Wails 编译 GUI 客户端 (diting-gui.exe)...")
        wails_cmd = [tools["wails"]["cmd"] or "wails", "build", "-platform", "windows/amd64", "-o", "diting-gui.exe", "-ldflags", ldflags]
        if not is_rel:
            wails_cmd.append("-debug")
        ok = run_cmd_live(wails_cmd, cwd=desktop_dir, on_log=on_log, is_cancelled=is_cancelled)
        if not ok:
            return {"success": False, "message": "GUI 客户端编译失败"}
        if os.path.exists(gui_bin):
            sz = os.path.getsize(gui_bin)
            on_log("info", f"[OK] GUI 客户端编译成功: {gui_bin} ({format_size(sz)})")
            built_artifacts.append(gui_bin)

    # 6. 打包 NSIS 安装包
    if target in ("all", "installer"):
        on_log("info", ">>> 正在打包 NSIS 独立安装包...")
        nsis_exe = tools["nsis"]["cmd"] or "makensis"
        nsi_script = os.path.join(desktop_dir, "build", "windows", "installer", "project.nsi")
        installer_name = f"DITING-{build_type.lower()}-v{version}.exe"
        installer_path = os.path.join(bin_dir, installer_name)
        fallback_installer = os.path.join(bin_dir, "diting-gui-amd64-installer.exe")

        cmd_nsis = [
            nsis_exe,
            "/INPUTCHARSET",
            "UTF8",
            f"-DARG_WAILS_AMD64_BINARY={gui_bin}",
            f"-DPRODUCT_VERSION={version}",
            f"-DINFO_PRODUCTVERSION={version}",
            f"-DOUTPUT_FILENAME={installer_path}",
            nsi_script,
        ]
        ok = run_cmd_live(cmd_nsis, cwd=desktop_dir, on_log=on_log, is_cancelled=is_cancelled)
        if not ok:
            return {"success": False, "message": "NSIS 安装包打包失败"}

        # 若生成到了默认名称，规范命名
        if not os.path.exists(installer_path) and os.path.exists(fallback_installer):
            try:
                shutil.move(fallback_installer, installer_path)
            except Exception:
                pass

        if os.path.exists(installer_path):
            sz = os.path.getsize(installer_path)
            on_log("info", f"[OK] 安装包打包成功: {installer_path} ({format_size(sz)})")
            built_artifacts.append(installer_path)
        else:
            on_log("error", f"未找到生成的安装包文件: {installer_path}")
            return {"success": False, "message": "未生成预期的安装包产物"}

    on_log("info", f"=== 全部构建流水线执行完毕，生成 {len(built_artifacts)} 个产物 ===")
    return {
        "success": True,
        "bin_dir": bin_dir,
        "artifacts": built_artifacts,
        "target": target,
        "version": version,
    }


def main():
    parser = argparse.ArgumentParser(description="谛听构建工具")
    parser.add_argument("-t", "--target", choices=["all", "service", "gui", "installer", "linux", "deb"], default="all")
    parser.add_argument("-v", "--version", default=None, help="目标版本号 (默认自动读取)")
    parser.add_argument("-b", "--build-type", choices=["release", "debug"], default="release")
    parser.add_argument("-a", "--arch", choices=["amd64", "arm64", "arm"], default="amd64", help="Linux 目标架构")
    parser.add_argument("--clean", action="store_true")
    parser.add_argument("--skip-check", action="store_true")
    args = parser.parse_args()

    res = build_pipeline(target=args.target, version=args.version, build_type=args.build_type, clean=args.clean, skip_check=args.skip_check, arch=args.arch)
    sys.exit(0 if res["success"] else 1)


if __name__ == "__main__":
    main()
