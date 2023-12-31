package replay

import (
	"fmt"
	"time"

	"github.com/cfoust/cy/pkg/emu"
	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/geom/image"
	"github.com/cfoust/cy/pkg/geom/tty"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// For a point that is off the screen, find the closest point that can be used
// as the start or end point of a selection.
func anchorToScreen(size geom.Vec2, v geom.Vec2) geom.Vec2 {
	if v.R < 0 {
		return geom.Vec2{}
	}

	if v.R >= size.R {
		return geom.Vec2{
			R: size.R - 1,
			C: size.C - 1,
		}
	}

	if v.C < 0 {
		return geom.Vec2{
			R: v.R,
			C: 0,
		}
	}

	if v.C > 0 {
		return geom.Vec2{
			R: v.R,
			C: size.C - 1,
		}
	}

	return v
}

func (r *Replay) highlightRange(state *tty.State, from, to geom.Vec2, fg, bg emu.Color) {
	from, to = normalizeRange(from, to)
	from = r.termToViewport(from)
	to = r.termToViewport(to)

	size := state.Image.Size()
	if !r.isInViewport(from) {
		from = anchorToScreen(size, from)
	}
	if !r.isInViewport(to) {
		to = anchorToScreen(size, to)
	}

	var startCol, endCol int
	for row := from.R; row <= to.R; row++ {
		startCol = 0
		if row == from.R {
			startCol = from.C
		}

		endCol = size.C - 1
		if row == to.R {
			endCol = to.C
		}

		for col := startCol; col <= endCol; col++ {
			state.Image[row][col].FG = fg
			state.Image[row][col].BG = bg
		}
	}
}

func (r *Replay) drawMatches(state *tty.State) {
	matches := r.matches
	if len(matches) == 0 {
		return
	}

	fgColor := r.render.ConvertLipgloss(lipgloss.Color("1"))
	bgColor := r.render.ConvertLipgloss(lipgloss.Color("14"))
	bgSelectedColor := r.render.ConvertLipgloss(lipgloss.Color("13"))

	location := r.location
	for _, match := range matches {
		// This match is not on the screen
		if location.Before(match.Begin) || location.After(match.End) {
			continue
		}

		bg := bgColor
		if location.Equal(match.Begin) {
			bg = bgSelectedColor
		}

		for _, appearance := range match.Appearances {
			if location.After(appearance.End) {
				continue
			}
			r.highlightRange(
				state,
				appearance.From,
				appearance.To,
				fgColor,
				bg,
			)
			break
		}
	}
}

func (r *Replay) drawStatusBar(state *tty.State) {
	size := state.Image.Size()

	statusBarStyle := r.render.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("8"))

	statusText := "⏵"
	statusBG := lipgloss.Color("#4D9DE0")
	if r.isCopyMode() {
		statusText = "COPY"
		statusBG = lipgloss.Color("#E1BC29")

		if r.isSelecting {
			statusText = "VISUAL"
			statusBG = lipgloss.Color("#3BB273")
		}
	}
	if r.isPlaying {
		statusText = "⏸"
		statusBG = lipgloss.Color("#7768AE")
	}

	if !r.isCopyMode() && r.playbackRate != 1 {
		statusText += fmt.Sprintf(" %dx", r.playbackRate)
	}

	statusStyle := r.render.NewStyle().
		Inherit(statusBarStyle).
		Background(statusBG).
		Padding(0, 1)

	index := r.location.Index
	if index < 0 || index >= len(r.events) || len(r.events) == 0 {
		return
	}

	status := statusStyle.Render(statusText)

	leftSide := lipgloss.JoinHorizontal(lipgloss.Top,
		status,
		statusBarStyle.
			Copy().
			Padding(0, 1).
			Render(
				r.currentTime.Format(
					time.RFC3339,
				),
			),
	)

	progressWidth := size.C - lipgloss.Width(leftSide) - 3
	percent := int((float64(r.location.Index) / float64(len(r.events))) * float64(progressWidth))
	progressBar := ""
	for i := 0; i < progressWidth; i++ {
		if i <= percent {
			progressBar += "▒"
		} else {
			progressBar += "-"
		}
	}

	progressBar = "[" + progressBar + "]"
	progressBar = statusBarStyle.
		Copy().
		Render(progressBar)

	statusBar := statusBarStyle.
		Width(size.C).
		Height(1).
		Render(lipgloss.JoinHorizontal(lipgloss.Left,
			leftSide,
			progressBar,
		))

	r.render.RenderAt(state.Image, size.R-1, 0, statusBar)
}

// drawScrollbackPosition renders "[1/N]" text in the top-right corner that
// looks just like tmux's copy mode.
func (r *Replay) drawScrollbackPosition(state *tty.State) {
	size := state.Image.Size()
	offsetStyle := r.render.NewStyle().
		Foreground(lipgloss.Color("9")).
		Background(lipgloss.Color("240"))

	// draw where the screen ends and scrollback begins
	linePos := r.termToViewport(geom.Vec2{R: 0}).R
	if linePos >= 0 && linePos < r.viewport.R {
		r.render.RenderAt(
			state.Image,
			linePos,
			r.viewport.C-3,
			offsetStyle.Render("<--"),
		)
	}

	r.render.RenderAt(
		state.Image,
		0,
		0,
		r.render.PlaceHorizontal(
			size.C,
			lipgloss.Right,
			offsetStyle.Render(fmt.Sprintf(
				"[%d/%d]",
				-r.offset.R,
				-r.minOffset.R,
			)),
		),
	)
}

