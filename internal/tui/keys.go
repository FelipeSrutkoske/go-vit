package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Top           key.Binding
	Bottom        key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	NextTab       key.Binding
	Tab1          key.Binding
	Tab2          key.Binding
	ToggleRemotes key.Binding
	Switch        key.Binding
	NewBranch     key.Binding
	DeleteBranch  key.Binding
	MergeBranch   key.Binding
	Stage         key.Binding
	Commit        key.Binding
	Stash         key.Binding
	Discard       key.Binding
	KeepOurs      key.Binding
	AcceptTheirs  key.Binding
	OpenEditor    key.Binding
	Fetch         key.Binding
	Pull          key.Binding
	Refresh       key.Binding
	Help          key.Binding
	Quit          key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("k/cima", "subir"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("j/baixo", "descer"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "início"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "fim"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u", "K"),
			key.WithHelp("K/PgUp", "rolar painel dir. acima"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d", "J"),
			key.WithHelp("J/PgDn", "rolar painel dir. abaixo"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "trocar aba"),
		),
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "aba Branches"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "aba Alterações"),
		),
		ToggleRemotes: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "mostrar/ocultar remotas"),
		),
		Switch: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "trocar branch"),
		),
		NewBranch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "nova branch"),
		),
		DeleteBranch: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "excluir branch"),
		),
		MergeBranch: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mesclar na atual"),
		),
		Stage: key.NewBinding(
			key.WithKeys("space", "s"),
			key.WithHelp("Espaço/s", "preparar/remover"),
		),
		Commit: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "fazer commit"),
		),
		Stash: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "stash / pop"),
		),
		Discard: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "descartar alterações"),
		),
		KeepOurs: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "manter nossos (ours)"),
		),
		AcceptTheirs: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "aceitar deles (theirs)"),
		),
		OpenEditor: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "abrir no $EDITOR"),
		),
		Fetch: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fetch"),
		),
		Pull: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pull"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "atualizar"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "ajuda"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "sair"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextTab, k.Up, k.Down, k.Fetch, k.Pull, k.Refresh, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.PageUp, k.PageDown, k.NextTab},
		{k.Switch, k.ToggleRemotes, k.NewBranch, k.DeleteBranch, k.MergeBranch},
		{k.Stage, k.Commit, k.Stash, k.Discard, k.KeepOurs, k.AcceptTheirs, k.OpenEditor},
		{k.Fetch, k.Pull, k.Refresh, k.Help, k.Quit},
	}
}
