package render

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Renderer holds parsed templates and rendering configuration.
type Renderer struct {
	templates   map[string]*template.Template
	devMode     bool
	templateDir string
	cssPath     string
}

// New parses all templates from templateDir and returns a ready Renderer.
func New(templateDir string, devMode bool) *Renderer {
	r := &Renderer{
		templates:   make(map[string]*template.Template),
		devMode:     devMode,
		templateDir: templateDir,
	}
	r.parseTemplates()
	return r
}

// SetCSSPath sets the path returned by the cssFile template function.
func (r *Renderer) SetCSSPath(path string) {
	r.cssPath = path
}

func (r *Renderer) funcMap() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("15:04")
		},
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04")
		},
		"cssFile": func() string {
			return r.cssPath
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}
}

func (r *Renderer) parseTemplates() {
	r.templates = make(map[string]*template.Template)

	layoutFiles, _ := filepath.Glob(filepath.Join(r.templateDir, "layouts", "*.html"))
	componentFiles, _ := filepath.Glob(filepath.Join(r.templateDir, "components", "*.html"))

	baseFiles := append(layoutFiles, componentFiles...)

	pageFiles, _ := filepath.Glob(filepath.Join(r.templateDir, "pages", "*.html"))
	for _, page := range pageFiles {
		name := strings.TrimSuffix(filepath.Base(page), filepath.Ext(page))

		// Clone base template set by parsing layouts+components first, then the page.
		base := template.New("").Funcs(r.funcMap())
		if len(baseFiles) > 0 {
			base = template.Must(base.ParseFiles(baseFiles...))
		}
		tmpl := template.Must(base.ParseFiles(page))

		r.templates[name] = tmpl
	}
}

// Page renders a full page or just the content block for HTMX requests.
func (r *Renderer) Page(w http.ResponseWriter, req *http.Request, name string, data any) {
	if r.devMode {
		r.parseTemplates()
	}

	tmpl, ok := r.templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	blockName := "layout"
	if req.Header.Get("HX-Request") == "true" {
		blockName = "content"
	}

	if err := tmpl.ExecuteTemplate(w, blockName, data); err != nil {
		fmt.Fprintf(os.Stderr, "render %s/%s: %v\n", name, blockName, err)
	}
}

// Component renders a named template block.
func (r *Renderer) Component(w http.ResponseWriter, name string, data any) {
	if r.devMode {
		r.parseTemplates()
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Search all page templates for the named block.
	for _, tmpl := range r.templates {
		t := tmpl.Lookup(name)
		if t != nil {
			if err := t.Execute(w, data); err != nil {
				fmt.Fprintf(os.Stderr, "render component %s: %v\n", name, err)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "component %q not found in any template\n", name)
}
