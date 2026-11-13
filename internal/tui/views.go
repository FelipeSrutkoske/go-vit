package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
)

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	if m.width == 0 || m.height == 0 {
		v := tea.NewView("Carregando go-vit...")
		v.AltScreen = true
		return v
	}

	var content string
	if m.modal != modalNone {
		content = m.renderModal()
	} else if m.state.Kind == git.KindNotARepo && !m.loading {
		content = m.renderNotARepo()
	} else {
		content = m.renderMain()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m *Model) resizeViewport() {
	if m.width == 0 || m.height == 0 {
		return
	}
	_, _, rightW, rightH := m.paneDimensions()
	vpW := max(16, rightW-4)
	vpH := max(3, rightH-4)
	if m.activeTab == tabChanges && len(m.files) > 0 && m.files[m.fileIdx].IsConflicted {
		vpH = max(2, vpH-2)
	}
	m.viewport.SetWidth(vpW)
	m.viewport.SetHeight(vpH)
}

func (m *Model) paneDimensions() (leftW, leftH, rightW, rightH int) {
	const headerH = 3
	const footerH = 2
	bodyH := max(8, m.height-headerH-footerH)

	if m.width < 90 {
		leftW = max(20, m.width)
		rightW = leftW
		leftH = max(5, bodyH*45/100)
		rightH = max(5, bodyH-leftH)
		return
	}

	leftW = max(32, m.width*38/100)
	rightW = max(30, m.width-leftW)
	leftH = bodyH
	rightH = bodyH
	return
}

func (m *Model) syncRightPane() {
	m.resizeViewport()
	if m.activeTab == tabBranches {
		m.viewport.SetContent(m.renderCommitLog())
		m.viewport.GotoTop()
		return
	}

	if len(m.files) == 0 {
		m.viewport.SetContent(m.styles.emptyState.Render("Nenhuma alteracao pendente. O diretorio de trabalho esta limpo."))
		m.viewport.GotoTop()
		return
	}

	f := m.files[m.fileIdx]
	if f.IsConflicted {
		m.viewport.SetContent(m.renderConflictDiff(m.diffText))
	} else {
		m.viewport.SetContent(m.renderColorDiff(m.diffText))
	}
	m.viewport.GotoTop()
}

