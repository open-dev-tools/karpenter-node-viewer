package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up, Down, Tab, Enter, Next, Prev, Sort, Reverse, All, Refresh, Help, Esc, Quit, Mouse key.Binding
}

// ShortHelp is the one-line footer; FullHelp is shown with "?".
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Up, k.Down, k.Tab, k.Enter, k.Sort, k.All, k.Mouse, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Tab, k.Enter}, {k.Next, k.Prev, k.Sort, k.Reverse}, {k.All, k.Refresh, k.Esc, k.Quit}, {k.Mouse}}
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "pools⇄nodes")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "pool→nodes · node→pods")),
	Next:    key.NewBinding(key.WithKeys("right", "l", "c"), key.WithHelp("→", "next ctx")),
	Prev:    key.NewBinding(key.WithKeys("left", "h", "C"), key.WithHelp("←", "prev ctx")),
	Sort:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
	Reverse: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "reverse sort")),
	All:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all pools")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh data now")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
	Esc:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close · back to all pools")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	// help-only entry for the mouse; no key matches it
	Mouse: key.NewBinding(key.WithKeys("\x00mouse"), key.WithHelp("click", "tab·row·header")),
}

type bindings []key.Binding

func (b bindings) ShortHelp() []key.Binding  { return b }
func (b bindings) FullHelp() [][]key.Binding { return [][]key.Binding{b} }

var detailKeys = bindings{keys.Up, keys.Down, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc/q/click", "close"))}
