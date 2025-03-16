package reader

import (
	"github.com/gdamore/tcell/v2"
)

func (r *Reader) ResetCapture() {
	r.UI.App.SetInputCapture(InitialCapture)
}

func (r *Reader) SetCapture(f func(event *tcell.EventKey) *tcell.EventKey) {
	r.UI.App.SetInputCapture(f)
}
