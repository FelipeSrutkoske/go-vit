package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitTest(t, dir, "init", "-b", "main")
	runGitTest(t, dir, "config", "user.name", "Test User")
	runGitTest(t, dir, "config", "user.email", "test@example.com")
	return dir
}

func writeTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", full, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func TestCheckGitInstalled(t *testing.T) {
	if err := CheckGitInstalled(); err != nil {
		t.Fatalf("CheckGitInstalled failed: %v", err)
	}
}

func TestInspectRepo_NotARepo(t *testing.T) {
	dir := t.TempDir()
	client := NewClient(dir)
	state, err := client.InspectRepo(context.Background())
	if err != nil {
		t.Fatalf("InspectRepo error: %v", err)
	}
	if state.Kind != KindNotARepo {
		t.Errorf("Kind = %v, want %v", state.Kind, KindNotARepo)
	}
}

func TestInspectRepo_UnbornStageUnstageCommit(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	client := NewClient(dir)

	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if state.Kind != KindUnborn {
		t.Errorf("Kind = %v, want %v", state.Kind, KindUnborn)
	}
	if state.Branch != "main" {
		t.Errorf("Branch = %q, want main", state.Branch)
	}

	commits, err := client.ListCommits(ctx, "", 10)
	if err != nil {
		t.Fatalf("ListCommits on unborn: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits on unborn, got %d", len(commits))
	}

	writeTestFile(t, dir, "hello.txt", "world\n")
	if err := client.StageFile(ctx, "hello.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}

	if err := client.UnstageFile(ctx, "hello.txt"); err != nil {
		t.Fatalf("UnstageFile on unborn: %v", err)
	}
	statuses, err := client.ListStatus(ctx)
	if err != nil || len(statuses) != 1 || !statuses[0].IsUntracked {
		t.Fatalf("expected untracked hello.txt after unstage, got %+v (err=%v)", statuses, err)
	}

	if err := client.StageFile(ctx, "hello.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}
	if err := client.Commit(ctx, "initial commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	state, err = client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo after commit: %v", err)
	}
	if state.Kind != KindNormal {
		t.Errorf("Kind = %v, want %v", state.Kind, KindNormal)
	}
	if state.HeadHash == "" || state.LastCommit != "initial commit" {
		t.Errorf("unexpected HeadHash=%q LastCommit=%q", state.HeadHash, state.LastCommit)
	}
	if state.ChangedCount != 0 {
		t.Errorf("ChangedCount = %d, want 0", state.ChangedCount)
	}
}

func TestInspectRepo_Subdir(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "sub/nested/file.txt", "v1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	writeTestFile(t, dir, "sub/nested/file.txt", "v2\n")
	subdir := filepath.Join(dir, "sub", "nested")
	client := NewClient(subdir)

	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo from subdir: %v", err)
	}
	if state.RootDir != dir {
		t.Errorf("RootDir = %q, want %q", state.RootDir, dir)
	}
	if state.ChangedCount != 1 {
		t.Errorf("ChangedCount = %d, want 1", state.ChangedCount)
	}

	statuses, err := client.ListStatus(ctx)
	if err != nil || len(statuses) != 1 {
		t.Fatalf("ListStatus from subdir: %+v (err=%v)", statuses, err)
	}
	diff, err := client.DiffFile(ctx, statuses[0])
	if err != nil {
		t.Fatalf("DiffFile from subdir: %v", err)
	}
	if !strings.Contains(diff, "+v2") || !strings.Contains(diff, "-v1") {
		t.Errorf("unexpected diff:\n%s", diff)
	}
}

func TestInspectRepo_DetachedHEAD(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.txt", "1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "c1")
	writeTestFile(t, dir, "a.txt", "2\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "c2")

	runGitTest(t, dir, "checkout", "HEAD~1")

	client := NewClient(dir)
	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo detached: %v", err)
	}
	if state.Kind != KindDetached {
		t.Errorf("Kind = %v, want %v", state.Kind, KindDetached)
	}
	if state.HeadHash == "" || state.LastCommit != "c1" {
		t.Errorf("HeadHash=%q LastCommit=%q, want c1", state.HeadHash, state.LastCommit)
	}
}

