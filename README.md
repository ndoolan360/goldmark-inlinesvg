# goldmark-inlinesvg

A small [goldmark](https://github.com/yuin/goldmark) extension that **inlines local SVG images** into the rendered HTML. Non‑SVG images (or remote/data URLs) are left as standard `<img>` tags.

## Install

```sh
go get go.doolan.dev/goldmark/inlinesvg
```

## Behavior

- **Local SVGs** are inlined as `<svg>...</svg>` in the output.
  - The `alt` attribute is omitted from inlined SVGs, but the title attribute is preserved if provided.
- **Non‑SVG images** are rendered as `<img>` (including PNG, JPEG, etc).
- **Remote images** (`http://`, `https://`) are **not** inlined.
- **Data URLs** are left as `<img>` sources.

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

	"github.com/yuin/goldmark"
	"go.doolan.dev/goldmark/inlinesvg"
)

func main() {
	md := goldmark.New(
		goldmark.WithExtensions(
			inlinesvg.InlineSvg,
		),
	)

	var buf bytes.Buffer
	_ = md.Convert([]byte(`![alt text](./logo.svg "title")`), &buf)

	fmt.Println(buf.String())
	
	// Output:
  // <svg width="100" height="100" xmlns="http://www.w3.org/2000/svg" title="title">
  //  <path d="M10 10 H 90 V 90 H 10 Z"/>
  // </svg>
}
```

## Security

The extension respects Goldmark’s safety behavior:
- Dangerous URLs (e.g. `javascript:`) are **stripped** unless you enable `html.WithUnsafe()` on the renderer.

## License

This project is licensed under the BSD 3-Clause License. See the [LICENSE](LICENSE) file for details.
