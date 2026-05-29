package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/clobrano/telekube/internal/config"
)

// KeyMap defines all keybindings for the application
type KeyMap struct {
	Quit            key.Binding
	Help            key.Binding
	Refresh         key.Binding
	Search          key.Binding
	Enter           key.Binding
	Escape          key.Binding
	Up              key.Binding
	Down            key.Binding
	TabNext         key.Binding
	TabPrev         key.Binding
	DeleteTab       key.Binding
	Describe key.Binding
	Logs     key.Binding
	Delete   key.Binding
	Edit            key.Binding
	Terminal        key.Binding
	PortForward     key.Binding
	Scale           key.Binding
	YAMLView        key.Binding
	JSONView        key.Binding
	SwitchNamespace key.Binding
	SwitchContext   key.Binding
	CopyData        key.Binding
	MultiSelect     key.Binding
}

// NewKeyMap creates a KeyMap from configuration
func NewKeyMap(cfg config.Keybindings) *KeyMap {
	return &KeyMap{
		Quit: key.NewBinding(
			key.WithKeys(cfg.Quit, "ctrl+c"),
			key.WithHelp(cfg.Quit, "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys(cfg.Help),
			key.WithHelp(cfg.Help, "help"),
		),
		Refresh: key.NewBinding(
			key.WithKeys(cfg.Refresh),
			key.WithHelp(cfg.Refresh, "refresh"),
		),
		Search: key.NewBinding(
			key.WithKeys(cfg.Search),
			key.WithHelp(cfg.Search, "search"),
		),
		Enter: key.NewBinding(
			key.WithKeys(cfg.Enter),
			key.WithHelp("enter", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys(cfg.Escape),
			key.WithHelp("esc", "back"),
		),
		Up: key.NewBinding(
			key.WithKeys(cfg.Up, "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys(cfg.Down, "j"),
			key.WithHelp("↓/j", "down"),
		),
		TabNext: key.NewBinding(
			key.WithKeys(cfg.TabNext, "right", "l"),
			key.WithHelp("→/l/tab", "next tab"),
		),
		TabPrev: key.NewBinding(
			key.WithKeys(cfg.TabPrev, "left", "h"),
			key.WithHelp("←/h/shift+tab", "prev tab"),
		),
		DeleteTab: key.NewBinding(
			key.WithKeys(cfg.DeleteTab),
			key.WithHelp(cfg.DeleteTab, "delete tab"),
		),
		Describe: key.NewBinding(
			key.WithKeys(cfg.Describe),
			key.WithHelp(cfg.Describe, "describe"),
		),
		Logs: key.NewBinding(
			key.WithKeys(cfg.Logs),
			key.WithHelp(cfg.Logs, "logs"),
		),
		Delete: key.NewBinding(
			key.WithKeys(cfg.Delete),
			key.WithHelp(cfg.Delete, "delete"),
		),
		Edit: key.NewBinding(
			key.WithKeys(cfg.Edit),
			key.WithHelp(cfg.Edit, "edit"),
		),
		Terminal: key.NewBinding(
			key.WithKeys(cfg.Terminal),
			key.WithHelp(cfg.Terminal, "terminal"),
		),
		PortForward: key.NewBinding(
			key.WithKeys(cfg.PortForward),
			key.WithHelp(cfg.PortForward, "port-forward"),
		),
		Scale: key.NewBinding(
			key.WithKeys(cfg.Scale),
			key.WithHelp(cfg.Scale, "scale"),
		),
		YAMLView: key.NewBinding(
			key.WithKeys(cfg.YAMLView),
			key.WithHelp(cfg.YAMLView, "yaml"),
		),
		JSONView: key.NewBinding(
			key.WithKeys(cfg.JSONView),
			key.WithHelp(cfg.JSONView, "json"),
		),
		SwitchNamespace: key.NewBinding(
			key.WithKeys(cfg.SwitchNamespace),
			key.WithHelp(cfg.SwitchNamespace, "namespace"),
		),
		SwitchContext: key.NewBinding(
			key.WithKeys(cfg.SwitchContext),
			key.WithHelp(cfg.SwitchContext, "context"),
		),
		CopyData: key.NewBinding(
			key.WithKeys(cfg.CopyData),
			key.WithHelp(cfg.CopyData, "copy"),
		),
		MultiSelect: key.NewBinding(
			key.WithKeys(cfg.MultiSelect),
			key.WithHelp("space", "select"),
		),
	}
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() *KeyMap {
	return NewKeyMap(config.DefaultConfig().Keybindings)
}

// ShortHelp returns keybindings to show in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings to show in the full help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Escape},
		{k.TabNext, k.TabPrev, k.Search, k.Refresh},
		{k.Describe, k.Logs, k.Delete, k.Edit},
		{k.Terminal, k.PortForward, k.Scale},
		{k.YAMLView, k.JSONView, k.SwitchNamespace, k.SwitchContext},
		{k.CopyData},
		{k.Help, k.Quit},
	}
}