func TestBranches_CreateSwitchDeleteAndCommits(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "base.txt", "base\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "base commit")

	client := NewClient(dir)
	if err := client.CreateBranch(ctx, "feature/test"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	writeTestFile(t, dir, "feat.txt", "feat\n")
	if err := client.StageFile(ctx, "feat.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}
	if err := client.Commit(ctx, "feat commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	branches, err := client.ListBranches(ctx, false)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("got %d branches, want 2", len(branches))
	}

	commits, err := client.ListCommits(ctx, "feature/test", 10)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 2 || commits[0].Subject != "feat commit" {
		t.Errorf("unexpected commits: %+v", commits)
	}

	if err := client.SwitchBranch(ctx, "main"); err != nil {
		t.Fatalf("SwitchBranch: %v", err)
	}
	if err := client.DeleteBranch(ctx, "feature/test", false); err == nil {
		t.Error("expected non-force delete of unmerged branch to fail")
	}
	if err := client.DeleteBranch(ctx, "feature/test", true); err != nil {
		t.Fatalf("force DeleteBranch: %v", err)
	}
}

func TestMergeConflictAndResolution(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "conflict.txt", "line 1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "base")

	client := NewClient(dir)
	if err := client.CreateBranch(ctx, "theirs"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	writeTestFile(t, dir, "conflict.txt", "from theirs\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "theirs change")

	if err := client.SwitchBranch(ctx, "main"); err != nil {
		t.Fatalf("SwitchBranch main: %v", err)
	}
	writeTestFile(t, dir, "conflict.txt", "from ours\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "ours change")

	res, err := client.MergeBranch(ctx, "theirs")
	if err != nil {
		t.Fatalf("MergeBranch returned unexpected error: %v", err)
	}
	if !res.Conflicted {
		t.Fatalf("expected Conflicted=true, got false (output=%q)", res.Output)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0] != "conflict.txt" {
		t.Fatalf("unexpected Conflicts: %v", res.Conflicts)
	}

	statuses, err := client.ListStatus(ctx)
	if err != nil || len(statuses) != 1 || !statuses[0].IsConflicted {
		t.Fatalf("expected conflicted status, got %+v (err=%v)", statuses, err)
	}

	diff, err := client.DiffFile(ctx, statuses[0])
	if err != nil {
		t.Fatalf("DiffFile on conflict: %v", err)
	}
	if !strings.Contains(diff, "<<<<<<<") || !strings.Contains(diff, ">>>>>>>") {
		t.Errorf("expected conflict markers in diff, got:\n%s", diff)
	}

	if err := client.ResolveConflict(ctx, "conflict.txt", SideTheirs); err != nil {
		t.Fatalf("ResolveConflict: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "conflict.txt"))
	if err != nil {
		t.Fatalf("read resolved file: %v", err)
	}
	if string(data) != "from theirs\n" {
		t.Errorf("resolved content = %q, want %q", string(data), "from theirs\n")
	}

	if err := client.Commit(ctx, "resolve conflict"); err != nil {
		t.Fatalf("Commit after resolution: %v", err)
	}

	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if state.ConflictCount != 0 || state.ChangedCount != 0 {
		t.Errorf("expected clean repo after resolution, got %+v", state)
	}
}

func TestDiffFile_StagedUnstagedUntrackedUnicode(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "tracked file.txt", "one\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	writeTestFile(t, dir, "tracked file.txt", "two\n")
	runGitTest(t, dir, "add", "tracked file.txt")
	writeTestFile(t, dir, "tracked file.txt", "three\n")

	writeTestFile(t, dir, "café/notas finais.txt", "olá mundo\n")

	client := NewClient(dir)
	statuses, err := client.ListStatus(ctx)
	if err != nil {
		t.Fatalf("ListStatus: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("got %d statuses, want 2: %+v", len(statuses), statuses)
	}

	for _, st := range statuses {
		diff, err := client.DiffFile(ctx, st)
		if err != nil {
			t.Fatalf("DiffFile(%s): %v", st.Path, err)
		}
		if st.IsUntracked {
			if !strings.Contains(diff, "+olá mundo") {
				t.Errorf("untracked diff missing content:\n%s", diff)
			}
		} else {
			if !strings.Contains(diff, "=== Staged ===") || !strings.Contains(diff, "=== Unstaged ===") {
				t.Errorf("MM diff missing staged/unstaged sections:\n%s", diff)
			}
		}
	}
}

func TestStashPushAndPop(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "f.txt", "v1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	writeTestFile(t, dir, "f.txt", "v2\n")
	writeTestFile(t, dir, "untracked.txt", "temp\n")

	client := NewClient(dir)
	if err := client.StashPush(ctx, "wip"); err != nil {
		t.Fatalf("StashPush: %v", err)
	}
	state, err := client.InspectRepo(ctx)
	if err != nil || state.ChangedCount != 0 {
		t.Fatalf("expected clean repo after stash, got %+v (err=%v)", state, err)
	}

	if err := client.StashPop(ctx); err != nil {
		t.Fatalf("StashPop: %v", err)
	}
	state, err = client.InspectRepo(ctx)
	if err != nil || state.ChangedCount != 2 {
		t.Fatalf("expected 2 changed files after stash pop, got %+v (err=%v)", state, err)
	}
}

func TestDiscardFile(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.txt", "original\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	writeTestFile(t, dir, "a.txt", "modified\n")
	writeTestFile(t, dir, "new.txt", "untracked\n")

	client := NewClient(dir)
	statuses, err := client.ListStatus(ctx)
	if err != nil {
		t.Fatalf("ListStatus: %v", err)
	}
	for _, st := range statuses {
		if err := client.DiscardFile(ctx, st); err != nil {
			t.Fatalf("DiscardFile(%s): %v", st.Path, err)
		}
	}

	state, err := client.InspectRepo(ctx)
	if err != nil || state.ChangedCount != 0 {
		t.Fatalf("expected 0 changes after discard, got %+v (err=%v)", state, err)
	}
}

func TestFetchAndPull_LocalBareRemote(t *testing.T) {
	ctx := context.Background()
	bare := t.TempDir()
	runGitTest(t, bare, "init", "--bare", "-b", "main")

	repoA := initTestRepo(t)
	runGitTest(t, repoA, "remote", "add", "origin", bare)
	writeTestFile(t, repoA, "readme.txt", "v1\n")
	runGitTest(t, repoA, "add", ".")
	runGitTest(t, repoA, "commit", "-m", "c1")
	runGitTest(t, repoA, "push", "-u", "origin", "main")

	repoBParent := t.TempDir()
	repoB := filepath.Join(repoBParent, "clone")
	cmd := exec.Command("git", "clone", bare, repoB)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone failed: %v\n%s", err, string(out))
	}
	runGitTest(t, repoB, "config", "user.name", "User B")
	runGitTest(t, repoB, "config", "user.email", "b@example.com")

	writeTestFile(t, repoA, "readme.txt", "v2\n")
	runGitTest(t, repoA, "add", ".")
	runGitTest(t, repoA, "commit", "-m", "c2")
	runGitTest(t, repoA, "push")

	clientB := NewClient(repoB)
	if err := clientB.Fetch(ctx); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	state, err := clientB.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if state.Behind != 1 {
		t.Errorf("Behind = %d, want 1", state.Behind)
	}

	branches, err := clientB.ListBranches(ctx, true)
	if err != nil {
		t.Fatalf("ListBranches(includeRemote=true): %v", err)
	}
	hasRemote := false
	for _, b := range branches {
		if b.IsRemote && b.Name == "origin/main" {
			hasRemote = true
		}
	}
	if !hasRemote {
		t.Errorf("expected origin/main in remote branches: %+v", branches)
	}

	if err := clientB.Pull(ctx); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	state, err = clientB.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo after pull: %v", err)
	}
	if state.Behind != 0 || state.LastCommit != "c2" {
		t.Errorf("unexpected state after pull: %+v", state)
	}
}
