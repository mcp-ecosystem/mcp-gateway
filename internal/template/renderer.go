package template

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"text/template"
)

// Renderer is responsible for rendering templates
type Renderer struct {
	templates map[string]*template.Template
}

// NewRenderer creates a new template renderer
func NewRenderer() *Renderer {
	return &Renderer{
		templates: make(map[string]*template.Template),
	}
}

// generateTemplateName generates a unique name for a template based on its content
func generateTemplateName(tmpl string) string {
	hash := sha256.Sum256([]byte(tmpl))
	return fmt.Sprintf("tmpl_%s", hex.EncodeToString(hash[:8]))
}

// Render renders a template with the given context
func (r *Renderer) Render(tmpl string, ctx *Context) (string, error) {
	name := generateTemplateName(tmpl)
	t, ok := r.templates[name]
	if !ok {
		var err error
		t, err = template.New(name).Funcs(template.FuncMap{
			"env":      ctx.Env,
			"add":      func(a, b int) int { return a + b },
			"fromJSON": fromJSON,
		}).Parse(tmpl)
		if err != nil {
			return "", err
		}
		r.templates[name] = t
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return "", err
	}

	return buf.String(), nil
}
