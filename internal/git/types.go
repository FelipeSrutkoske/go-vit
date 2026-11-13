package git

type RepoKind int

const (
	KindNormal RepoKind = iota
	KindDetached
	KindUnborn
	KindNotARepo
)

func (k RepoKind) String() string {
	switch k {
	case KindNormal:
		return "normal"
	case KindDetached:
		return "detached"
	case KindUnborn:
		return "unborn"
	case KindNotARepo:
		return "not_a_repo"
	default:
		return "unknown"
	}
}

type RepoState struct {
	RootDir       string
	RepoName      string
	Kind          RepoKind
	Branch        string
	HeadHash      string
	LastCommit    string
	Ahead         int
	Behind        int
	ChangedCount  int
	ConflictCount int
	StagedCount   int
}

type Branch struct {
	Name         string
	IsCurrent    bool
	IsRemote     bool
	Upstream     string
	Ahead        int
	Behind       int
	LastHash     string
	LastSubject  string
	RelativeDate string
}

type FileStatus struct {
	Path         string
	OldPath      string
	IndexStatus  byte
	WorkStatus   byte
	IsConflicted bool
	IsUntracked  bool
	IsStaged     bool
	IsUnstaged   bool
}

type Commit struct {
	Hash         string
	Subject      string
	Author       string
	RelativeDate string
}

type MergeResult struct {
	Conflicted bool
	Conflicts  []string
	Output     string
}

type ConflictSide int

const (
	SideOurs ConflictSide = iota
	SideTheirs
)
