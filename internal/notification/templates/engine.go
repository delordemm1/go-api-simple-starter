package templates

import (
	"bytes"
	"context"
	"fmt"
	htmltmpl "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	texttmpl "text/template"

	"log/slog"
)

// Config controls how the template engine loads templates.
// Dir: when non-empty, loads templates from this directory (expects files named <id>.tmpl).
// Reload: when true and Dir is set, templates are reparsed on every render.
type Config struct {
	Dir    string
	Reload bool
}

// Rendered holds the per-channel materialized content from a scenario template.
type Rendered struct {
	Subject   string
	EmailHTML string
	EmailText string
	SMSText   string
	PushTitle string
	PushBody  string
}

// IHandle is a runtime-typed handle to a template scenario.
// A generic Handle[T] implements this to carry compile-time data type info.
type IHandle interface {
	ID() string
	DataType() reflect.Type
}

// Handle is a generic, typed handle for a template scenario.
type Handle[T any] struct {
	id string
}

// Expect creates a typed handle for a given template ID (e.g., "user.verify_email").
func Expect[T any](id string) Handle[T] { return Handle[T]{id: id} }

func (h Handle[T]) ID() string { return h.id }
func (h Handle[T]) DataType() reflect.Type {
	var zero *T
	return reflect.TypeOf(zero).Elem()
}

// Renderer is the DI-friendly interface for the engine.
type Renderer interface {
	// RenderAny renders the given template ID with arbitrary data.
	// Prefer using the typed helper Engine.Render with a Handle[T] in internal code.
	RenderAny(ctx context.Context, id string, data any) (Rendered, error)
}

// Engine compiles and renders scenario templates with optional dev reload.
type Engine struct {
	cfg   Config
	log   *slog.Logger
	fs    fs.FS
	mu    sync.RWMutex
	cache map[string]*compiled
}

type compiled struct {
	text *texttmpl.Template
	html *htmltmpl.Template
}

// NewEngine creates a template engine. It uses embedded templates by default.
// If cfg.Dir is provided, templates are loaded from disk; if cfg.Reload is true,
// disk templates are reparsed on every render call.
func NewEngine(cfg Config, log *slog.Logger) *Engine {
	if log == nil {
		log = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	return &Engine{
		cfg:   cfg,
		log:   log,
		fs:    EmbeddedFS,
		cache: make(map[string]*compiled),
	}
}

 // Render is a typed helper that enforces the data type associated with the handle at compile time.
func Render[T any](ctx context.Context, e *Engine, h Handle[T], data T) (Rendered, error) {
	return e.RenderAny(ctx, h.ID(), data)
}

// RenderAny renders a scenario by ID using either embedded or disk templates.
func (e *Engine) RenderAny(ctx context.Context, id string, data any) (Rendered, error) {
	c, err := e.getCompiled(id)
	if err != nil {
		return Rendered{}, err
	}

	var out Rendered
	// text blocks
	if c.text.Lookup("subject") != nil {
		if s, err := execText(c.text, "subject", data); err != nil {
			return Rendered{}, fmt.Errorf("render subject: %w", err)
		} else {
			out.Subject = s
		}
	}
	if c.text.Lookup("email_text") != nil {
		if s, err := execText(c.text, "email_text", data); err != nil {
			return Rendered{}, fmt.Errorf("render email_text: %w", err)
		} else {
			out.EmailText = s
		}
	}
	if c.text.Lookup("sms_text") != nil {
		if s, err := execText(c.text, "sms_text", data); err != nil {
			return Rendered{}, fmt.Errorf("render sms_text: %w", err)
		} else {
			out.SMSText = s
		}
	}
	if c.text.Lookup("push_title") != nil {
		if s, err := execText(c.text, "push_title", data); err != nil {
			return Rendered{}, fmt.Errorf("render push_title: %w", err)
		} else {
			out.PushTitle = s
		}
	}
	if c.text.Lookup("push_body") != nil {
		if s, err := execText(c.text, "push_body", data); err != nil {
			return Rendered{}, fmt.Errorf("render push_body: %w", err)
		} else {
			out.PushBody = s
		}
	}
	// html block
	if c.html.Lookup("email_html") != nil {
		if s, err := execHTML(c.html, "email_html", data); err != nil {
			return Rendered{}, fmt.Errorf("render email_html: %w", err)
		} else {
			out.EmailHTML = s
		}
	}

	return out, nil
}

func (e *Engine) getCompiled(id string) (*compiled, error) {
	// Disk reload path: always reparse when Reload is true and Dir is set.
	if e.cfg.Dir != "" && e.cfg.Reload {
		return e.parseFromDisk(id)
	}

	// Cache path: try cache first.
	e.mu.RLock()
	cached, ok := e.cache[id]
	e.mu.RUnlock()
	if ok {
		return cached, nil
	}

	// Not cached: parse and cache.
	var (
		c   *compiled
		err error
	)
	if e.cfg.Dir != "" {
		c, err = e.parseFromDisk(id)
	} else {
		c, err = e.parseFromEmbed(id)
	}
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.cache[id] = c
	e.mu.Unlock()
	return c, nil
}

func (e *Engine) parseFromDisk(id string) (*compiled, error) {
	path := filepath.Join(e.cfg.Dir, id+".tmpl")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template from disk %q: %w", path, err)
	}
	return parseBoth(id, string(b))
}

func (e *Engine) parseFromEmbed(id string) (*compiled, error) {
	path := "files/" + id + ".tmpl"
	b, err := fs.ReadFile(e.fs, path)
	if err != nil {
		return nil, fmt.Errorf("read embedded template %q: %w", path, err)
	}
	return parseBoth(id, string(b))
}

func parseBoth(id, content string) (*compiled, error) {
	// text/template for subject, email_text, sms_text, push_title, push_body
	tText, err := texttmpl.New(id).Option("missingkey=error").Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse text blocks (%s): %w", id, err)
	}
	// html/template for email_html
	tHTML, err := htmltmpl.New(id).Option("missingkey=error").Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse html block (%s): %w", id, err)
	}
	return &compiled{text: tText, html: tHTML}, nil
}

func execText(t *texttmpl.Template, name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func execHTML(t *htmltmpl.Template, name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}