func (r *Replay) renderInput() image.Image {
	r.searchInput.Cursor.Style = r.render.NewStyle().
		Background(lipgloss.Color("15"))

	width := 20
	common := r.render.NewStyle().Width(width)
	inputStyle := common.Copy().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("8"))

	promptStyle := common.Copy().
		Foreground(lipgloss.Color("8")).
		Background(lipgloss.Color("15"))

	prompt := "search-forward"
	if !r.isForward {
		prompt = "search-backward"
	}

	value := r.searchInput.Value()
	if match := TIME_DELTA_REGEX.FindStringSubmatch(value); len(value) > 0 && match != nil {
		promptStyle = common.Copy().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("#7768AE"))

		prompt = "jump-forward"
		if !r.isForward {
			prompt = "jump-backward"
		}
	}

	input := inputStyle.Render(r.searchInput.View())

	if r.isWaiting {
		percent := r.progressPercent

		spin := spinner.Dot
		first := spin.Frames[percent%len(spin.Frames)]
		left := "searching..."
		prompt = left + lipgloss.PlaceHorizontal(
			width-len(left),
			lipgloss.Right,
			first,
		)

		progressStyle := inputStyle.Copy().
			Background(lipgloss.Color("#4D9DE0"))

		filled := int((float64(percent) / 100) * float64(width))

		input = progressStyle.Width(filled).Render("") + inputStyle.Width(width-filled).Render("")
	} else if r.isEmpty {
		prompt = "no matches found"
	}

	prompt = promptStyle.Render(prompt)

	input = lipgloss.JoinVertical(
		lipgloss.Left,
		input,
		prompt,
	)
	return r.render.RenderImage(input)
}

func (r *Replay) View(state *tty.State) {
	screen := r.terminal.Screen()
	history := r.terminal.History()
	state.CursorVisible = true

	// Return nothing when View() is called before we've actually gotten
	// the viewport
	if r.viewport.R == 0 && r.viewport.C == 0 {
		return
	}

	// Draw the underlying terminal state
	//////////////////////////////////////////////
	termSize := r.getTerminalSize()
	var point geom.Vec2
	var glyph emu.Glyph
	for row := 0; row <= r.viewport.R; row++ {
		point.R = row + r.offset.R
		for col := 0; col < r.viewport.C; col++ {
			point.C = r.offset.C + col

			if point.C >= termSize.C || point.R >= termSize.R {
				glyph = emu.EmptyGlyph()
				glyph.FG = 8
				glyph.Char = '-'
			} else if point.R < 0 {
				glyph = history[len(history)+point.R][point.C]
			} else {
				glyph = screen[point.R][point.C]
			}

			state.Image[row][col] = glyph
		}
	}

	termCursor := r.termToViewport(r.getTerminalCursor())
	if r.isCopyMode() {
		state.Cursor.X = r.cursor.C
		state.Cursor.Y = r.cursor.R

		// In copy mode, leave behind a ghost cursor where the
		// terminal's cursor is
		if r.isInViewport(termCursor) {
			state.Image[termCursor.R][termCursor.C].BG = 8
		}
	} else {
		state.Cursor = r.terminal.Cursor()
		state.Cursor.X = termCursor.C
		state.Cursor.Y = termCursor.R
		if r.isPlaying {
			state.CursorVisible = r.terminal.CursorVisible()
		}
	}

	// Show the selection state
	////////////////////////////
	if r.isCopyMode() && r.isSelecting {
		r.highlightRange(
			state,
			r.selectStart,
			r.viewportToTerm(r.cursor),
			r.render.ConvertLipgloss(lipgloss.Color("9")),
			r.render.ConvertLipgloss(lipgloss.Color("240")),
		)
	}

	// Highlight any matches on the screen
	///////////////////////////////////////////////
	r.drawMatches(state)

	// Render overlays
	///////////////////////////
	r.drawStatusBar(state)

	if r.offset.R < 0 {
		r.drawScrollbackPosition(state)
	}

	// Render text input
	/////////////////////////////
	if r.mode != ModeInput && !r.isWaiting && !r.isEmpty {
		return
	}

	// hide the cursor when typing in the search bar (it has its own)
	state.CursorVisible = false

	size := state.Image.Size()
	input := r.renderInput()
	inputSize := input.Size()
	image.Copy(
		geom.Vec2{
			// -1 for the status bar
			R: geom.Clamp(r.cursor.R, 0, size.R-inputSize.R-1),
			C: geom.Clamp(r.cursor.C, 0, size.C-inputSize.C),
		},
		state.Image,
		input,
	)
}
