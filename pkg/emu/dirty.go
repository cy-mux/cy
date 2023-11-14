package emu

func (t *State) markDirtyLine(row int) {
	index := clamp(row, 0, t.rows-1)

	if _, ok := t.dirty[index]; ok {
		return
	}

	t.dirty[index] = true
}

func (t *State) LastCell() (cell Cell) {
	cell = t.lastCell
	return
}

// resetChanges resets the change mask and dirtiness.
func (t *State) resetChanges() {
	t.dirty = make(map[int]bool)
	t.changed = 0
}

func (t *State) ScreenChanged() bool {
	return t.changed&ChangedScreen != 0
}
