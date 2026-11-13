package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
)

type tabKind int

const (
	tabBranches tabKind = iota
	tabChanges
)

type modalKind int

const (
	modalNone modalKind = iota
	modalHelp
	modalCreateBranch
	modalDeleteBranch
	modalForceDelete
	modalStashSwitch
	modalCommit
	modalDiscard
)

type Model struct {
	client        *git.Client
	state         git.RepoState
	branches      []git.Branch
	files         []git.FileStatus
	commits       []git.Commit
	commitsRef    string
	diffText      string
	diffPath      string
	activeTab     tabKind
	branchIdx     int
	fileIdx       int
	includeRemote bool

	modal       modalKind
	modalTarget string
	modalFile   git.FileStatus
	modalChoice int
	input       textinput.Model

	viewport viewport.Model
	spinner  spinner.Model
	help     help.Model
	keys     keyMap
	styles   styles

	width    int
	height   int
	loading  bool
	status   string
	errMsg   string
	quitting bool
}

func NewModel(client *git.Client) Model {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	vp := viewport.New()
	ti := textinput.New()
	ti.CharLimit = 120
	ti.SetWidth(44)

	h := help.New()
	st := defaultStyles()

	return Model{
		client:   client,
		viewport: vp,
		spinner:  sp,
		input:    ti,
		help:     h,
		keys:     defaultKeyMap(),
		styles:   st,
		loading:  true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadRepoCmd(m.client, m.includeRemote))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		m.help.SetWidth(max(20, msg.Width-8))
		m.syncRightPane()
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case repoLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.state = msg.state
		m.branches = msg.branches
		m.files = msg.status
		m.clampCursors()
		return m, m.refreshSelectionCmd()

	case commitsLoadedMsg:
		m.commitsRef = msg.ref
		if msg.err != nil {
			m.commits = nil
			m.viewport.SetContent(m.styles.bannerError.Render("Erro ao carregar commits: " + msg.err.Error()))
			return m, nil
		}
		m.commits = msg.commits
		m.syncRightPane()
		return m, nil

	case diffLoadedMsg:
		m.diffPath = msg.path
		if msg.err != nil {
			m.diffText = ""
			m.viewport.SetContent(m.styles.bannerError.Render("Erro ao carregar diff: " + msg.err.Error()))
			return m, nil
		}
		m.diffText = msg.diff
		m.syncRightPane()
		return m, nil

	case actionDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = ""
			return m, nil
		}
		m.errMsg = ""
		m.status = msg.message
		return m, m.reloadRepo()

	case mergeDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = ""
			return m, nil
		}
		m.errMsg = ""
		if msg.res.Conflicted {
			m.activeTab = tabChanges
			m.status = fmt.Sprintf(
				"Conflitos ao mesclar %s (%d arquivo(s)). Resolva na aba Alteracoes & Conflitos.",
				msg.branch,
				len(msg.res.Conflicts),
			)
		} else {
			m.status = fmt.Sprintf("Branch %s mesclada com sucesso", msg.branch)
		}
		return m, m.reloadRepo()

	case deleteBranchErrMsg:
		m.loading = false
		if !msg.force {
			m.modal = modalForceDelete
			m.modalTarget = msg.branch
			m.errMsg = ""
			return m, nil
		}
		m.errMsg = msg.err.Error()
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.errMsg = "Erro ao abrir editor: " + msg.err.Error()
		}
		return m, m.reloadRepo()

	case tea.KeyPressMsg:
		if m.modal != modalNone {
			return m.handleModalKey(msg)
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) reloadRepo() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Tick, loadRepoCmd(m.client, m.includeRemote))
}

func (m *Model) clampCursors() {
	if len(m.branches) == 0 {
		m.branchIdx = 0
	} else if m.branchIdx >= len(m.branches) {
		m.branchIdx = len(m.branches) - 1
	}
	if len(m.files) == 0 {
		m.fileIdx = 0
	} else if m.fileIdx >= len(m.files) {
		m.fileIdx = len(m.files) - 1
	}
}