func (m Model) renderMain() string {
	header := m.renderHeader()
	leftW, leftH, rightW, rightH := m.paneDimensions()
	left := m.renderLeftPane(leftW, leftH)
	right := m.renderRightPane(rightW, rightH)

	var body string
	if m.width < 90 {
		body = lipgloss.JoinVertical(lipgloss.Left, left, right)
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) renderHeader() string {
	repo := m.state.RepoName
	if repo == "" {
		repo = m.client.Dir()
	}

	var branchBadge string
	switch m.state.Kind {
	case git.KindDetached:
		label := "HEAD destacado"
		if m.state.HeadHash != "" {
			label += " @" + m.state.HeadHash
		}
		branchBadge = m.styles.detachedBadge.Render(label)
	case git.KindUnborn:
		label := m.state.Branch
		if label == "" {
			label = "main"
		}
		branchBadge = m.styles.unbornBadge.Render(label + " (sem commits)")
	default:
		label := m.state.Branch
		if label == "" {
			label = "..."
		}
		branchBadge = m.styles.branchBadge.Render(label)
	}

	var syncParts []string
	if m.state.Ahead > 0 {
		syncParts = append(syncParts, fmt.Sprintf("^%d", m.state.Ahead))
	}
	if m.state.Behind > 0 {
		syncParts = append(syncParts, fmt.Sprintf("v%d", m.state.Behind))
	}
	syncStr := ""
	if len(syncParts) > 0 {
		syncStr = " " + m.styles.aheadBehind.Render(strings.Join(syncParts, " "))
	}

	var statusBadge string
	switch {
	case m.state.ConflictCount > 0:
		label := fmt.Sprintf("%d conflitos", m.state.ConflictCount)
		if m.state.ConflictCount == 1 {
			label = "1 conflito"
		}
		statusBadge = m.styles.statusConflict.Render(label)
	case m.state.ChangedCount > 0:
		label := fmt.Sprintf("%d alteracoes", m.state.ChangedCount)
		if m.state.ChangedCount == 1 {
			label = "1 alteracao"
		}
		statusBadge = m.styles.statusDirty.Render(label)
	default:
		statusBadge = m.styles.statusClean.Render("Limpo")
	}

	left := m.styles.repoName.Render("go-vit :: "+repo) + "  " + branchBadge + syncStr + "  " + statusBadge
	if m.loading {
		left += "  " + m.spinner.View()
	}

	innerW := max(20, m.width-4)
	return m.styles.headerBox.Width(innerW).Render(left)
}

func (m Model) renderLeftPane(w, h int) string {
	innerW := max(16, w-4)
	innerH := max(3, h-2)

	tab1Label := "[1] Branches"
	tab2Label := "[2] Alteracoes"
	if m.state.ConflictCount > 0 {
		tab2Label = fmt.Sprintf("[2] Alteracoes (%d!)", m.state.ConflictCount)
	} else if len(m.files) > 0 {
		tab2Label = fmt.Sprintf("[2] Alteracoes (%d)", len(m.files))
	}

	var t1, t2 string
	if m.activeTab == tabBranches {
		t1 = m.styles.tabActive.Render(tab1Label)
		t2 = m.styles.tabInactive.Render(tab2Label)
	} else {
		t1 = m.styles.tabInactive.Render(tab1Label)
		t2 = m.styles.tabActive.Render(tab2Label)
	}

	tabsRow := t1 + " " + t2
	listH := max(1, innerH-2)

	var body string
	if m.activeTab == tabBranches {
		body = m.renderBranchList(innerW, listH)
	} else {
		body = m.renderFileList(innerW, listH)
	}

	content := tabsRow + "\n" + m.styles.itemMuted.Render(strings.Repeat("-", innerW)) + "\n" + body
	return m.styles.paneActive.Width(w - 2).Height(h - 2).Render(content)
}

func (m Model) renderBranchList(w, h int) string {
	if len(m.branches) == 0 {
		return m.styles.emptyState.Render("Nenhuma branch encontrada.\nPressione 'n' para criar uma.")
	}

	hintH := 1
	visible := max(1, h-hintH)
	start, end := visibleWindow(m.branchIdx, len(m.branches), visible)

	var lines []string
	for i := start; i < end; i++ {
		b := m.branches[i]
		prefix := "  "
		if b.IsCurrent {
			prefix = "* "
		}
		if i == m.branchIdx {
			prefix = "> "
			if b.IsCurrent {
				prefix = ">*"
			}
		}

		label := b.Name
		var suffix string
		if b.IsRemote {
			suffix += " [remota]"
		}
		if b.Ahead > 0 || b.Behind > 0 {
			suffix += fmt.Sprintf(" ^%d v%d", b.Ahead, b.Behind)
		}

		row := truncateText(prefix+label+suffix, w)
		if i == m.branchIdx {
			lines = append(lines, m.styles.itemSelected.Width(w).Render(row))
		} else if b.IsCurrent {
			lines = append(lines, m.styles.badgeCurrent.Render(row))
		} else if b.IsRemote {
			lines = append(lines, m.styles.badgeRemote.Render(row))
		} else {
			lines = append(lines, m.styles.itemNormal.Render(row))
		}
	}

	for len(lines) < visible {
		lines = append(lines, "")
	}

	remoteState := "off"
	if m.includeRemote {
		remoteState = "on"
	}
	hint := fmt.Sprintf("Enter:trocar  n:nova  d:excluir  m:merge  r:remotas(%s)", remoteState)
	lines = append(lines, m.styles.itemMuted.Render(truncateText(hint, w)))
	return strings.Join(lines, "\n")
}

func (m Model) renderFileList(w, h int) string {
	if len(m.files) == 0 {
		return m.styles.emptyState.Render("Nenhuma alteracao no repositorio.\nPressione 'z' para restaurar stash (pop).")
	}

	hintH := 1
	visible := max(1, h-hintH)
	start, end := visibleWindow(m.fileIdx, len(m.files), visible)

	var lines []string
	for i := start; i < end; i++ {
		f := m.files[i]
		badge, badgeStyle := m.fileBadge(f)

		cursor := "  "
		if i == m.fileIdx {
			cursor = "> "
		}

		name := f.Path
		if f.OldPath != "" {
			name = f.OldPath + " -> " + f.Path
		}

		maxNameW := max(8, w-len(cursor)-len(badge)-1)
		name = truncateText(name, maxNameW)

		if i == m.fileIdx {
			row := fmt.Sprintf("%s%s %s", cursor, badge, name)
			lines = append(lines, m.styles.itemSelected.Width(w).Render(truncateText(row, w)))
		} else {
			row := cursor + badgeStyle.Render(badge) + " " + m.styles.itemNormal.Render(name)
			lines = append(lines, row)
		}
	}

	for len(lines) < visible {
		lines = append(lines, "")
	}

	hint := "Espaco/s:stage  c:commit  z:stash/pop  x:descartar"
	lines = append(lines, m.styles.itemMuted.Render(truncateText(hint, w)))
	return strings.Join(lines, "\n")
}

func (m Model) fileBadge(f git.FileStatus) (string, lipgloss.Style) {
	switch {
	case f.IsConflicted:
		return "[!]", m.styles.badgeConflict
	case f.IsUntracked:
		return "[?]", m.styles.badgeUntracked
	case f.IsStaged && f.IsUnstaged:
		return "[+M]", m.styles.badgeUnstaged
	case f.IsStaged:
		return "[+]", m.styles.badgeStaged
	default:
		return "[M]", m.styles.badgeUnstaged
	}
}

func (m Model) renderRightPane(w, h int) string {
	innerW := max(16, w-4)

	var title string
	var conflictBar string

	if m.activeTab == tabBranches {
		if len(m.branches) > 0 {
			title = fmt.Sprintf("Commits em %s", m.branches[m.branchIdx].Name)
		} else {
			title = "Commits"
		}
	} else if len(m.files) > 0 {
		f := m.files[m.fileIdx]
		if f.IsConflicted {
			title = "Conflito: " + f.Path
			conflictBar = m.styles.conflictBar.Width(innerW).Render(
				"[1] Manter nossos (ours)   [2] Aceitar deles (theirs)   [e] Abrir no $EDITOR",
			)
		} else {
			title = "Diff: " + f.Path
		}
	} else {
		title = "Visualizacao"
	}

	header := m.styles.repoName.Render(truncateText(title, innerW))
	parts := []string{header}
	if conflictBar != "" {
		parts = append(parts, conflictBar)
	}
	parts = append(parts, m.viewport.View())

	content := strings.Join(parts, "\n")
	return m.styles.paneInactive.Width(w - 2).Height(h - 2).Render(content)
}

func (m Model) renderCommitLog() string {
	if len(m.commits) == 0 {
		return m.styles.emptyState.Render("Nenhum commit nesta branch ainda.")
	}

	var lines []string
	for _, c := range m.commits {
		hash := m.styles.commitHash.Render(c.Hash)
		subj := m.styles.commitSubject.Render(c.Subject)
		meta := m.styles.commitMeta.Render(fmt.Sprintf("(%s, %s)", c.Author, c.RelativeDate))
		lines = append(lines, fmt.Sprintf("%s %s %s", hash, subj, meta))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderColorDiff(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return m.styles.emptyState.Render("Sem diferencas para exibir.")
	}
	rawLines := strings.Split(raw, "\n")
	out := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		switch {
		case strings.HasPrefix(line, "diff --git"),
			strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "--- "),
			strings.HasPrefix(line, "+++ "):
			out = append(out, m.styles.diffMeta.Render(line))
		case strings.HasPrefix(line, "@@"):
			out = append(out, m.styles.diffHunk.Render(line))
		case strings.HasPrefix(line, "+"):
			out = append(out, m.styles.diffAdd.Render(line))
		case strings.HasPrefix(line, "-"):
			out = append(out, m.styles.diffDel.Render(line))
		default:
			out = append(out, m.styles.itemNormal.Render(line))
		}
	}
	return strings.Join(out, "\n")
}

