package template

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
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
			"toJSON":   toJSON,
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

func AssembleTemplateContext(req *RequestWrapper, args map[string]any, serverCfg map[string]string) (*Context, error) {
	tmplCtx := NewContext()
	tmplCtx.Args = preprocessArgs(args)

	if req != nil {
		tmplCtx.Request = *req
	}

	if serverCfg != nil {
		renderedCfg, err := renderServerConfigTemplates(serverCfg, tmplCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render config template: %w", err)
		}
		tmplCtx.Config = renderedCfg
	}

	return tmplCtx, nil
}

// PrepareTemplateContext prepares the template context with request and config data
func PrepareTemplateContext(requestMeta *session.RequestInfo, args map[string]any, request *http.Request, serverCfg map[string]string) (*Context, error) {
	tmplCtx := NewContext()
	tmplCtx.Args = preprocessArgs(args)

	mergeHeaders(tmplCtx, requestMeta, request)
	mergeQuery(tmplCtx, requestMeta, request)
	mergeCookies(tmplCtx, requestMeta, request)

	if serverCfg != nil {
		renderedCfg, err := renderServerConfigTemplates(serverCfg, tmplCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render config template: %w", err)
		}
		tmplCtx.Config = renderedCfg
	}

	return tmplCtx, nil
}

func mergeHeaders(ctx *Context, meta *session.RequestInfo, req *http.Request) {
	if meta != nil {
		for k, v := range meta.Headers {
			ctx.Request.Headers[k] = v
		}
	}
	if req != nil {
		for k, v := range req.Header {
			if len(v) > 0 {
				ctx.Request.Headers[k] = v[0]
			}
		}
	}
}

func mergeQuery(ctx *Context, meta *session.RequestInfo, req *http.Request) {
	if meta != nil {
		for k, v := range meta.Query {
			ctx.Request.Query[k] = v
		}
	}
	if req != nil {
		for k, v := range req.URL.Query() {
			if len(v) > 0 {
				ctx.Request.Query[k] = v[0]
			}
		}
	}
}

func mergeCookies(ctx *Context, meta *session.RequestInfo, req *http.Request) {
	if meta != nil {
		for k, v := range meta.Cookies {
			ctx.Request.Cookies[k] = v
		}
	}
	if req != nil {
		for _, cookie := range req.Cookies() {
			if cookie.Name != "" {
				ctx.Request.Cookies[cookie.Name] = cookie.Value
			}
		}
	}
}

func renderServerConfigTemplates(cfg map[string]string, ctx *Context) (map[string]string, error) {
	rendered := make(map[string]string, len(cfg))
	for k, v := range cfg {
		out, err := RenderTemplate(v, ctx)
		if err != nil {
			return nil, err
		}
		rendered[k] = out
	}
	return rendered, nil
}

func preprocessArgs(args map[string]any) map[string]any {
	processed := make(map[string]any)

	if args == nil {
		return processed
	}

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