func (m *Model) refreshSelectionCmd() tea.Cmd {
	if m.state.Kind == git.KindNotARepo {
		return nil
	}
	if m.activeTab == tabBranches {
		if len(m.branches) == 0 {
			m.commits = nil
			m.commitsRef = ""
			m.syncRightPane()
			return nil
		}
		return loadCommitsCmd(m.client, m.branches[m.branchIdx].Name)
	}
	if len(m.files) == 0 {
		m.diffText = ""
		m.diffPath = ""
		m.syncRightPane()
		return nil
	}
	return loadDiffCmd(m.client, m.files[m.fileIdx])
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		m.quitting = true
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Help) {
		m.modal = modalHelp
		return m, nil
	}
	if key.Matches(msg, m.keys.Refresh) {
		m.errMsg = ""
		m.status = "Atualizando..."
		return m, m.reloadRepo()
	}

	if m.state.Kind == git.KindNotARepo {
		return m, nil
	}

	// 1/2/e tem prioridade quando o arquivo selecionado esta em conflito
	if m.activeTab == tabChanges && len(m.files) > 0 && m.files[m.fileIdx].IsConflicted {
		f := m.files[m.fileIdx]
		switch {
		case key.Matches(msg, m.keys.KeepOurs):
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, resolveConflictCmd(m.client, f.Path, git.SideOurs))
		case key.Matches(msg, m.keys.AcceptTheirs):
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, resolveConflictCmd(m.client, f.Path, git.SideTheirs))
		case key.Matches(msg, m.keys.OpenEditor):
			return m, openEditorCmd(m.state.RootDir, f.Path)
		}
	}

	switch {
	case key.Matches(msg, m.keys.NextTab):
		if m.activeTab == tabBranches {
			m.activeTab = tabChanges
		} else {
			m.activeTab = tabBranches
		}
		return m, m.refreshSelectionCmd()
	case key.Matches(msg, m.keys.Tab1):
		if m.activeTab != tabBranches {
			m.activeTab = tabBranches
			return m, m.refreshSelectionCmd()
		}
		return m, nil
	case key.Matches(msg, m.keys.Tab2):
		if m.activeTab != tabChanges {
			m.activeTab = tabChanges
			return m, m.refreshSelectionCmd()
		}
		return m, nil
	case key.Matches(msg, m.keys.Fetch):
		m.loading = true
		m.errMsg = ""
		m.status = "Executando fetch..."
		return m, tea.Batch(m.spinner.Tick, fetchCmd(m.client))
	case key.Matches(msg, m.keys.Pull):
		m.loading = true
		m.errMsg = ""
		m.status = "Executando pull..."
		return m, tea.Batch(m.spinner.Tick, pullCmd(m.client))
	case key.Matches(msg, m.keys.PageUp):
		m.viewport.HalfPageUp()
		return m, nil
	case key.Matches(msg, m.keys.PageDown):
		m.viewport.HalfPageDown()
		return m, nil
	}

	if m.activeTab == tabBranches {
		return m.handleBranchKey(msg)
	}
	return m.handleChangesKey(msg)
}

func (m Model) handleBranchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.branchIdx > 0 {
			m.branchIdx--
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Down):
		if m.branchIdx < len(m.branches)-1 {
			m.branchIdx++
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Top):
		if len(m.branches) > 0 && m.branchIdx != 0 {
			m.branchIdx = 0
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Bottom):
		if len(m.branches) > 0 && m.branchIdx != len(m.branches)-1 {
			m.branchIdx = len(m.branches) - 1
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.ToggleRemotes):
		m.includeRemote = !m.includeRemote
		return m, m.reloadRepo()
	case key.Matches(msg, m.keys.NewBranch):
		m.modal = modalCreateBranch
		m.input.SetValue("")
		m.input.Placeholder = "nome-da-nova-branch"
		m.input.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Switch):
		if len(m.branches) == 0 {
			return m, nil
		}
		b := m.branches[m.branchIdx]
		if b.IsCurrent {
			return m, nil
		}
		if m.state.ChangedCount > 0 || m.state.ConflictCount > 0 {
			m.modal = modalStashSwitch
			m.modalTarget = b.Name
			m.modalChoice = 0
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, switchBranchCmd(m.client, b.Name))
	case key.Matches(msg, m.keys.DeleteBranch):
		if len(m.branches) == 0 {
			return m, nil
		}
		b := m.branches[m.branchIdx]
		if b.IsCurrent {
			m.errMsg = "Nao e possivel excluir a branch atual"
			return m, nil
		}
		if b.IsRemote {
			m.errMsg = "Exclusao disponivel apenas para branches locais"
			return m, nil
		}
		m.modal = modalDeleteBranch
		m.modalTarget = b.Name
		return m, nil
	case key.Matches(msg, m.keys.MergeBranch):
		if len(m.branches) == 0 {
			return m, nil
		}
		b := m.branches[m.branchIdx]
		if b.IsCurrent {
			return m, nil
		}
		m.loading = true
		m.errMsg = ""
		m.status = fmt.Sprintf("Mesclando %s...", b.Name)
		return m, tea.Batch(m.spinner.Tick, mergeBranchCmd(m.client, b.Name))
	}
	return m, nil
}

