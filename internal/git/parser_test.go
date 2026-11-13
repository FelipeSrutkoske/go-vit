package git

import (
	"testing"
)

func TestParseStatusPorcelainZ_BranchHeaders(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantKind   RepoKind
		wantBranch string
		wantAhead  int
		wantBehind int
	}{
		{
			name:       "normal with ahead and behind",
			input:      "## main...origin/main [ahead 2, behind 3]\x00",
			wantKind:   KindNormal,
			wantBranch: "main",
			wantAhead:  2,
			wantBehind: 3,
		},
		{
			name:       "normal ahead only",
			input:      "## feat/ui...origin/feat/ui [ahead 5]\x00",
			wantKind:   KindNormal,
			wantBranch: "feat/ui",
			wantAhead:  5,
		},
		{
			name:       "normal behind only",
			input:      "## main...origin/main [behind 4]\x00",
			wantKind:   KindNormal,
			wantBranch: "main",
			wantBehind: 4,
		},
		{
			name:       "normal upstream gone",
			input:      "## topic...origin/topic [gone]\x00",
			wantKind:   KindNormal,
			wantBranch: "topic",
		},
		{
			name:       "normal no upstream",
			input:      "## local-only\x00",
			wantKind:   KindNormal,
			wantBranch: "local-only",
		},
		{
			name:       "unborn no commits yet",
			input:      "## No commits yet on main\x00",
			wantKind:   KindUnborn,
			wantBranch: "main",
		},
		{
			name:       "unborn initial commit",
			input:      "## Initial commit on master\x00",
			wantKind:   KindUnborn,
			wantBranch: "master",
		},
		{
			name:     "detached HEAD",
			input:    "## HEAD (no branch)\x00",
			wantKind: KindDetached,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hdr, files, err := parseStatusPorcelainZ([]byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(files) != 0 {
				t.Fatalf("expected 0 files, got %d", len(files))
			}
			if hdr.Kind != tc.wantKind {
				t.Errorf("Kind = %v, want %v", hdr.Kind, tc.wantKind)
			}
			if hdr.Branch != tc.wantBranch {
				t.Errorf("Branch = %q, want %q", hdr.Branch, tc.wantBranch)
			}
			if hdr.Ahead != tc.wantAhead {
				t.Errorf("Ahead = %d, want %d", hdr.Ahead, tc.wantAhead)
			}
			if hdr.Behind != tc.wantBehind {
				t.Errorf("Behind = %d, want %d", hdr.Behind, tc.wantBehind)
			}
		})
	}
}

func TestParseStatusPorcelainZ_FilesAndConflicts(t *testing.T) {
	input := "## main\x00" +
		"M  staged.go\x00" +
		" M unstaged.go\x00" +
		"MM both.go\x00" +
		"A  new file with spaces.txt\x00" +
		"?? café/relatório-ñ.md\x00" +
		"R  novo nome.txt\x00antigo nome.txt\x00" +
		"C  cópia.txt\x00original.txt\x00" +
		"UU conflict_uu.txt\x00" +
		"AA conflict_aa.txt\x00" +
		"DD conflict_dd.txt\x00" +
		"AU conflict_au.txt\x00" +
		"UA conflict_ua.txt\x00" +
		"DU conflict_du.txt\x00" +
		"UD conflict_ud.txt\x00"

	hdr, files, err := parseStatusPorcelainZ([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hdr.Kind != KindNormal || hdr.Branch != "main" {
		t.Fatalf("unexpected header: %+v", hdr)
	}
	if len(files) != 14 {
		t.Fatalf("got %d files, want 14", len(files))
	}

	if !files[0].IsStaged || files[0].IsUnstaged || files[0].IsConflicted || files[0].IsUntracked {
		t.Errorf("files[0] flags unexpected: %+v", files[0])
	}
	if files[1].IsStaged || !files[1].IsUnstaged {
		t.Errorf("files[1] flags unexpected: %+v", files[1])
	}
	if !files[2].IsStaged || !files[2].IsUnstaged {
		t.Errorf("files[2] flags unexpected: %+v", files[2])
	}
	if files[3].Path != "new file with spaces.txt" || !files[3].IsStaged {
		t.Errorf("files[3] unexpected: %+v", files[3])
	}
	if files[4].Path != "café/relatório-ñ.md" || !files[4].IsUntracked || files[4].IsStaged || files[4].IsUnstaged {
		t.Errorf("files[4] unexpected: %+v", files[4])
	}
	if files[5].Path != "novo nome.txt" || files[5].OldPath != "antigo nome.txt" || !files[5].IsStaged {
		t.Errorf("files[5] rename unexpected: %+v", files[5])
	}
	if files[6].Path != "cópia.txt" || files[6].OldPath != "original.txt" || !files[6].IsStaged {
		t.Errorf("files[6] copy unexpected: %+v", files[6])
	}
	for i := 7; i < 14; i++ {
		f := files[i]
		if !f.IsConflicted || f.IsStaged || f.IsUnstaged || f.IsUntracked {
			t.Errorf("conflict files[%d] (%s) flags unexpected: %+v", i, f.Path, f)
		}
	}
}

func TestParseStatusPorcelainZ_Errors(t *testing.T) {
	if _, _, err := parseStatusPorcelainZ([]byte("bad\x00")); err == nil {
		t.Error("expected error for malformed record")
	}
	if _, _, err := parseStatusPorcelainZ([]byte("R  new.txt\x00")); err == nil {
		t.Error("expected error for rename missing original path")
	}
}

func TestParseBranches(t *testing.T) {
	raw := "*\x1frefs/heads/main\x1fmain\x1forigin/main\x1f[ahead 1, behind 2]\x1fabc1234\x1finit commit\x1f2 hours ago\x00" +
		" \x1frefs/remotes/origin/HEAD\x1forigin/HEAD\x1f\x1f\x1fabc1234\x1finit commit\x1f2 hours ago\x00" +
		" \x1frefs/remotes/origin/main\x1forigin/main\x1f\x1f\x1fabc1234\x1finit commit\x1f2 hours ago\x00"

	branches := parseBranches([]byte(raw))
	if len(branches) != 2 {
		t.Fatalf("got %d branches, want 2", len(branches))
	}
	if !branches[0].IsCurrent || branches[0].IsRemote || branches[0].Ahead != 1 || branches[0].Behind != 2 {
		t.Errorf("branches[0] unexpected: %+v", branches[0])
	}
	if branches[1].IsCurrent || !branches[1].IsRemote || branches[1].Name != "origin/main" {
		t.Errorf("branches[1] unexpected: %+v", branches[1])
	}
}

func TestParseCommits(t *testing.T) {
	raw := "111aaaa\x1ffirst commit\x1fAlice\x1f1 day ago\x00222bbbb\x1fsecond\x1fBob\x1f3 hours ago\x00"
	commits := parseCommits([]byte(raw))
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}
	if commits[0].Hash != "111aaaa" || commits[0].Subject != "first commit" || commits[0].Author != "Alice" {
		t.Errorf("commits[0] unexpected: %+v", commits[0])
	}
	if commits[1].Hash != "222bbbb" || commits[1].RelativeDate != "3 hours ago" {
		t.Errorf("commits[1] unexpected: %+v", commits[1])
	}
}
