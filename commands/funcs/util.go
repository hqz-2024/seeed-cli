package funcs

import (
	"os"
	"path/filepath"
)

// 最大文件大小，超过这个大小后直接跳过文件
var MaxFileSize int64 = 1 * 1024 * 1024

var IgnoreDir = map[string]struct{}{
	".git":    {},
	".svn":    {},
	".hg":     {},
	".vscode": {},
	".idea":   {},
	".cursor": {},
	".pio": {},

	"node_modules":  {},
	"dist":          {},
	"build":         {},
	".next":         {},
	".nuxt":         {},
	".output":       {},
	".cache":        {},
	".parcel-cache": {},
	".turbo":        {},

	"__pycache__":   {},
	".pytest_cache": {},
	".mypy_cache":   {},
	".venv":         {},
	"venv":          {},
	"env":           {},

	"vendor": {},

	"target":  {},
	".gradle": {},
	"out":     {},

	"cmake-build-debug":   {},
	"cmake-build-release": {},

	"DerivedData": {},
	".build":      {},

	".dart_tool": {},

	".bundle": {},

	"bin": {},
	"obj": {},

	"coverage":  {},
	".coverage": {},
	"tmp":       {},
	"temp":      {},
	"logs":      {},
}

var CodeExt = map[string]struct{}{
	// Go
	".go": {},

	// JavaScript / TypeScript
	".js":  {},
	".mjs": {},
	".cjs": {},
	".ts":  {},
	".tsx": {},
	".jsx": {},

	// Python
	".py": {},

	// Java / Kotlin
	".java": {},
	".kt":   {},
	".kts":  {},

	// C / C++
	".c":   {},
	".cpp": {},
	".cc":  {},
	".h":   {},
	".hpp": {},

	// Rust
	".rs": {},

	// Swift
	".swift": {},

	// Dart / Flutter
	".dart": {},

	// PHP
	".php": {},

	// Ruby
	".rb": {},

	// Shell
	".sh":   {},
	".bash": {},
	".zsh":  {},

	// 前端模板
	".vue":    {},
	".svelte": {},
}

func ShouldIgnore(name string) bool {
	_, ok := IgnoreDir[name]
	return ok
}

func IsCodeFile(path string) bool {
	ext := filepath.Ext(path)
	_, ok := CodeExt[ext]
	return ok
}

/**
 * 统计总文件数量
 */
func CountFiles(root string) (int, error) {
	total := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && IsCodeFile(path) && (info.Size() < MaxFileSize) {
			total++
		}

		return nil
	})

	return total, err
}