func (m Model) handleChangesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.fileIdx > 0 {
			m.fileIdx--
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Down):
		if m.fileIdx < len(m.files)-1 {
			m.fileIdx++
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Top):
		if len(m.files) > 0 && m.fileIdx != 0 {
			m.fileIdx = 0
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Bottom):
		if len(m.files) > 0 && m.fileIdx != len(m.files)-1 {
			m.fileIdx = len(m.files) - 1
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Stage):
		if len(m.files) == 0 {
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, toggleStageCmd(m.client, m.files[m.fileIdx]))
	case key.Matches(msg, m.keys.Commit):
		if m.state.ConflictCount > 0 {
			m.errMsg = "Resolva os conflitos antes de fazer commit"
			return m, nil
		}
		if m.state.StagedCount == 0 {
			m.errMsg = "Nenhum arquivo preparado (staged). Pressione Espaco ou 's' antes de commitar."
			return m, nil
		}
		m.modal = modalCommit
		m.input.SetValue("")
		m.input.Placeholder = "Mensagem do commit"
		m.input.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Stash):
		m.loading = true
		hasChanges := m.state.ChangedCount > 0
		return m, tea.Batch(m.spinner.Tick, stashToggleCmd(m.client, hasChanges))
	case key.Matches(msg, m.keys.Discard):
		if len(m.files) == 0 {
			return m, nil
		}
		m.modal = modalDiscard
		m.modalFile = m.files[m.fileIdx]
		m.modalTarget = m.modalFile.Path
		return m, nil
	}
	return m, nil
}

func (m Model) handleModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	switch m.modal {
	case modalHelp:
		if k == "esc" || k == "?" || k == "q" || k == "enter" {
			m.modal = modalNone
		}
		return m, nil

	case modalCreateBranch:
		switch k {
		case "esc":
			m.modal = modalNone
			m.input.Blur()
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			m.modal = modalNone
			m.input.Blur()
			if name == "" {
				return m, nil
			}
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, createBranchCmd(m.client, name))
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case modalCommit:
		switch k {
		case "esc":
			m.modal = modalNone
			m.input.Blur()
			return m, nil
		case "enter":
			message := strings.TrimSpace(m.input.Value())
			m.modal = modalNone
			m.input.Blur()
			if message == "" {
				m.errMsg = "A mensagem do commit nao pode estar vazia"
				return m, nil
			}
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, commitCmd(m.client, message))
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case modalDeleteBranch:
		switch k {
		case "esc", "n", "N":
			m.modal = modalNone
			return m, nil
		case "enter", "s", "S", "y", "Y":
			target := m.modalTarget
			m.modal = modalNone
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, deleteBranchCmd(m.client, target, false))
		}
		return m, nil

	case modalForceDelete:
		switch k {
		case "esc", "n", "N":
			m.modal = modalNone
			return m, nil
		case "enter", "f", "F", "s", "S", "y", "Y":
			target := m.modalTarget
			m.modal = modalNone
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, deleteBranchCmd(m.client, target, true))
		}
		return m, nil

	case modalDiscard:
		switch k {
		case "esc", "n", "N":
			m.modal = modalNone
			return m, nil
		case "enter", "s", "S", "y", "Y":
			f := m.modalFile
			m.modal = modalNone
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, discardFileCmd(m.client, f))
		}
		return m, nil

	case modalStashSwitch:
		switch k {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "up", "k":
			if m.modalChoice > 0 {
				m.modalChoice--
			}
			return m, nil
		case "down", "j":
			if m.modalChoice < 2 {
				m.modalChoice++
			}
			return m, nil
		case "1":
			m.modalChoice = 0
			return m.execStashSwitch()
		case "2":
			m.modalChoice = 1
			return m.execStashSwitch()
		case "3":
			m.modal = modalNone
			return m, nil
		case "enter":
			return m.execStashSwitch()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) execStashSwitch() (tea.Model, tea.Cmd) {
	target := m.modalTarget
	choice := m.modalChoice
	m.modal = modalNone
	switch choice {
	case 0:
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, stashAndSwitchCmd(m.client, target))
	case 1:
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, switchBranchCmd(m.client, target))
	default:
		return m, nil
	}
}
