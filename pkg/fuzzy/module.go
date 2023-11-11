package fuzzy

import (
	"context"

	"github.com/cfoust/cy/pkg/anim"
	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/geom/image"
	"github.com/cfoust/cy/pkg/mux/screen/replay"
	"github.com/cfoust/cy/pkg/mux/screen/server"
	"github.com/cfoust/cy/pkg/mux/screen/tree"
	"github.com/cfoust/cy/pkg/taro"
	"github.com/cfoust/cy/pkg/util"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Fuzzy struct {
	util.Lifetime
	anim *anim.Animator

	result   chan<- interface{}
	location geom.Vec2
	size     geom.Vec2

	render    *taro.Renderer
	textInput textinput.Model

	// Don't allow Fuzzy to quit or the user to choose anything
	isSticky bool

	// before the user has done anything, we don't show the preview window
	haveMoved bool

	options  []Option
	filtered []Option
	selected int
	pattern  string

	tree       *tree.Tree
	client     *server.Client
	isAttached bool
	replay     *taro.Program
}

var _ taro.Model = (*Fuzzy)(nil)

func (f *Fuzzy) quit() (taro.Model, tea.Cmd) {
	return f, tea.Quit
}

type AttachEvent struct {
	pane *tree.Pane
}

type DetachEvent struct {
}

func (f *Fuzzy) Attach(id tree.NodeID) taro.Cmd {
	f.isAttached = false

	return func() tea.Msg {
		pane, ok := f.tree.PaneById(id)
		if !ok {
			return nil
		}

		f.client.Attach(f.Ctx(), pane.Screen())
		return AttachEvent{
			pane: pane,
		}
	}
}

func (f *Fuzzy) handlePreview() taro.Cmd {
	options := f.getOptions()
	if len(options) == 0 {
		return nil
	}

	option := options[f.selected]
	if option.Preview == nil {
		return nil
	}

	switch preview := option.Preview.(type) {
	case nodePreview:
		return f.Attach(preview.Id)
	case replayPreview:
		if f.replay != nil {
			f.replay.Cancel()
		}
		f.replay = replay.NewPreview(
			f.Ctx(),
			preview.Path,
		)
		f.replay.Resize(f.size)
	}

	return nil
}

func (f *Fuzzy) Init() taro.Cmd {
	cmds := []taro.Cmd{
		textinput.Blink,
	}

	if f.anim != nil {
		cmds = append(cmds, taro.WaitScreens(f.Ctx(), f.anim))
	}

	return tea.Batch(cmds...)
}

func (f *Fuzzy) getOptions() []Option {
	if len(f.pattern) > 0 {
		return f.filtered
	}
	return f.options
}

func (f *Fuzzy) isInverted() bool {
	return f.location.R > (f.size.R / 2)
}

type SelectedEvent struct {
	Option Option
}

func (f *Fuzzy) setSelected(index int) {
	f.selected = geom.Clamp(index, 0, len(f.getOptions())-1)
}

func (f *Fuzzy) emitOption() taro.Cmd {
	return func() taro.Msg {
		return taro.PublishMsg{
			Msg: SelectedEvent{
				Option: f.getOptions()[f.selected],
			},
		}
	}
}

func (f *Fuzzy) Update(msg tea.Msg) (taro.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case AttachEvent:
		f.isAttached = true
		return f, func() tea.Msg {
			select {
			case <-f.client.Attachment().Ctx().Done():
				return nil
			case <-msg.pane.Ctx().Done():
				return DetachEvent{}
			}
		}
	case DetachEvent:
		f.isAttached = false
		return f, nil
	case taro.ScreenUpdate:
		return f, taro.WaitScreens(f.Ctx(), f.anim)
	case matchResult:
		f.filtered = msg.Filtered
		f.setSelected(f.selected)
		return f, f.emitOption()
	case tea.WindowSizeMsg:
		size := geom.Size{
			R: msg.Height,
			C: msg.Width,
		}
		f.anim.Resize(size)
		f.size = size
	case taro.KeyMsg:
		switch msg.Type {
		case taro.KeyEsc, taro.KeyCtrlC:
			f.result <- nil
			return f.quit()
		case taro.KeyDown, taro.KeyCtrlJ, taro.KeyUp, taro.KeyCtrlK:
			f.haveMoved = true
			isUp := false
			switch msg.Type {
			case taro.KeyUp, taro.KeyCtrlK:
				isUp = true
			}
			if f.isInverted() {
				isUp = !isUp
			}

			delta := -1
			if !isUp {
				delta = 1
			}

			f.setSelected(f.selected + delta)
			return f, tea.Batch(
				f.handlePreview(),
				f.emitOption(),
			)
		case taro.KeyEnter:
			if f.isSticky {
				return f, nil
			}

			if f.selected >= 0 && f.selected < len(f.getOptions()) {
				option := f.getOptions()[f.selected]
				f.result <- option.Result
			} else {
				f.result <- nil
			}
			return f.quit()
		}
	}

	inputMsg := msg
	// We need to translate taro.KeyMsg to tea.KeyMsg (for now)
	if key, ok := msg.(taro.KeyMsg); ok {
		inputMsg = key.ToTea()
	}
	f.textInput, cmd = f.textInput.Update(inputMsg)
	cmds = append(cmds, cmd)

	value := f.textInput.Value()
	if f.pattern != value {
		f.pattern = value
		cmds = append(cmds, queryOptions(f.options, value))
	}

	return f, tea.Batch(cmds...)
}

type Setting func(context.Context, *Fuzzy)

func WithAnimation(image image.Image) Setting {
	return func(ctx context.Context, f *Fuzzy) {
		f.anim = anim.NewAnimator(
			ctx,
			anim.Random(),
			image,
			23,
		)
	}
}

// Don't allow Fuzzy to quit.
func WithSticky(ctx context.Context, f *Fuzzy) {
	f.isSticky = true
}

// If both of these are provided, Fuzzy can show previews for panes.
func WithNodes(t *tree.Tree, client *server.Client) Setting {
	return func(ctx context.Context, f *Fuzzy) {
		f.tree = t
		f.client = client
	}
}

func WithResult(result chan<- interface{}) Setting {
	return func(ctx context.Context, f *Fuzzy) {
		f.result = result
	}
}

func NewFuzzy(
	ctx context.Context,
	options []Option,
	location geom.Vec2,
	settings ...Setting,
) *taro.Program {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20
	ti.Prompt = ""

	f := &Fuzzy{
		Lifetime:  util.NewLifetime(ctx),
		render:    taro.NewRenderer(),
		location:  location,
		options:   options,
		selected:  0,
		textInput: ti,
	}

	for _, setting := range settings {
		setting(f.Ctx(), f)
	}

	return taro.New(ctx, f)
}

type matchResult struct {
	Filtered []Option
}

func queryOptions(options []Option, pattern string) tea.Cmd {
	return func() tea.Msg {
		return matchResult{
			Filtered: Filter(options, pattern),
		}
	}
}
