package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

type styles struct {
	headerBox       lipgloss.Style
	paneActive      lipgloss.Style
	paneInactive    lipgloss.Style
	repoName        lipgloss.Style
	branchBadge     lipgloss.Style
	detachedBadge   lipgloss.Style
	unbornBadge     lipgloss.Style
	aheadBehind     lipgloss.Style
	statusClean     lipgloss.Style
	statusDirty     lipgloss.Style
	statusConflict  lipgloss.Style
	tabActive       lipgloss.Style
	tabInactive     lipgloss.Style
	itemNormal      lipgloss.Style
	itemSelected    lipgloss.Style
	itemMuted       lipgloss.Style
	badgeCurrent    lipgloss.Style
	badgeRemote     lipgloss.Style
	badgeConflict   lipgloss.Style
	badgeStaged     lipgloss.Style
	badgeUnstaged   lipgloss.Style
	badgeUntracked  lipgloss.Style
	commitHash      lipgloss.Style
	commitSubject   lipgloss.Style
	commitMeta      lipgloss.Style
	diffAdd         lipgloss.Style
	diffDel         lipgloss.Style
	diffHunk        lipgloss.Style
	diffMeta        lipgloss.Style
	conflictOurs    lipgloss.Style
	conflictSep     lipgloss.Style
	conflictTheirs  lipgloss.Style
	conflictBar     lipgloss.Style
	bannerInfo      lipgloss.Style
	bannerError     lipgloss.Style
	footer          lipgloss.Style
	modalBox        lipgloss.Style
	modalTitle      lipgloss.Style
	modalWarn       lipgloss.Style
	modalOption     lipgloss.Style
	modalOptionSel  lipgloss.Style
	emptyState      lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		headerBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(charmtone.Charple).
			Padding(0, 1),
		paneActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(charmtone.Charple).
			Padding(0, 1),
		paneInactive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(charmtone.Iron).
			Padding(0, 1),
		repoName: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Salt),
		branchBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Salt).
			Background(charmtone.Charple).
			Padding(0, 1),
		detachedBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Pepper).
			Background(charmtone.Butter).
			Padding(0, 1),
		unbornBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Pepper).
			Background(charmtone.Malibu).
			Padding(0, 1),
		aheadBehind: lipgloss.NewStyle().
			Foreground(charmtone.Malibu).
			Bold(true),
		statusClean: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Pepper).
			Background(charmtone.Guac).
			Padding(0, 1),
		statusDirty: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Pepper).
			Background(charmtone.Mustard).
			Padding(0, 1),
		statusConflict: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Salt).
			Background(charmtone.Coral).
			Padding(0, 1),
		tabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Salt).
			Background(charmtone.Charple).
			Padding(0, 1),
		tabInactive: lipgloss.NewStyle().
			Foreground(charmtone.Oyster).
			Padding(0, 1),
		itemNormal: lipgloss.NewStyle().
			Foreground(charmtone.Soda),
		itemSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Salt).
			Background(charmtone.Char),
		itemMuted: lipgloss.NewStyle().
			Foreground(charmtone.Oyster),
		badgeCurrent: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Guac),
		badgeRemote: lipgloss.NewStyle().
			Foreground(charmtone.Malibu),
		badgeConflict: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Coral),
		badgeStaged: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Guac),
		badgeUnstaged: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Mustard),
		badgeUntracked: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Malibu),
		commitHash: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Butter),
		commitSubject: lipgloss.NewStyle().
			Foreground(charmtone.Salt),
		commitMeta: lipgloss.NewStyle().
			Foreground(charmtone.Oyster),
		diffAdd: lipgloss.NewStyle().
			Foreground(charmtone.Julep),
		diffDel: lipgloss.NewStyle().
			Foreground(charmtone.Coral),
		diffHunk: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Malibu),
		diffMeta: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Oyster),
		conflictOurs: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Guac),
		conflictSep: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Mustard),
		conflictTheirs: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Malibu),
		conflictBar: lipgloss.NewStyle().
			Foreground(charmtone.Salt).
			Background(charmtone.Char).
			Padding(0, 1),
		bannerInfo: lipgloss.NewStyle().
			Foreground(charmtone.Guac),
		bannerError: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Coral),
		footer: lipgloss.NewStyle().
			Foreground(charmtone.Oyster),
		modalBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(charmtone.Charple).
			Padding(1, 2),
		modalTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Salt),
		modalWarn: lipgloss.NewStyle().
			Foreground(charmtone.Mustard),
		modalOption: lipgloss.NewStyle().
			Foreground(charmtone.Soda).
			Padding(0, 1),
		modalOptionSel: lipgloss.NewStyle().
			Bold(true).
			Foreground(charmtone.Salt).
			Background(charmtone.Charple).
			Padding(0, 1),
		emptyState: lipgloss.NewStyle().
			Foreground(charmtone.Oyster).
			Italic(true),
	}
}
