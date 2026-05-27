package bubblecomplete

import "charm.land/bubbles/v2/key"

// KeyMap is the set of bindings the component responds to in Update.
// Override via Model.SetKeyMap to remap keys to host conventions.
type KeyMap struct {
	NextCompletion   key.Binding
	PrevCompletion   key.Binding
	AcceptCompletion key.Binding
	Submit           key.Binding
	HistoryPrev      key.Binding
	HistoryNext      key.Binding
}

// DefaultKeyMap returns the bindings applied by New.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		NextCompletion:   key.NewBinding(key.WithKeys("tab", "ctrl+n")),
		PrevCompletion:   key.NewBinding(key.WithKeys("shift+tab", "ctrl+p")),
		AcceptCompletion: key.NewBinding(key.WithKeys("right", "ctrl+e")),
		Submit:           key.NewBinding(key.WithKeys("enter")),
		HistoryPrev:      key.NewBinding(key.WithKeys("up")),
		HistoryNext:      key.NewBinding(key.WithKeys("down")),
	}
}
