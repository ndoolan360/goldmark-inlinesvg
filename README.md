# goldmark-inlinesvg

A small [goldmark](https://github.com/yuin/goldmark) extension that **inlines local SVG images** into the rendered HTML. Non‑SVG images (or remote/data URLs) are left as standard `<img>` tags.

## Install

```sh
go get go.doolan.dev/goldmark/inlinesvg/v2
```

## Behavior

- **Local SVGs** are inlined as `<svg>...</svg>` in the output.
  - The `alt` attribute is omitted from inlined SVGs, but the title attribute is preserved if provided.
- **Non‑SVG images** are rendered as `<img>` (including PNG, JPEG, etc).
- **Remote images** (`http://`, `https://`) are **not** inlined.
- **Safe raster data URLs** (PNG, GIF, JPEG, and WebP) are left as `<img>` sources.
- **SVG data URLs** are stripped in safe mode because they can execute active content. Enable `html.WithUnsafe()` only if you trust the Markdown input.

## Options

### `WithParentPath(path string)`

Sets the base directory used to resolve relative image paths.

## Usage

```xml
<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
  <path d="M10 10 H 90 V 90 H 10 Z"/>
</svg>
```

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
	"go.doolan.dev/goldmark/inlinesvg/v2"
)

func main() {
	source := []byte(`![alt text](./logo.svg "title")`)
	doc := parser.New().Parse(source)
	renderer := html.New(
		html.WithExtensions(inlinesvg.HTMLRenderer),
	)

	var buf bytes.Buffer
	_ = renderer.Render(&buf, source, doc)

	fmt.Println(buf.String())

	// Output:
	// <p><svg width="100" height="100" xmlns="http://www.w3.org/2000/svg" title="title">
	//  <path d="M10 10 H 90 V 90 H 10 Z"/>
	// </svg></p>
}
```

To resolve relative image paths from a specific directory, construct a configured renderer extension:

```go
renderer := html.New(
	html.WithExtensions(
		inlinesvg.NewHTMLRenderer(inlinesvg.WithParentPath("./assets")),
	),
)
```

## Security

The extension respects Goldmark’s safety behavior:
- Dangerous URLs (for example, `javascript:` and SVG data URLs) are **stripped** unless you enable `html.WithUnsafe()` on the renderer.
- URL safety is checked after Markdown entity decoding, so entity-encoded dangerous schemes are also stripped.

## License

This project is licensed under the BSD 3-Clause License. See the [LICENSE](LICENSE) file for details.
