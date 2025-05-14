package template

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
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

// RenderTemplate renders a template with the given context
func RenderTemplate(tmpl string, ctx *Context) (string, error) {
	renderer := NewRenderer()
	return renderer.Render(tmpl, ctx)
}

// PrepareTemplateContext prepares the template context with request and config data
func PrepareTemplateContext(args map[string]any, request *http.Request, serverCfg map[string]string) (*Context, error) {
	tmplCtx := NewContext()
	tmplCtx.Args = preprocessArgs(args)

	// Process request headers
	for k, v := range request.Header {
		if len(v) > 0 {
			tmplCtx.Request.Headers[k] = v[0]
		}
	}

	// Process request querystring
	for k, v := range request.URL.Query() {
		if len(v) > 0 {
			tmplCtx.Request.Query[k] = v[0]
		}
	}

	// Process request cookies
	for _, cookie := range request.Cookies() {
		if cookie.Name != "" {
			tmplCtx.Request.Cookies[cookie.Name] = cookie.Value
		}
	}

	// Only process server config templates if serverCfg is provided
	if serverCfg != nil {
		// Process server config templates
		for k, v := range serverCfg {
			rendered, err := RenderTemplate(v, tmplCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to render config template: %w", err)
			}
			serverCfg[k] = rendered
		}
		tmplCtx.Config = serverCfg
	}

	return tmplCtx, nil
}

func preprocessArgs(args map[string]any) map[string]any {
	processed := make(map[string]any)

	for k, v := range args {
		switch val := v.(type) {
		case []any:
			ss, _ := json.Marshal(val)
			processed[k] = string(ss)
		case float64:
			// If the float64 equals its integer conversion, it's an integer
			if val == float64(int64(val)) {
				processed[k] = int64(val)
			} else {
				processed[k] = val
			}
		default:
			processed[k] = v
		}
	}
	return processed
}
