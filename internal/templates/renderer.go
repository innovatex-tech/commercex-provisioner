package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/innovatex-tech/commercex-provisioner/templates"
)

type Renderer struct {
	templateDir string
}

func NewRenderer(dir string) *Renderer {
	return &Renderer{templateDir: dir}
}

func (r *Renderer) Render(templateName string, data interface{}, outputPath string) error {
	var tmpl *template.Template
	var err error

	// 1. Try to load from external directory first (allows user customization)
	tmplPath := filepath.Join(r.templateDir, templateName)
	if _, err := os.Stat(tmplPath); err == nil {
		tmpl, err = template.ParseFiles(tmplPath)
	} else {
		// 2. Fallback to embedded templates in the binary
		tmpl, err = template.ParseFS(templates.FS, templateName)
		if err != nil {
			return fmt.Errorf("template %s not found in %s or binary: %v", templateName, r.templateDir, err)
		}
	}

	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}
