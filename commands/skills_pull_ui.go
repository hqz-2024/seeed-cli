// skills_pull_ui：skills-pull 的远程 IndexEntry 多选 TUI。
package commands

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"seeed-cli/commands/funcs"
)

// remoteMultiSelectModel：多选远程 skill（↑/↓ 移动、Space 选/取消、a 全选、Enter 确认、q 取消）。
type remoteMultiSelectModel struct {
	items   []funcs.IndexEntry
	cursor  int
	checked map[int]bool
	done    bool
	cancel  bool
}

func (m *remoteMultiSelectModel) Init() tea.Cmd { return nil }

func (m *remoteMultiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *remoteMultiSelectModel) View() tea.View {
	var v tea.View
	v.AltScreen = true
	var sb strings.Builder
	sb.WriteString(selHeader.Render("Select skills to pull from GitHub") + "\n")
	sb.WriteString(selHint.Render("↑/↓ move · space toggle · a all · enter confirm · q cancel") + "\n\n")
	for i, e := range m.items {
		mark := "[ ]"
		if m.checked[i] {
			mark = selChosen.Render("[x]")
		}
		stars := fmt.Sprintf("★%d", e.Stars)
		line := fmt.Sprintf(" %s  %-40s  %-8s  %s", mark, e.ID, stars, selDim.Render(truncate(e.Description, 50)))
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

// SelectRemoteEntriesInteractive：多选 TUI，返回用户勾选的远程索引条目。
func SelectRemoteEntriesInteractive(entries []funcs.IndexEntry) ([]funcs.IndexEntry, error) {
	if len(entries) == 0 {
		return nil, errors.New("no entries available")
	}
	m := &remoteMultiSelectModel{items: entries, checked: map[int]bool{}}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return nil, err
	}
	if m.cancel {
		return nil, ErrSelectionCancelled
	}
	var out []funcs.IndexEntry
	for i, e := range m.items {
		if m.checked[i] {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("nothing selected")
	}
	return out, nil
}
