package output

type UI struct {
	interactive bool
}

func NewUI() *UI {
	return &UI{
		interactive: IsInteractive(),
	}
}

func (u *UI) IsInteractive() bool {
	return u.interactive
}
