package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/urfave/cli/v3"
)

// HandleInstallShort：在与当前可执行文件同目录创建 s（Windows 为 s.exe）软链接指向本程序。
func HandleInstallShort(ctx context.Context, cmd *cli.Command) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}

	dir := filepath.Dir(exe)
	linkName := "s"
	if runtime.GOOS == "windows" {
		linkName = "s.exe"
	}
	linkPath := filepath.Join(dir, linkName)

	if filepath.Clean(linkPath) == filepath.Clean(exe) {
		return fmt.Errorf("可执行文件与链接同名，无法创建")
	}

	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			existing, err := os.Readlink(linkPath)
			if err != nil {
				return err
			}
			res := existing
			if !filepath.IsAbs(existing) {
				res = filepath.Join(dir, existing)
			}
			res, _ = filepath.Abs(res)
			res, _ = filepath.EvalSymlinks(res)
			if filepath.Clean(res) == filepath.Clean(exe) {
				fmt.Println("s 已存在且指向本程序")
				return nil
			}
			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("无法替换已有软链接: %w", err)
			}
		} else {
			return fmt.Errorf("已存在同名普通文件: %s，请手动删除后重试", linkPath)
		}
	}

	if err := os.Symlink(exe, linkPath); err != nil {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("创建软链接失败: %w（可开启「设置-开发者选项-开发人员模式」或以管理员运行）", err)
		}
		return fmt.Errorf("创建软链接失败: %w", err)
	}

	fmt.Printf("已创建软链接: %s -> %s\n", linkPath, exe)
	return nil
}
