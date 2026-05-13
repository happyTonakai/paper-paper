package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m *Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()
	if mouse.Button != tea.MouseLeft || !m.pointInViewport(mouse.X, mouse.Y) {
		return m, nil
	}

	p := m.viewportPoint(mouse.X, mouse.Y)
	m.selection = viewportSelection{selecting: true, active: true, start: p, end: p}
	return m, nil
}

func (m *Model) handleMouseMotion(msg tea.MouseMotionMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()
	if !m.selection.selecting || mouse.Button != tea.MouseLeft {
		return m, nil
	}

	m.selection.end = m.viewportPoint(mouse.X, mouse.Y)
	return m, nil
}

func (m *Model) handleMouseRelease(msg tea.MouseReleaseMsg) (tea.Model, tea.Cmd) {
	if !m.selection.selecting {
		return m, nil
	}

	mouse := msg.Mouse()
	m.selection.end = m.viewportPoint(mouse.X, mouse.Y)
	m.selection.selecting = false
	m.selection.active = true

	selected := m.selectedViewportText()
	if selected == "" {
		m.clearSelection()
		return m, nil
	}

	m.copyStatus = "已复制选中文本到剪贴板"
	return m, tea.SetClipboard(selected)
}

func (m *Model) clearSelection() {
	m.selection = viewportSelection{}
}

func (m *Model) pointInViewport(x, y int) bool {
	return x >= 0 && x < m.viewport.Width() && y >= 1 && y < 1+m.viewport.Height()
}

func (m *Model) viewportPoint(x, y int) selectionPoint {
	if x < 0 {
		x = 0
	}
	if x >= m.viewport.Width() {
		x = m.viewport.Width() - 1
	}
	vy := y - 1
	if vy < 0 {
		vy = 0
	}
	if vy >= m.viewport.Height() {
		vy = m.viewport.Height() - 1
	}
	return selectionPoint{x: x, y: vy}
}

func (m *Model) visiblePlainViewportLines() []string {
	lines := strings.Split(ansi.Strip(m.viewport.View()), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return lines
}

func normalizedSelection(sel viewportSelection) (start, end selectionPoint) {
	start, end = sel.start, sel.end
	if start.y > end.y || (start.y == end.y && start.x > end.x) {
		start, end = end, start
	}
	return start, end
}

func (m *Model) selectedViewportText() string {
	if !m.selection.active {
		return ""
	}

	lines := m.visiblePlainViewportLines()
	if len(lines) == 0 {
		return ""
	}
	start, end := normalizedSelection(m.selection)
	if start.y >= len(lines) {
		return ""
	}
	if end.y >= len(lines) {
		end.y = len(lines) - 1
	}

	selected := make([]string, 0, end.y-start.y+1)
	for y := start.y; y <= end.y; y++ {
		line := lines[y]
		from, to := 0, ansi.StringWidth(line)
		if y == start.y {
			from = start.x
		}
		if y == end.y {
			to = end.x + 1
		}
		if to < from {
			to = from
		}
		part := ansi.Cut(line, from, to)
		selected = append(selected, strings.TrimRight(part, " "))
	}

	return strings.Trim(strings.Join(selected, "\n"), "\n")
}

func (m *Model) applySelectionToViewportLines(lines []string) []string {
	if !m.selection.active || len(lines) == 0 {
		return lines
	}

	plain := make([]string, len(lines))
	for i, line := range lines {
		plain[i] = ansi.Strip(line)
	}

	start, end := normalizedSelection(m.selection)
	if start.y >= len(plain) {
		return lines
	}
	if end.y >= len(plain) {
		end.y = len(plain) - 1
	}

	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	for y := start.y; y <= end.y; y++ {
		line := plain[y]
		from, to := 0, ansi.StringWidth(line)
		if y == start.y {
			from = start.x
		}
		if y == end.y {
			to = end.x + 1
		}
		if to < from {
			to = from
		}

		before := ansi.Cut(line, 0, from)
		middle := ansi.Cut(line, from, to)
		after := ansi.Cut(line, to, ansi.StringWidth(line))
		plain[y] = before + selectedStyle.Render(middle) + after
	}

	return plain
}
