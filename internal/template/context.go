package template

import "os"

// Context represents the template context
type (
	Context struct {
		Args     map[string]any      `json:"args"`
		Config   map[string]string   `json:"config"`
		Request  RequestWrapper      `json:"request"`
		Response ResponseWrapper     `json:"response"`
		Env      func(string) string `json:"-"` // Function to get environment variables
	}
	RequestWrapper struct {
		Headers map[string]string `json:"headers"`
		Query   map[string]string `json:"query"`
		Path    map[string]string `json:"path"`
		Body    map[string]any    `json:"body"`
	}
	ResponseWrapper struct {
		Data any `json:"data"`
		Body any `json:"body"`
	}
)

// NewContext creates a new template context
func NewContext() *Context {
	return &Context{
		Args:   make(map[string]any),
		Config: make(map[string]string),
		Request: RequestWrapper{
			Headers: make(map[string]string),
			Query:   make(map[string]string),
			Path:    make(map[string]string),
			Body:    make(map[string]any),
		},
		Response: ResponseWrapper{},
		Env:      os.Getenv,
	}
}
