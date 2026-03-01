package inlinesvg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	svgContent          = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="green"/></svg>`
	svgTitle            = "svg title"
	svgContentWithTitle = `<svg title="svg title" xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="green"/></svg>`
	svgFileName         = "test.svg"
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
			var opts []rendererOption
			if tt.parentPath != "" {
				opts = append(opts, WithParentPath(tt.parentPath).(rendererOption))
			}
			r := NewInlineSvgRenderer(opts...).(*inlineSvgRenderer)

			gotContent, gotMtype, err := r.getImage([]byte(tt.source))

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
	prepareTestFile(t, dir, pngFileName, pngContent)
	prepareTestFile(t, dir, nonImageFileName, nonImageContent)

	tests := []struct {
		name          string
		source        string
		extOptions    []Option
		renderOptions []renderer.Option
		want          string
	}{
		{
			name:       "inline local svg relative path with ParentPath",
			source:     fmt.Sprintf(`![alt text](%s "%s")`, svgFileName, svgTitle),
			extOptions: []Option{WithParentPath(dir)},
			want:       fmt.Sprintf("<p>%s</p>", svgContentWithTitle),
		},
		{
			name:   "render local png as img tag absolute path",
			source: fmt.Sprintf(`![alt text](%s "png title")`, pngFileName),
			want:   fmt.Sprintf(`<p><img src="%s" alt="alt text" title="png title"></p>`, pngFileName),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var allRenderOptions []renderer.Option
			if tt.renderOptions != nil {
				allRenderOptions = append(allRenderOptions, tt.renderOptions...)
			}

			md := goldmark.New(
				goldmark.WithExtensions(New(tt.extOptions...)),
				goldmark.WithRendererOptions(allRenderOptions...),
			)
			var buf bytes.Buffer
			if err := md.Convert([]byte(tt.source), &buf); err != nil {
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
		mdSafe := goldmark.New(
			goldmark.WithExtensions(New(WithParentPath(dir))),
		)
		var bufSafe bytes.Buffer
		if err := mdSafe.Convert([]byte(jsSource), &bufSafe); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(bufSafe.String()); got != strings.TrimSpace(safeWant) {
			t.Errorf("Safe rendering want:\n'%s'\ngot:\n'%s'", strings.TrimSpace(safeWant), got)
		}

		// Test unsafe rendering
		unsafeWant := `<p><img src="javascript:alert('XSS')" alt="alt text" title="unsafe title"></p>`
		mdUnsafe := goldmark.New(
			goldmark.WithExtensions(New(WithParentPath(dir))),
			goldmark.WithRendererOptions(
				html.WithUnsafe(),
			),
		)
		var bufUnsafe bytes.Buffer
		if err := mdUnsafe.Convert([]byte(jsSource), &bufUnsafe); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(bufUnsafe.String()); got != strings.TrimSpace(unsafeWant) {
			t.Errorf("Unsafe rendering want:\n'%s'\ngot:\n'%s'", strings.TrimSpace(unsafeWant), got)
		}
	})
}
