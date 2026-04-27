---
title: Plain Markdown Sample
description: A reference document showcasing every GFM feature the docs renderer supports.
---

# Plain Markdown Sample

This file is rendered by **remark-gfm** in the browser. It exercises every
markdown feature the docs page supports — useful as a visual regression check
when you tweak the prose/typography styles.

For a sibling document that mixes markdown with live React components, open
[sample.mdx](/docs/sample.mdx).

## Inline formatting

You can write **bold**, *italic*, ***bold italic***, `inline code`, and
~~strikethrough~~. Links look like [this one](https://example.com), and
auto-links like https://golinks.example.com are picked up by GFM.

## Lists

Unordered:

- Memorable shortcuts for long URLs
- Variable substitution with `{*}` placeholders
- Single-binary deployment

Ordered:

1. Open the homepage
2. Add a keyword
3. Type `go <keyword>` in the address bar

Task list:

- [x] Render markdown
- [x] Render MDX with components
- [ ] Authenticate uploads

Nested:

- Frontend
  - React
  - Tailwind
  - shadcn/ui
- Backend
  - Go
  - SQLite
  - `embed.FS`

## Code blocks

Go, with syntax highlighting:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, GoLinks!")
}
```

JavaScript:

```javascript
const greet = (name) => `Hello, ${name}!`;
console.log(greet("GoLinks"));
```

Shell:

```bash
curl -X POST -H 'Content-Type: application/json' \
  -d '{"word":"docs","link":"https://docs.example.com"}' \
  http://localhost:8080/api/links
```

Inline code: `go run ./cmd/server`.

## Tables

| Feature             | Markdown | MDX | Notes                                  |
| ------------------- | :------: | :-: | -------------------------------------- |
| GFM tables          |    ✅     |  ✅  | Rendered with shadcn `Table` styling   |
| Strikethrough       |    ✅     |  ✅  | `~~text~~`                             |
| Task lists          |    ✅     |  ✅  | `- [x]` syntax                         |
| Syntax highlighting |    ✅     |  ✅  | `rehype-highlight`, GitHub theme       |
| React components    |    ❌     |  ✅  | Only `.mdx` can embed components       |

## Blockquotes

> Good design is as little design as possible.
>
> — Dieter Rams, *Ten Principles for Good Design*

## Horizontal rule

---

## Headings deeper than two

### Third level

Body copy under a third-level heading.

#### Fourth level

Body copy under a fourth-level heading.

##### Fifth level

Body copy under a fifth-level heading.

## Wrap-up

If something here looks broken, the issue is most likely in
`web/frontend/src/index.css` (prose overrides) or
`web/frontend/src/lib/mdx.tsx` (component map). Both files are small and
self-contained.
