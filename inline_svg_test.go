package inlinesvg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
)

const (
	svgContent          = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="green"/></svg>`
	svgTitle            = "svg title"
	svgContentWithTitle = `<svg title="svg title" xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="green"/></svg>`
	svgFileName         = "test.svg"
	absoluteSvgFileName = "/test.svg"
	pngContent          = "dummy png content"
	pngFileName         = "test.png"
	nonImageContent     = "dummy non-image content"
	nonImageFileName    = "test.txt"
)

func prepareTestFile(t *testing.T, dir, fileName, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, fileName)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return filePath
}

func TestGetImage(t *testing.T) {
	dir := t.TempDir()

	prepareTestFile(t, dir, svgFileName, svgContent)
	prepareTestFile(t, dir, pngFileName, pngContent)
	prepareTestFile(t, dir, nonImageFileName, nonImageContent)
	prepareTestFile(t, dir, absoluteSvgFileName, svgContent)

	tests := []struct {
		name        string
		source      string
		parentPath  string
		wantContent []byte
		wantMtype   string
		wantErr     bool
	}{
		{
			name:        "local svg",
			source:      svgFileName,
			parentPath:  dir,
			wantContent: []byte(svgContent),
			wantMtype:   "image/svg+xml",
		},
		{
			name:        "local png",
			source:      pngFileName,
			parentPath:  dir,
			wantContent: []byte(pngFileName),
			wantMtype:   "text/plain; charset=utf-8",
		},
		{
			name:        "local non-image",
			source:      nonImageFileName,
			parentPath:  dir,
			wantContent: []byte(nonImageFileName),
			wantMtype:   "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &inlineSVGRenderer{parentPath: tt.parentPath}

			gotContent, gotMtype, err := r.getImage([]byte(tt.source))

			if (err != nil) != tt.wantErr {
				t.Fatalf("getImage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if gotMtype != tt.wantMtype {
				t.Errorf("getImage() gotMtype = %s, want %s", gotMtype, tt.wantMtype)
			}
			if !bytes.Equal(gotContent, tt.wantContent) {
				t.Errorf("getImage() gotContent = %s, want %s", string(gotContent), string(tt.wantContent))
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	dir := t.TempDir()

	prepareTestFile(t, dir, svgFileName, svgContent)
	prepareTestFile(t, dir, absoluteSvgFileName, svgContent)
	prepareTestFile(t, dir, pngFileName, pngContent)
	prepareTestFile(t, dir, nonImageFileName, nonImageContent)

	tests := []struct {
		name          string
		source        string
		extOptions    []HTMLRendererOption
		parserOptions []parser.Option
		renderOptions []html.Option
		want          string
	}{
		{
			name:       "inline local svg relative path with ParentPath",
			source:     fmt.Sprintf(`![alt text](%s "%s")`, svgFileName, svgTitle),
			extOptions: []HTMLRendererOption{WithParentPath(dir)},
			want:       fmt.Sprintf("<p>%s</p>", svgContentWithTitle),
		},
		{
			name:       "inline local svg absolute path with ParentPath",
			source:     fmt.Sprintf(`![alt text](%s "%s")`, absoluteSvgFileName, svgTitle),
			extOptions: []HTMLRendererOption{WithParentPath(dir)},
			want:       fmt.Sprintf(`<p><img src="%s" alt="alt text" title="%s"></p>`, absoluteSvgFileName, svgTitle),
		},
		{
			name:   "render local png as img tag with relative path",
			source: fmt.Sprintf(`![alt text](%s "png title")`, pngFileName),
			want:   fmt.Sprintf(`<p><img src="%s" alt="alt text" title="png title"></p>`, pngFileName),
		},
		{
			name:          "render local png as img tag with XHTML option",
			source:        fmt.Sprintf(`![alt text](%s "png title")`, pngFileName),
			renderOptions: []html.Option{html.WithXHTML()},
			want:          fmt.Sprintf(`<p><img src="%s" alt="alt text" title="png title" /></p>`, pngFileName),
		},
		{
			name:   "render local non-image as img tag",
			source: fmt.Sprintf(`![alt text](%s "non-image title")`, nonImageFileName),
			want:   `<p><img src="test.txt" alt="alt text" title="non-image title"></p>`,
		},
		{
			name:   "render online svg as img tag",
			source: `![alt text](http://example.com/image.svg "svg title")`,
			want:   `<p><img src="http://example.com/image.svg" alt="alt text" title="svg title"></p>`,
		},
		{
			name:   "render data url png as img tag",
			source: `![alt text](data:image/png;base64,something= "data png title")`,
			want:   `<p><img src="data:image/png;base64,something=" alt="alt text" title="data png title"></p>`,
		},
		{
			name:   "strip data url svg in safe mode",
			source: `![alt text](data:image/svg+xml;base64,PHN2Zy8+ "data svg title")`,
			want:   `<p><img src="" alt="alt text" title="data svg title"></p>`,
		},
		{
			name:          "render data url svg in unsafe mode",
			source:        `![alt text](data:image/svg+xml;base64,PHN2Zy8+ "data svg title")`,
			renderOptions: []html.Option{html.WithUnsafe()},
			want:          `<p><img src="data:image/svg+xml;base64,PHN2Zy8+" alt="alt text" title="data svg title"></p>`,
		},
		{
			name:   "strip encoded dangerous url in safe mode",
			source: `![alt text](&#106;avascript:alert%281%29 "unsafe title")`,
			want:   `<p><img src="" alt="alt text" title="unsafe title"></p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.source)
			doc := parser.New(tt.parserOptions...).Parse(source)
			renderOptions := []html.Option{
				html.WithExtensions(NewHTMLRenderer(tt.extOptions...)),
			}
			renderOptions = append(renderOptions, tt.renderOptions...)
			r := html.New(renderOptions...)

			var buf bytes.Buffer
			if err := r.Render(&buf, source, doc); err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(buf.String()); got != strings.TrimSpace(tt.want) {
				t.Errorf("want:\n'%s'\ngot:\n'%s'", strings.TrimSpace(tt.want), got)
			}
		})
	}

	t.Run("unsafe rendering for non-svg", func(t *testing.T) {
		jsSource := `![alt text](javascript:alert('XSS') "unsafe title")`

		// Test safe rendering (default)
		safeWant := `<p><img src="" alt="alt text" title="unsafe title"></p>`
		source := []byte(jsSource)
		doc := parser.New().Parse(source)
		rSafe := html.New(
			html.WithExtensions(NewHTMLRenderer(WithParentPath(dir))),
		)
		var bufSafe bytes.Buffer
		if err := rSafe.Render(&bufSafe, source, doc); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(bufSafe.String()); got != strings.TrimSpace(safeWant) {
			t.Errorf("Safe rendering want:\n'%s'\ngot:\n'%s'", strings.TrimSpace(safeWant), got)
		}

		// Test unsafe rendering
		unsafeWant := `<p><img src="javascript:alert('XSS')" alt="alt text" title="unsafe title"></p>`
		rUnsafe := html.New(
			html.WithExtensions(NewHTMLRenderer(WithParentPath(dir))),
			html.WithUnsafe(),
		)
		var bufUnsafe bytes.Buffer
		if err := rUnsafe.Render(&bufUnsafe, source, doc); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(bufUnsafe.String()); got != strings.TrimSpace(unsafeWant) {
			t.Errorf("Unsafe rendering want:\n'%s'\ngot:\n'%s'", strings.TrimSpace(unsafeWant), got)
		}
	})
}