func (m Model) renderConflictDiff(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return m.styles.emptyState.Render("Arquivo em conflito vazio ou binario.")
	}

	const (
		secNormal = iota
		secOurs
		secTheirs
	)

	section := secNormal
	rawLines := strings.Split(raw, "\n")
	out := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		switch {
		case strings.HasPrefix(line, "<<<<<<<"):
			section = secOurs
			out = append(out, m.styles.conflictOurs.Render(line+"  [NOSSOS / OURS]"))
		case strings.HasPrefix(line, "======="):
			section = secTheirs
			out = append(out, m.styles.conflictSep.Render(line))
		case strings.HasPrefix(line, ">>>>>>>"):
			section = secNormal
			out = append(out, m.styles.conflictTheirs.Render(line+"  [DELES / THEIRS]"))
		case section == secOurs:
			out = append(out, m.styles.diffAdd.Render("| "+line))
		case section == secTheirs:
			out = append(out, m.styles.conflictTheirs.Render("| "+line))
		default:
			out = append(out, m.styles.itemNormal.Render(line))
		}
	}
	return strings.Join(out, "\n")
}

func (m Model) renderFooter() string {
	var top string
	if m.errMsg != "" {
		top = m.styles.bannerError.Render("Erro: " + m.errMsg)
	} else if m.status != "" {
		top = m.styles.bannerInfo.Render(m.status)
	} else {
		top = m.styles.footer.Render("f:fetch  p:pull  R:atualizar  Tab:aba  J/K:rolar dir.  ?:ajuda  q:sair")
	}
	return truncateText(top, max(20, m.width))
}

