package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLocalTimeout   = 15 * time.Second
	defaultNetworkTimeout = 60 * time.Second
	maxPreviewBytes       = 256 * 1024
)

type Client struct {
	dir string
}

func NewClient(dir string) *Client {
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	return &Client{dir: filepath.Clean(dir)}
}

func (c *Client) Dir() string {
	return c.dir
}

func CheckGitInstalled() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "--version")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git --version failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *Client) InspectRepo(ctx context.Context) (RepoState, error) {
	rootDir, isRepo, err := c.resolveRoot(ctx)
	if err != nil {
		return RepoState{}, err
	}
	if !isRepo {
		return RepoState{Kind: KindNotARepo}, nil
	}

	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "status", "--porcelain=v1", "--branch", "-z", "-uall")
	if err != nil {
		return RepoState{}, err
	}

	header, files, err := parseStatusPorcelainZ(stdout)
	if err != nil {
		return RepoState{}, err
	}

	state := RepoState{
		RootDir:      rootDir,
		RepoName:     filepath.Base(rootDir),
		Kind:         header.Kind,
		Branch:       header.Branch,
		Ahead:        header.Ahead,
		Behind:       header.Behind,
		ChangedCount: len(files),
	}

	for _, f := range files {
		if f.IsConflicted {
			state.ConflictCount++
		}
		if f.IsStaged {
			state.StagedCount++
		}
	}

	if state.Kind != KindUnborn {
		if hashOut, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "rev-parse", "--short", "HEAD"); err == nil {
			state.HeadHash = strings.TrimSpace(string(hashOut))
		}
		if subjOut, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "log", "-1", "--format=%s", "HEAD"); err == nil {
			state.LastCommit = strings.TrimSpace(string(subjOut))
		}
	}

	return state, nil
}

func (c *Client) ListBranches(ctx context.Context, includeRemote bool) ([]Branch, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return nil, err
	}

	args := []string{
		"for-each-ref",
		"--format=%(HEAD)%1f%(refname)%1f%(refname:short)%1f%(upstream:short)%1f%(upstream:track)%1f%(objectname:short)%1f%(contents:subject)%1f%(committerdate:relative)%00",
		"refs/heads",
	}
	if includeRemote {
		args = append(args, "refs/remotes")
	}

	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, args...)
	if err != nil {
		return nil, err
	}
	return parseBranches(stdout), nil
}

func (c *Client) ListCommits(ctx context.Context, ref string, limit int) ([]Commit, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	args := []string{
		"log",
		"-n", strconv.Itoa(limit),
		"--format=%h\x1f%s\x1f%an%x1f%ar%x00",
	}
	if strings.TrimSpace(ref) != "" {
		args = append(args, ref)
	}

	stdout, stderr, err := c.runGit(ctx, rootDir, defaultLocalTimeout, args...)
	if err != nil {
		if strings.Contains(string(stderr), "does not have any commits yet") {
			return nil, nil
		}
		return nil, err
	}
	return parseCommits(stdout), nil
}

func (c *Client) ListStatus(ctx context.Context) ([]FileStatus, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return nil, err
	}

	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "status", "--porcelain=v1", "--branch", "-z", "-uall")
	if err != nil {
		return nil, err
	}
	_, files, err := parseStatusPorcelainZ(stdout)
	return files, err
}

func (c *Client) DiffFile(ctx context.Context, file FileStatus) (string, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return "", err
	}

	if file.IsConflicted {
		return readPreviewFile(rootDir, file.Path)
	}

	if file.IsUntracked {
		return c.diffUntracked(ctx, rootDir, file.Path)
	}

	if file.IsStaged && file.IsUnstaged {
		stagedOut, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--cached", "--", file.Path)
		if err != nil {
			return "", err
		}
		unstagedOut, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--", file.Path)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		if len(stagedOut) > 0 {
			b.WriteString("=== Staged ===\n")
			b.Write(stagedOut)
		}
		if len(unstagedOut) > 0 {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("=== Unstaged ===\n")
			b.Write(unstagedOut)
		}
		return b.String(), nil
	}

	if file.IsStaged {
		stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--cached", "--", file.Path)
		if err != nil {
			return "", err
		}
		return string(stdout), nil
	}

	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--", file.Path)
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

func (c *Client) diffUntracked(ctx context.Context, rootDir, relPath string) (string, error) {
	absPath, err := safeJoin(rootDir, relPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Untracked directory: %s (%d entries)\n", relPath, len(entries))
		for _, e := range entries {
			b.WriteString("  ")
			b.WriteString(e.Name())
			if e.IsDir() {
				b.WriteByte('/')
			}
			b.WriteByte('\n')
		}
		return b.String(), nil
	}

	// git diff --no-index retorna exit code 1 quando ha diferencas
	tctx, cancel := withTimeout(ctx, defaultLocalTimeout)
	defer cancel()

	cmd := exec.CommandContext(tctx, "git", "-C", rootDir, "diff", "--no-index", "--", "/dev/null", relPath)
	cmd.Env = gitEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && stdout.Len() > 0 {
			return truncateOutput(stdout.String(), maxPreviewBytes), nil
		}
		return "", fmt.Errorf("git diff --no-index: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return truncateOutput(stdout.String(), maxPreviewBytes), nil
}

func (c *Client) SwitchBranch(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "switch", name)
	return err
}

func (c *Client) CreateBranch(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "switch", "-c", name)
	return err
}

func (c *Client) DeleteBranch(ctx context.Context, name string, force bool) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "branch", flag, name)
	return err
}

