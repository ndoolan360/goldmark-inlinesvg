package inlinesvg

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/util"
)

// HTMLRendererOption configures the inline SVG HTML renderer.
type HTMLRendererOption func(*htmlRendererConfig)

type htmlRendererConfig struct {
	parentPath string
}

// WithParentPath sets the base directory used to resolve relative image paths.
func WithParentPath(path string) HTMLRendererOption {
	return func(c *htmlRendererConfig) {
		c.parentPath = path
	}
}

type inlineSVGRenderer struct {
	config     *html.Config
	parentPath string
}

func (r *inlineSVGRenderer) getImage(src []byte) ([]byte, string, error) {
	s := string(src)
	// Do not inline remote images.
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return src, "online", nil
	}
	// Data URLs are already encoded.
	if strings.HasPrefix(s, "data:") {
		return src, "data", nil
	}
	if !filepath.IsAbs(s) && r.parentPath != "" {
		s = filepath.Join(r.parentPath, s)
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

func (r *inlineSVGRenderer) renderImage(
	writer io.Writer,
	source []byte,
	node ast.Node,
	entering bool,
	rc renderer.Context,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	w := writer.(util.BufWriter)
	n := node.(*ast.Image)
	destination := n.Destination.Value(source)
	svg, mtype, err := r.getImage([]byte(destination))

	if err == nil && mtype == "image/svg+xml" {
		before, after, found := bytes.Cut(svg, []byte("<svg"))
		if found {
			hw := html.ContextHTMLWriter(rc)
			_, _ = hw.Write(before)
			_, _ = w.WriteString("<svg")
			r.applyAttributes(w, n, source, rc, true)
			_, _ = hw.Write(after)
			return ast.WalkSkipChildren, nil
		}
	}

	_, _ = w.WriteString(`<img`)
	r.writeSource(w, n, source, rc)
	r.applyAttributes(w, n, source, rc, false)
	if r.config.XHTML {
		_, _ = w.WriteString(" />")
	} else {
		_ = w.WriteByte('>')
	}

	return ast.WalkSkipChildren, nil
}

func (r *inlineSVGRenderer) writeSource(
	w util.BufWriter,
	n *ast.Image,
	source []byte,
	rc renderer.Context,
) {
	_, _ = w.WriteString(` src="`)
	src := n.Destination.Value(source)
	if r.config.Unsafe || !html.IsDangerousURL(src) {
		_, _ = n.Destination.WriteTo(html.ContextLinkURLWriter(rc), source)
	}
	_ = w.WriteByte('"')
}

func (r *inlineSVGRenderer) applyAttributes(
	w util.BufWriter,
	n *ast.Image,
	source []byte,
	rc renderer.Context,
	excludeAlt bool,
) {
	if !excludeAlt {
		_, _ = w.WriteString(` alt="`)
		writeNodeText(html.ContextTextWriter(rc), n, source)
		_ = w.WriteByte('"')
	}
	if !n.Title.IsEmpty() {
		_, _ = w.WriteString(` title="`)
		_, _ = n.Title.WriteTo(html.ContextTextWriter(rc), source)
		_ = w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, source, n, html.ImageAttributeFilter, rc)
	}
}

func writeNodeText(w io.Writer, n ast.Node, source []byte) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch child := child.(type) {
		case *ast.Text:
			_, _ = child.Value.WriteTo(w, source)
		case *ast.CodeSpan:
			_, _ = child.Value.WriteTo(w, source)
		default:
			writeNodeText(w, child, source)
		}
	}
}

type inlineSVGHTMLRendererExtension struct {
	options []HTMLRendererOption
}

// NewHTMLRenderer returns an HTML renderer extension that inlines local SVG images.
func NewHTMLRenderer(opts ...HTMLRendererOption) html.Extension {
	return &inlineSVGHTMLRendererExtension{options: opts}
}

func (e *inlineSVGHTMLRendererExtension) RendererOptions(config *html.Config) []html.Option {
	cfg := htmlRendererConfig{}
	for _, option := range e.options {
		option(&cfg)
	}

	r := &inlineSVGRenderer{
		config:     config,
		parentPath: cfg.parentPath,
	}
	return []html.Option{
		html.WithNodeRenderer(ast.KindImage, html.NodeRendererFunc(r.renderImage)),
	}
}

// HTMLRenderer is the default HTML renderer extension.
var HTMLRenderer = NewHTMLRenderer()
