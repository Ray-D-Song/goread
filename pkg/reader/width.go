package reader

// increaseWidth increases the width
func (r *Reader) increaseWidth() {
	r.UI.SetWidth(r.UI.Width + 5)
}

// decreaseWidth decreases the width
func (r *Reader) decreaseWidth() {
	r.UI.SetWidth(r.UI.Width - 5)
}
