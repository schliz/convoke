package viewmodel

// NavUnit is a lightweight projection of a unit for nav rendering.
type NavUnit struct {
	Name string
	Slug string
}

// LayoutData is the view model for the base layout and nav component.
type LayoutData struct {
	Title     string
	CSRFToken string
	UserEmail string
	IsAdmin   bool
	Units     []NavUnit // units the user belongs to (via IdP groups)
	HasUnits  bool      // precomputed: len(Units) > 0
}

type Toast struct {
	Type    string // "success", "error", "warning", "info"
	Message string
}