func (m Model) renderModal() string {
	var body string
	switch m.modal {
	case modalHelp:
		body = m.styles.modalTitle.Render("Atalhos do go-vit") + "\n\n" +
			m.help.FullHelpView(m.keys.FullHelp()) + "\n\n" +
			m.styles.itemMuted.Render("Pressione Esc, ? ou Enter para fechar")

	case modalCreateBranch:
		body = m.styles.modalTitle.Render("Criar nova branch") + "\n\n" +
			"Nome da branch:\n" +
			m.input.View() + "\n\n" +
			m.styles.itemMuted.Render("Enter: criar e trocar   Esc: cancelar")

	case modalCommit:
		body = m.styles.modalTitle.Render(fmt.Sprintf("Fazer commit (%d arquivo(s) preparado(s))", m.state.StagedCount)) + "\n\n" +
			"Mensagem:\n" +
			m.input.View() + "\n\n" +
			m.styles.itemMuted.Render("Enter: commitar   Esc: cancelar")

	case modalDeleteBranch:
		body = m.styles.modalTitle.Render("Excluir branch") + "\n\n" +
			fmt.Sprintf("Deseja excluir a branch local '%s'?", m.modalTarget) + "\n\n" +
			m.styles.itemMuted.Render("Enter/s: excluir   Esc/n: cancelar")

	case modalForceDelete:
		body = m.styles.modalTitle.Render("Branch nao mesclada") + "\n\n" +
			m.styles.modalWarn.Render(fmt.Sprintf("A branch '%s' possui commits nao mesclados.", m.modalTarget)) + "\n" +
			"Deseja forcar a exclusao (-D)?\n\n" +
			m.styles.itemMuted.Render("Enter/f/s: forcar exclusao   Esc/n: cancelar")

	case modalDiscard:
		body = m.styles.modalTitle.Render("Descartar alteracoes") + "\n\n" +
			m.styles.modalWarn.Render(fmt.Sprintf("Descartar todas as alteracoes em '%s'?", m.modalTarget)) + "\n" +
			"Esta acao nao pode ser desfeita.\n\n" +
			m.styles.itemMuted.Render("Enter/s: descartar   Esc/n: cancelar")

	case modalStashSwitch:
		opts := []string{
			"[1] Fazer stash das alteracoes e trocar (recomendado)",
			"[2] Trocar mesmo assim",
			"[3] Cancelar",
		}
		var renderedOpts []string
		for i, o := range opts {
			if i == m.modalChoice {
				renderedOpts = append(renderedOpts, m.styles.modalOptionSel.Render(o))
			} else {
				renderedOpts = append(renderedOpts, m.styles.modalOption.Render(o))
			}
		}
		body = m.styles.modalTitle.Render("Diretorio com alteracoes pendentes") + "\n\n" +
			fmt.Sprintf("Ao trocar para '%s', o que deseja fazer?", m.modalTarget) + "\n\n" +
			strings.Join(renderedOpts, "\n") + "\n\n" +
			m.styles.itemMuted.Render("j/k: navegar   Enter ou 1/2/3: escolher   Esc: cancelar")
	}

	box := m.styles.modalBox.MaxWidth(max(40, m.width-6)).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) renderNotARepo() string {
	msg := m.styles.modalTitle.Render("Nenhum repositorio Git encontrado") + "\n\n" +
		fmt.Sprintf("O diretorio '%s' nao pertence a um repositorio Git.", m.client.Dir()) + "\n" +
		"Execute 'git init' ou abra o go-vit dentro de um projeto Git.\n\n" +
		m.styles.itemMuted.Render("R: verificar novamente   q: sair")
	box := m.styles.modalBox.Render(msg)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func visibleWindow(cursor, total, height int) (int, int) {
	if total <= height {
		return 0, total
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}

func truncateText(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}
