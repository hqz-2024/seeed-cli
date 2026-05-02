package commands

import (
    "fmt"  
    "context"
	"os"
	"path/filepath"

	// "time"

    "github.com/urfave/cli/v3" 
	// tea "github.com/charmbracelet/bubbletea"
	// "seeed-cli/commands/funcs"
)

// 最大文件大小，超过这个大小后直接跳过文件
var maxFileSize int64 = 1*1024*1024;

var ignoreDir = map[string]struct{}{
	".git": {},
	".svn": {},
	".hg": {},
	".vscode": {},
	".idea": {},
	".cursor": {},

	"node_modules": {},
	"dist": {},
	"build": {},
	".next": {},
	".nuxt": {},
	".output": {},
	".cache": {},
	".parcel-cache": {},
	".turbo": {},

	"__pycache__": {},
	".pytest_cache": {},
	".mypy_cache": {},
	".venv": {},
	"venv": {},
	"env": {},

	"vendor": {},

	"target": {},
	".gradle": {},
	"out": {},

	"cmake-build-debug": {},
	"cmake-build-release": {},

	"DerivedData": {},
	".build": {},

	".dart_tool": {},

	".bundle": {},

	"bin": {},
	"obj": {},

	"coverage": {},
	".coverage": {},
	"tmp": {},
	"temp": {},
	"logs": {},
}


var codeExt = map[string]struct{}{
	// Go
	".go": {},

	// JavaScript / TypeScript
	".js": {},
	".mjs": {},
	".cjs": {},
	".ts": {},
	".tsx": {},
	".jsx": {},

	// Python
	".py": {},

	// Java / Kotlin
	".java": {},
	".kt": {},
	".kts": {},

	// C / C++
	".c": {},
	".cpp": {},
	".cc": {},
	".h": {},
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
	".sh": {},
	".bash": {},
	".zsh": {},

	// 前端模板
	".vue": {},
	".svelte": {}, 
}

func shouldIgnore(name string) bool{
	_, ok := ignoreDir[name]
	return ok
}

func isCodeFile(path string) bool {
	ext := filepath.Ext(path)
	_, ok := codeExt[ext]
	return ok
}


/**
 * 统计总文件数量
 */
func countFiles(root string) (int, error) {
	total := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isCodeFile(path) && (info.Size() < maxFileSize) {  
			total++
		}

		return nil
	})

	return total, err
}

/**
 * 检测项目中每个代码文件的安全性
 * 原理是递归读取每个文件，然后将内容提交给 ai
*/
func HandleFrame(ctx context.Context, cmd *cli.Command) error {

	// 当前命令执行目录 
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	
	fmt.Println("扫描目录：", pwd)

	total, err := countFiles(pwd)
	if err != nil {
		return err
	}

	current := 0

	filepath.Walk(pwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && shouldIgnore(info.Name()) {
			return filepath.SkipDir
		}
 
		if !info.IsDir() && isCodeFile(path) && (info.Size() < maxFileSize) {  
			current++ 
			progress := float64(current) / float64(total) * 100


			// time.Sleep(1 * time.Second) // ⏱️ 等 2 秒

			fmt.Printf("进度: %.2f%% (%d/%d)  %s \n", progress, current, total, path)
			
			// 读取文件内容
			content, err := os.ReadFile(path)
			if err != nil {
				fmt.Println("读取失败:", err)
				return nil
			}
			strContent := string(content)
			// println(strContent)

			llmRes := funcs.FetchLLM(`
			帮我检查下面代码中是否存在安全问题：

			
			`, "qwen3.6-flash")

			println("")
		}

		// fmt.Printf("\r进度: ", path)

		return nil
	})

	// llmRes := funcs.FetchLLM("你好", "qwen3.6-flash")
	// fmt.Println(llmRes)
	
	println("")
	return nil
}

