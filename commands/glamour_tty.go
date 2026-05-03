package commands

import (
	"regexp"

	"charm.land/glamour/v2"
)

// Chroma 无独立 mermaid 词法分析器，使用 mermaid 标签时易被误判为其它语言或导致版式错乱。
// 终端也无法像浏览器那样渲染 Mermaid，故在「仅用于 TTY 展示」的副本里把围栏语言改为 text，
// 并插入一行说明；写入磁盘的 .md 仍为原始内容（见 repStreamEndMsg 处传入的 msg.full）。
var mermaidFenceOpen = regexp.MustCompile(`(?m)^\x60{3}\s*(?i:mermaid)\s*$`)

func sanitizeMermaidFencesForTTY(md string) string {
	note := "# （Mermaid 源码：用 VS Code / Cursor / GitHub 打开 .md 预览可渲染为图；终端内为纯文本）\n"
	return mermaidFenceOpen.ReplaceAllString(md, "```text\n"+note)
}

// glamourRenderForViewport 用 glamour 将 Markdown 转为 ANSI：按 viewport 宽度换行，并预处理 Mermaid 围栏。
func glamourRenderForViewport(md string, viewportWidth int) (string, error) {
	wrap := viewportWidth - 8
	if wrap < 52 {
		wrap = 88
	}
	md = sanitizeMermaidFencesForTTY(md)
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(wrap),
	)
	if err != nil {
		return glamour.Render(md, "dark")
	}
	return r.Render(md)
}