func (c *Client) MergeBranch(ctx context.Context, name string) (MergeResult, error) {
	if strings.TrimSpace(name) == "" {
		return MergeResult{}, errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return MergeResult{}, err
	}

	stdout, stderr, mergeErr := c.runGit(ctx, rootDir, defaultLocalTimeout, "merge", "--no-edit", name)
	combined := combineOutputs(stdout, stderr)
	if mergeErr == nil {
		return MergeResult{Conflicted: false, Output: combined}, nil
	}

	statuses, statusErr := c.ListStatus(ctx)
	if statusErr == nil {
		var conflicts []string
		for _, s := range statuses {
			if s.IsConflicted {
				conflicts = append(conflicts, s.Path)
			}
		}
		if len(conflicts) > 0 {
			return MergeResult{
				Conflicted: true,
				Conflicts:  conflicts,
				Output:     combined,
			}, nil
		}
	}

	return MergeResult{Output: combined}, mergeErr
}

func (c *Client) StageFile(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "add", "--", path)
	return err
}

func (c *Client) UnstageFile(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}

	if c.hasHead(ctx, rootDir) {
		_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "reset", "HEAD", "--", path)
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "rm", "--cached", "--", path)
	return err
}

func (c *Client) DiscardFile(ctx context.Context, file FileStatus) error {
	if strings.TrimSpace(file.Path) == "" {
		return errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}

	if file.IsUntracked {
		absPath, err := safeJoin(rootDir, file.Path)
		if err != nil {
			return err
		}
		return os.RemoveAll(absPath)
	}

	if file.IsStaged && !file.IsUnstaged && file.IndexStatus == 'A' {
		if err := c.UnstageFile(ctx, file.Path); err != nil {
			return err
		}
		absPath, err := safeJoin(rootDir, file.Path)
		if err != nil {
			return err
		}
		return os.RemoveAll(absPath)
	}

	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "checkout", "--", file.Path)
	return err
}

func (c *Client) Commit(ctx context.Context, message string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("commit message cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "commit", "-m", message)
	return err
}

func (c *Client) StashPush(ctx context.Context, message string) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	args := []string{"stash", "push", "-u"}
	if strings.TrimSpace(message) != "" {
		args = append(args, "-m", message)
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, args...)
	return err
}

func (c *Client) StashPop(ctx context.Context) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "stash", "pop")
	return err
}

func (c *Client) ResolveConflict(ctx context.Context, path string, side ConflictSide) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}

	var sideFlag string
	switch side {
	case SideOurs:
		sideFlag = "--ours"
	case SideTheirs:
		sideFlag = "--theirs"
	default:
		return fmt.Errorf("invalid conflict side: %d", side)
	}

	if _, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "checkout", sideFlag, "--", path); err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "add", "--", path)
	return err
}

func (c *Client) Fetch(ctx context.Context) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultNetworkTimeout, "fetch", "--prune")
	return err
}

func (c *Client) Pull(ctx context.Context) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultNetworkTimeout, "pull")
	return err
}

func (c *Client) resolveRoot(ctx context.Context) (string, bool, error) {
	stdout, stderr, err := c.runGit(ctx, c.dir, defaultLocalTimeout, "rev-parse", "--show-toplevel")
	if err != nil {
		if isNotARepoError(string(stderr)) {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(string(stdout)), true, nil
}

func (c *Client) requireRoot(ctx context.Context) (string, error) {
	rootDir, isRepo, err := c.resolveRoot(ctx)
	if err != nil {
		return "", err
	}
	if !isRepo {
		return "", fmt.Errorf("%s is not a git repository", c.dir)
	}
	return rootDir, nil
}

func (c *Client) hasHead(ctx context.Context, rootDir string) bool {
	_, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "rev-parse", "--verify", "HEAD")
	return err == nil
}

func (c *Client) runGit(ctx context.Context, dir string, timeout time.Duration, args ...string) ([]byte, []byte, error) {
	tctx, cancel := withTimeout(ctx, timeout)
	defer cancel()

	fullArgs := make([]string, 0, len(args)+2)
	fullArgs = append(fullArgs, "-C", dir)
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(tctx, "git", fullArgs...)
	cmd.Env = gitEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("git %s: %w: %s", args[0], err, errMsg)
		}
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("git %s: %w", args[0], err)
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

func gitEnv() []string {
	return append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
}

func isNotARepoError(stderr string) bool {
	lower := strings.ToLower(stderr)
	return strings.Contains(lower, "not a git repository")
}

func combineOutputs(stdout, stderr []byte) string {
	out := strings.TrimSpace(string(stdout))
	errStr := strings.TrimSpace(string(stderr))
	switch {
	case out != "" && errStr != "":
		return out + "\n" + errStr
	case out != "":
		return out
	default:
		return errStr
	}
}

func safeJoin(rootDir, relPath string) (string, error) {
	cleanRel := filepath.Clean(relPath)
	if filepath.IsAbs(cleanRel) || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid relative path: %q", relPath)
	}
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(absRoot, cleanRel)
	if joined != absRoot && !strings.HasPrefix(joined, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository root: %q", relPath)
	}
	return joined, nil
}

func readPreviewFile(rootDir, relPath string) (string, error) {
	absPath, err := safeJoin(rootDir, relPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	if len(data) > maxPreviewBytes {
		data = data[:maxPreviewBytes]
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return "(binary file)", nil
	}
	return string(data), nil
}

func truncateOutput(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "\n... (truncated)\n"
}
