package viewmodel

type LayoutData struct {
	Title     string
	CSRFToken string
	UserEmail string
	IsAdmin   bool
}

type Toast struct {
	Type    string // "success", "error", "warning", "info"
	Message string
}
