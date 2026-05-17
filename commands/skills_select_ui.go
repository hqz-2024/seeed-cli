// skills_select_ui：skills-sync 的两个交互 TUI —— 源 skill 多选与目标工具单选。
package commands

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"seeed-cli/commands/funcs"
)

var (
	selHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#00cc44")).Bold(true)
	selHint   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	selChosen = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	selDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// ErrSelectionCancelled：用户按 q / esc / ctrl+c 主动取消。
var ErrSelectionCancelled = errors.New("selection cancelled")

// skillsMultiSelectModel：多选 skill 列表（↑/↓ 移动、Space 选/取消、a 全选、Enter 确认、q 取消）。
type skillsMultiSelectModel struct {
	items    []funcs.Skill
	cursor   int
	checked  map[int]bool
	done     bool
	cancel   bool
	width    int
	height   int
}

func (m *skillsMultiSelectModel) Init() tea.Cmd { return nil }

func (m *skillsMultiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.cancel = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ", "space":
			m.checked[m.cursor] = !m.checked[m.cursor]
		case "a":
			allOn := true
			for i := range m.items {
				if !m.checked[i] {
					allOn = false
					break
				}
			}
			for i := range m.items {
				m.checked[i] = !allOn
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *skillsMultiSelectModel) View() tea.View {
	var v tea.View
	v.AltScreen = true
	var sb strings.Builder
	sb.WriteString(selHeader.Render("Select source skills") + "\n")
	sb.WriteString(selHint.Render("↑/↓ move · space toggle · a all · enter confirm · q cancel") + "\n\n")
	for i, sk := range m.items {
		mark := "[ ]"
		if m.checked[i] {
			mark = selChosen.Render("[x]")
		}
		line := fmt.Sprintf(" %s  %s/%s  %s", mark, sk.Source, sk.Name, selDim.Render(truncate(sk.Description, 60)))
		if i == m.cursor {
			line = selCursor.Render("›") + line[1:]
		} else {
			line = " " + line[1:]
		}
		sb.WriteString(line + "\n")
	}
	v.SetContent(sb.String())
	return v
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

// SelectSourceSkillsInteractive：多选 TUI，返回用户勾选的源 skill。
func SelectSourceSkillsInteractive(skills []funcs.Skill) ([]funcs.Skill, error) {
	if len(skills) == 0 {
		return nil, errors.New("no skills available")
	}
	m := &skillsMultiSelectModel{items: skills, checked: map[int]bool{}}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return nil, err
	}
	if m.cancel {
		return nil, ErrSelectionCancelled
	}
	var out []funcs.Skill
	for i, sk := range m.items {
		if m.checked[i] {
			out = append(out, sk)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("nothing selected")
	}
	return out, nil
}


// skillsTargetSelectModel：单选目标工具（四个固定项）。
type skillsTargetSelectModel struct {
	options []funcs.SkillSource
	cursor  int
	done    bool
	cancel  bool
}

func (m *skillsTargetSelectModel) Init() tea.Cmd { return nil }

func (m *skillsTargetSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "ctrl+c", "q", "esc":
			m.cancel = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter", " ", "space":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *skillsTargetSelectModel) View() tea.View {
	var v tea.View
	v.AltScreen = true
	var sb strings.Builder
	sb.WriteString(selHeader.Render("Select target tool") + "\n")
	sb.WriteString(selHint.Render("↑/↓ move · enter confirm · q cancel") + "\n\n")
	for i, opt := range m.options {
		prefix := "  "
		label := string(opt)
		if i == m.cursor {
			prefix = selCursor.Render("› ")
			label = selChosen.Render(label)
		}
		sb.WriteString(prefix + label + "\n")
	}
	v.SetContent(sb.String())
	return v
}

// SelectTargetInteractive：四个工具固定单选。
func SelectTargetInteractive() (funcs.SkillSource, error) {
	m := &skillsTargetSelectModel{
		options: []funcs.SkillSource{funcs.SourceCursor, funcs.SourceClaude, funcs.SourceWindsurf, funcs.SourceAugment},
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return "", err
	}
	if m.cancel {
		return "", ErrSelectionCancelled
	}
	return m.options[m.cursor], nil
}
