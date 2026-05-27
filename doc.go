// Package bubblecomplete provides a command suggestion and autocompletion
// component for Bubble Tea applications.
//
// A [Model] holds a tree of [Command] values (each with optional
// [Command.SubCommands], [Command.PositionalArguments], and [Command.Flags])
// and renders completion suggestions and validation feedback as the user
// types. Construct one with [New] and embed it in a host Bubble Tea model:
//
//	bc, err := bubblecomplete.New(commands, 100,
//	    bubblecomplete.WithHistoryLimit(50),
//	    bubblecomplete.WithIcons(true),
//	)
//
// The host forwards Bubble Tea messages to the component's [Model.Update] and
// either embeds [Model.View] directly or composes [Model.Render] into a larger
// tea.View.
//
// Styling, key bindings, and most layout settings can be configured via either
// the public fields on [Model] or the matching With* construction options. See
// [Styles] and [KeyMap] for the structured types.
//
// The component validates input continuously; the current error is available
// via [Model.ValidationError] as a [*ValidationError] that exposes a
// [ValidationErrorKind] for host-side routing without string matching.
package bubblecomplete
