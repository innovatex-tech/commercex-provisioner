package templates

import (
	"os"
	"path/filepath"
	"text/template"
)

type Renderer struct {
	templateDir string
}

func NewRenderer(dir string) *Renderer {
	return &Renderer{templateDir: dir}
}

func (r *Renderer) Render(templateName string, data interface{}, outputPath string) error {
	tmplPath := filepath.Join(r.templateDir, templateName)

	tmpl, err := template.ParseFiles(tmplPath)
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
