package inlinesvg

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// This extension simply inlines SVG images, and leaves other images as is.

// Option interface sets options for this extension.
type Option interface {
	inlineSvgOption()
}

// InlineSvgConfig embeds html.Config to refer to some fields like unsafe and xhtml.
type InlineSvgConfig struct {
	html.Config
	ParentPath string
}

// SetOption implements renderer.NodeRenderer.SetOption
func (c *InlineSvgConfig) SetOption(name renderer.OptionName, value any) {
	c.Config.SetOption(name, value)
}

type rendererOption interface {
	Option
	SetInlineSvgOption(*InlineSvgConfig)
}

func WithParentPath(path string) Option {
	return &withParentPath{path}
}

type withParentPath struct {
	path string
}

func (o *withParentPath) inlineSvgOption() {}

func (o *withParentPath) SetInlineSvgOption(c *InlineSvgConfig) {
	c.ParentPath = o.path
}

type inlineSvgRenderer struct {
	InlineSvgConfig
}

func NewInlineSvgRenderer(opts ...rendererOption) renderer.NodeRenderer {
	r := &inlineSvgRenderer{
		InlineSvgConfig: InlineSvgConfig{
			Config: html.NewConfig(),
		},
	}
	for _, o := range opts {
		o.SetInlineSvgOption(&r.InlineSvgConfig)
	}
	return r
}

func (r *inlineSvgRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *inlineSvgRenderer) getImage(src []byte) ([]byte, string, error) {
	s := string(src)
	// do not encode online image
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return src, "online", nil
	}
	// already encoded
	if strings.HasPrefix(s, "data:") {
		return src, "data", nil
	}
	if !filepath.IsAbs(s) && r.ParentPath != "" {
		s = filepath.Join(r.ParentPath, s)
	} else if filepath.IsAbs(s) {
		s = filepath.Join(".", s)
	}
	f, err := os.Open(filepath.Clean(s))
	if err != nil {
		return nil, "", fmt.Errorf("fail to open %s: %w", s, err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, "", fmt.Errorf("fail to read %s: %w", s, err)
	}
	mtype := mimetype.Detect(b)

	if !mtype.Is("image/svg+xml") {
		return src, mtype.String(), nil
	}

	return b, "image/svg+xml", nil
}

// renderImage adds svg embedding function to github.com/yuin/goldmark/renderer/html (MIT).
func (r *inlineSvgRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)
	src, mtype, err := r.getImage(n.Destination)

	if err == nil && mtype == "image/svg+xml" {
		split := bytes.Split(src, []byte("<svg"))
		_, _ = w.Write(split[0])
		_, _ = w.WriteString("<svg")
		r.applyAttributes(w, n, source, true)
		_, _ = w.Write(split[1])
	} else {
		_, _ = w.WriteString(`<img`)
		r.writeSource(w, n)
		r.applyAttributes(w, n, source, false)
		if r.XHTML {
			_, _ = w.WriteString(" />")
		} else {
			_, _ = w.WriteString(">")
		}
	}

	return ast.WalkSkipChildren, nil
}

// writeSource writes the src attribute of the image element
func (r *inlineSvgRenderer) writeSource(w util.BufWriter, n *ast.Image) {
	_, _ = w.WriteString(` src="`)
	src := util.URLEscape(n.Destination, true)
	if r.Unsafe || !html.IsDangerousURL(src) {
		_, _ = w.Write(util.EscapeHTML(src))
	}
	_, _ = w.WriteString(`"`)
}

// applyAttributes writes attributes of the image element
func (r *inlineSvgRenderer) applyAttributes(w util.BufWriter, n *ast.Image, source []byte, exclAlt bool) {
	if !exclAlt {
		_, _ = w.WriteString(` alt="`)
		_, _ = w.Write(nodeToHTMLText(n, source))
		_ = w.WriteByte('"')
	}
	if n.Title != nil {
		_, _ = w.WriteString(` title="`)
		r.Writer.Write(w, n.Title)
		_ = w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, html.ImageAttributeFilter)
	}
}

// nodeToHTMLText converts ast.Node to HTML text
func nodeToHTMLText(n ast.Node, source []byte) []byte {
	var buf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if s, ok := c.(*ast.String); ok && s.IsCode() {
			buf.Write(s.Value)
		} else if !c.HasChildren() {
			buf.Write(util.EscapeHTML(c.Text(source)))
		} else {
			buf.Write(nodeToHTMLText(c, source))
		}
	}
	return buf.Bytes()
}

// inlineSvg implements goldmark.Extender
type inlineSvg struct {
	options []Option
}

// InlineSvg is an implementation of goldmark.Extender
var InlineSvg = &inlineSvg{}

// New returns a new InlineSvg extension.
func New(opts ...Option) goldmark.Extender {
	return &inlineSvg{
		options: opts,
	}
}

func (e *inlineSvg) Extend(m goldmark.Markdown) {
	ropts := []rendererOption{}
	for _, opt := range e.options {
		if ropt, ok := opt.(rendererOption); ok {
			ropts = append(ropts, ropt)
		}
	}

	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewInlineSvgRenderer(ropts...), 501),
	))
}
