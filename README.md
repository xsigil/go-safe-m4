# go-safe-m4

A deterministic, sandboxed, and secure-by-default Pure-Go implementation of the POSIX M4 macro processor.

`go-safe-m4` strips dangerous side effects (such as arbitrary OS command execution via `syscmd` and arbitrary path traversals via `include`) while adding modern escaping primitives like Rails-style `h()` for safe template rendering across Web frameworks, IaC tooling, and LLM pipelines.

---

## Key Features

- **Safe by Default (Hermetic Sandbox)**:
  - `syscmd` / `esyscmd` are disabled by default.
  - `include` enforces sandbox boundaries to prevent path traversal (`../`) attacks.
  - Recursion depth and output buffer guards prevent resource exhaustion.
- **Modern Escaping Primitives**:
  - `h(...)` / `html_escape(...)`: HTML entity escaping (XSS prevention).
  - `json_escape(...)`: Safe escaping for embedding text inside JSON string literals.
  - `sh_quote(...)`: Safe single-quote wrapping for shell argument passing.
- **Pure-Go & Zero Dependencies**:
  - No CGo, no external binaries required.
  - Trivial to embed into Go Web frameworks (e.g., Gin, Echo, Fiber) or CLI tools.
- **POSIX M4 Primitives Supported**:
  - `define`, `undefine`, `ifdef`, `ifelse`, `changequote`, `divert`, `undivert`, `dnl`.

---

## Usage

### Go Library

```go
package main

import (
	"fmt"
	"log"

	"[github.com/xsigil/go-safe-m4/pkg/engine](https://github.com/xsigil/go-safe-m4/pkg/engine)"
)

func main() {
	eng := engine.NewEngine(
		engine.WithSandboxRoot("./templates"),
	)

	template := `
define(` + "`GREET'" + `, ` + "`Hello, h($1)!'" + `)dnl
GREET(` + "`<script>alert('world')</script>'" + `)
`

	out, err := eng.Expand(template)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(out)
	// Output: Hello, &lt;script&gt;alert(&#39;world&#39;)&lt;/script&gt;!
}

```

---

## License

MIT
