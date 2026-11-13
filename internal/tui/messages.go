package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
)

type repoLoadedMsg struct {
	state    git.RepoState
	branches []git.Branch
	status   []git.FileStatus
	err      error
}

type commitsLoadedMsg struct {
	ref     string
	commits []git.Commit
	err     error
}

type diffLoadedMsg struct {
	path string
	diff string
	err  error
}

type actionDoneMsg struct {
	message string
	err     error
}

type mergeDoneMsg struct {
	branch string
	res    git.MergeResult
	err    error
}

type deleteBranchErrMsg struct {
	branch string
	force  bool
	err    error
}

type editorFinishedMsg struct {
	err error
}

func loadRepoCmd(client *git.Client, includeRemote bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		state, err := client.InspectRepo(ctx)
		if err != nil {
			return repoLoadedMsg{err: err}
		}
		if state.Kind == git.KindNotARepo {
			return repoLoadedMsg{state: state}
		}
		branches, err := client.ListBranches(ctx, includeRemote)
		if err != nil {
			return repoLoadedMsg{state: state, err: err}
		}
		status, err := client.ListStatus(ctx)
		if err != nil {
			return repoLoadedMsg{state: state, branches: branches, err: err}
		}
		return repoLoadedMsg{
			state:    state,
			branches: branches,
			status:   status,
		}
	}
}

func loadCommitsCmd(client *git.Client, ref string) tea.Cmd {
	return func() tea.Msg {
		commits, err := client.ListCommits(context.Background(), ref, 50)
		return commitsLoadedMsg{ref: ref, commits: commits, err: err}
	}
}

func loadDiffCmd(client *git.Client, file git.FileStatus) tea.Cmd {
	return func() tea.Msg {
		diff, err := client.DiffFile(context.Background(), file)
		return diffLoadedMsg{path: file.Path, diff: diff, err: err}
	}
}

func switchBranchCmd(client *git.Client, name string) tea.Cmd {
	return func() tea.Msg {
		if err := client.SwitchBranch(context.Background(), name); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Branch alterada para %s", name)}
	}
}

func stashAndSwitchCmd(client *git.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := client.StashPush(ctx, "go-vit: stash automatico antes de trocar para "+name); err != nil {
			return actionDoneMsg{err: err}
		}
		if err := client.SwitchBranch(ctx, name); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Stash salvo e branch alterada para %s", name)}
	}
}

func createBranchCmd(client *git.Client, name string) tea.Cmd {
	return func() tea.Msg {
		if err := client.CreateBranch(context.Background(), name); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Branch %s criada e selecionada", name)}
	}
}

func deleteBranchCmd(client *git.Client, name string, force bool) tea.Cmd {
	return func() tea.Msg {
		if err := client.DeleteBranch(context.Background(), name, force); err != nil {
			return deleteBranchErrMsg{branch: name, force: force, err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Branch %s excluida", name)}
	}
}

func mergeBranchCmd(client *git.Client, name string) tea.Cmd {
	return func() tea.Msg {
		res, err := client.MergeBranch(context.Background(), name)
		return mergeDoneMsg{branch: name, res: res, err: err}
	}
}

func toggleStageCmd(client *git.Client, file git.FileStatus) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if file.IsStaged && !file.IsUnstaged && !file.IsConflicted {
			if err := client.UnstageFile(ctx, file.Path); err != nil {
				return actionDoneMsg{err: err}
			}
			return actionDoneMsg{message: fmt.Sprintf("Removido do stage: %s", file.Path)}
		}
		if err := client.StageFile(ctx, file.Path); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Preparado (staged): %s", file.Path)}
	}
}

func commitCmd(client *git.Client, message string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Commit(context.Background(), message); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "Commit realizado com sucesso"}
	}
}

func stashToggleCmd(client *git.Client, hasChanges bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if hasChanges {
			if err := client.StashPush(ctx, "go-vit: stash rapido"); err != nil {
				return actionDoneMsg{err: err}
			}
			return actionDoneMsg{message: "Alteracoes guardadas no stash"}
		}
		if err := client.StashPop(ctx); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "Stash restaurado (pop)"}
	}
}

func discardFileCmd(client *git.Client, file git.FileStatus) tea.Cmd {
	return func() tea.Msg {
		if err := client.DiscardFile(context.Background(), file); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Alteracoes descartadas: %s", file.Path)}
	}
}

func resolveConflictCmd(client *git.Client, path string, side git.ConflictSide) tea.Cmd {
	return func() tea.Msg {
		if err := client.ResolveConflict(context.Background(), path, side); err != nil {
			return actionDoneMsg{err: err}
		}
		label := "nossos (ours)"
		if side == git.SideTheirs {
			label = "deles (theirs)"
		}
		return actionDoneMsg{message: fmt.Sprintf("Conflito resolvido (%s): %s", label, path)}
	}
}

func fetchCmd(client *git.Client) tea.Cmd {
	return func() tea.Msg {
		if err := client.Fetch(context.Background()); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "Fetch concluido"}
	}
}

func pullCmd(client *git.Client) tea.Cmd {
	return func() tea.Msg {
		if err := client.Pull(context.Background()); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: "Pull concluido"}
	}
}

func openEditorCmd(rootDir, relPath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	fullPath := filepath.Join(rootDir, filepath.FromSlash(relPath))
	c := exec.Command(editor, fullPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}
