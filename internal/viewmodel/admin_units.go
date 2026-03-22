package viewmodel

// AdminUnitListItem holds data for one row in the admin unit list.
type AdminUnitListItem struct {
	ID            int64
	Name          string
	Slug          string
	Description   string
	AdminGroup    string
	ContactEmail  string
	GroupBindings []string
}

// AdminUnitsPage is the page-level struct for the unit list page.
type AdminUnitsPage struct {
	Layout LayoutData
	Units  []AdminUnitListItem
}

// AdminUnitFormData holds the form field values for both create and edit.
type AdminUnitFormData struct {
	ID            int64
	Name          string
	Slug          string
	Description   string
	ContactEmail  string
	AdminGroup    string
	GroupBindings []string
}

// AdminUnitFormPage is the page-level struct for the create/edit form.
type AdminUnitFormPage struct {
	Layout LayoutData
	IsNew  bool
	Unit   AdminUnitFormData
	Errors map[string]string
}
