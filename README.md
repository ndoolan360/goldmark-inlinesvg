# goldmark-inlinesvg

A small [goldmark](https://github.com/yuin/goldmark) extension that **inlines local SVG images** into the rendered HTML. Non‑SVG images (or remote/data URLs) are left as standard `<img>` tags.

## Install

```sh
go get go.doolan.dev/goldmark/inlinesvg
```

## Behavior

- **Local SVGs** are inlined as `<svg>...</svg>` in the output.
- **Non‑SVG images** are rendered as `<img>` (including PNG, JPEG, etc).
- **Remote images** (`http://`, `https://`) are **not** inlined.
- **Data URLs** are left as `<img>` sources.
- **Alt text** is derived from plain text only; inline formatting is ignored.

## Options

### `WithParentPath(path string)`

Sets the base directory used to resolve relative image paths.

## Usage

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
}
```

## Security

The extension respects Goldmark’s safety behavior:
- Dangerous URLs (e.g. `javascript:`) are **stripped** unless you enable `html.WithUnsafe()` on the renderer.

## License

This project is licensed under the BSD 3-Clause License. See the [LICENSE](LICENSE) file for details.
