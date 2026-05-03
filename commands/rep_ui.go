// rep_ui：bubbletea 全屏报告 UI，供 safe-scan、frame 等子命令复用（流式 LLM + glamour 终端渲染）。
package commands

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"seeed-cli/commands/funcs"
)

// repAppendLogMsg：后台任务写入一行日志（已带 ANSI 时原样追加）。
type repAppendLogMsg string

// repFinishedMsg：work 结束，携带落盘路径（可为空）。
type repFinishedMsg struct {
	savePath string
}

// repLoadingSetMsg：控制顶栏 NEURAL 与底部 spinner（LLM 请求期间为 true）。
type repLoadingSetMsg bool

type repStreamStartMsg struct{}

type repStreamDeltaMsg string

type repStreamEndMsg struct {
	full string
	err  error
}

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00cc44")).BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()

	logDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	logHi   = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	logText = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	logWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
)

func bootOKLine(msg string) string {
	return logDim.Render("[ ") + logHi.Render("OK") + logDim.Render(" ] ") + logText.Render(msg)
}

func bootWaitLine(msg string) string {
	return logDim.Render("[ ") + logWarn.Render("..") + logDim.Render(" ] ") + logText.Render(msg)
}

// repModel：单例 tea Model；work 在 Init 里起 goroutine 执行，禁止在 work 里直接操作 viewport。

type repModel struct {
	prog        *tea.Program
	headerTitle string
	pwd         string
	total       int
	content     string
	ready       bool
	finished    bool
	savePath    string
	viewport    viewport.Model
	loading     bool
	spin        spinner.Model
	scanCtx     context.Context
	llmStream   string
	work        func(*repModel)
	lastSave    string // work 结束前写入，供 repFinishedMsg 带给 UI
}

func (m *repModel) SendLog(line string) {
	if m.prog != nil {
		m.prog.Send(repAppendLogMsg(line))
	}
}

// RunLLMStream：打开流式 UI（loading + delta），返回聚合全文与 API 错误。

func (m *repModel) RunLLMStream(prompt, model string) (string, error) {
	m.prog.Send(repLoadingSetMsg(true))
	defer m.prog.Send(repLoadingSetMsg(false))
	m.prog.Send(repStreamStartMsg{})
	full, err := funcs.FetchLLMStream(m.scanCtx, prompt, model, func(d string) {
		if d != "" {
			m.prog.Send(repStreamDeltaMsg(d))
		}
	})
	m.prog.Send(repStreamEndMsg{full: full, err: err})
	return full, err
}

func (m *repModel) wantSpinner() bool {
	return !m.ready || m.loading
}

func (m *repModel) Init() tea.Cmd {
	go func() {
		m.work(m)
		m.prog.Send(repFinishedMsg{savePath: m.lastSave})
	}()
	return func() tea.Msg {
		return m.spin.Tick()
	}
}

func (m *repModel) headerView() string {
	title := titleStyle.Render(m.headerTitle)
	if m.loading {
		title = lipgloss.JoinHorizontal(lipgloss.Center, title, logWarn.Render(" ▸ NEURAL "))
	}
	line := strings.Repeat("─", max(0, m.viewport.Width()-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m *repModel) footerView() string {
	st := "RUN"
	if m.finished {
		st = "OK"
	}
	prefix := ""
	if m.wantSpinner() {
		prefix = m.spin.View() + " "
	}
	info := infoStyle.Render(fmt.Sprintf("%s%s │ %3.f%%", prefix, st, m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width()-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m *repModel) syncViewport() {
	if !m.ready {
		return
	}
	m.viewport.SetContent(m.content + m.llmStream)
	m.viewport.GotoBottom()
}

func (m *repModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if !m.wantSpinner() {
			break
		}
		var sc tea.Cmd
		m.spin, sc = m.spin.Update(msg)
		cmds = append(cmds, sc)

	case repLoadingSetMsg:
		was := m.loading
		m.loading = bool(msg)
		if m.loading && !was {
			cmds = append(cmds, func() tea.Msg {
				return m.spin.Tick()
			})
		}

	case repStreamStartMsg:
		m.llmStream = ""
		m.syncViewport()

	case repStreamDeltaMsg:
		m.llmStream += string(msg)
		m.syncViewport()

	case repStreamEndMsg:
		m.llmStream = ""
		if msg.err != nil {
			m.content += logWarn.Render(msg.err.Error()) + "\n"
		} else if msg.full != "" {
			vw := m.viewport.Width()
			if vw < 40 {
				vw = 100
			}
			out, gerr := glamourRenderForViewport(msg.full, vw)
			if gerr != nil {
				m.content += logWarn.Render(gerr.Error()) + "\n"
			} else {
				m.content += out + "\n"
			}
		}
		m.syncViewport()

	case tea.KeyPressMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

	case repAppendLogMsg:
		m.content += string(msg) + "\n"
		m.syncViewport()

	case repFinishedMsg:
		m.finished = true
		m.savePath = msg.savePath
		m.content += bootOKLine("done — q to exit") + "\n"
		m.syncViewport()

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(msg.Height-verticalMarginHeight))
			m.viewport.YPosition = headerHeight
			m.viewport.LeftGutterFunc = func(info viewport.GutterContext) string {
				if info.Soft {
					return "     │ "
				}
				if info.Index >= info.TotalLines {
					return "   ~ │ "
				}
				return fmt.Sprintf("%4d │ ", info.Index+1)
			}
			m.viewport.MouseWheelEnabled = true
			m.viewport.SetContent(m.content + m.llmStream)
			m.viewport.GotoBottom()
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - verticalMarginHeight)
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *repModel) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if !m.ready {
		v.SetContent(fmt.Sprintf("\n  %s  Initializing…", m.spin.View()))
	} else {
		v.SetContent(fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView()))
	}
	return v
}

// runRepUI：启动 tea Program；pwd/total 供 work 使用（如安全扫描进度分母）。

func runRepUI(ctx context.Context, headerTitle string, pwd string, total int, work func(*repModel)) error {
	m := &repModel{
		headerTitle: headerTitle,
		pwd:         pwd,
		total:       total,
		scanCtx:     ctx,
		work:        work,
		spin: spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff9f")).Bold(true)),
		),
	}
	p := tea.NewProgram(m)
	m.prog = p
	_, err := p.Run()
	return err
}
