# Browser Context Tools

You can interact with the user's active Chrome tab using these tools.

## Quick Reference

| Tool | Use when... |
|------|-------------|
| `get_console_logs` | User asks about errors, bugs, or "what's wrong with this page" |
| `inspect_page` | **ALWAYS USE FIRST** to check page size/complexity before getting DOM |
| `get_browser_dom` | User asks to look at, analyze, or read page content |
| `get_browser_url` | User asks "what page is this" or you need the current URL |
| `get_browser_selection` | User says "this text" or "what I highlighted" |
| `capture_browser_screenshot` | User asks to "look at" the page visually |
| `modify_dom` | User wants to change something on the page |
| `execute_browser_script` | You need to read JS variables (use sparingly) |

---

## inspect_page

**Use this FIRST before getting the DOM to decide the best strategy.**

Returns stats about the page size, element count, and a recommendation on whether the page is too large.

```js
inspect_page({})
```

**Returns:** JSON object with `stats` (htmlLength, elementCount), `meta` (description, headings), and `recommendation` ("direct" or "download").

---

## get_console_logs

**Use this when the user asks about errors, bugs, or debugging.**

Gets errors, warnings, and logs from the browser's DevTools console.

```js
// "Check for errors on this page"
get_console_logs({ level: "error" })

// "Show me all console output"
get_console_logs({})

// "What warnings are there?"
get_console_logs({ level: "warning" })
```

**Parameters:**
- `level`: `"all"` | `"error"` | `"warning"` | `"info"` (default: `"all"`)
- `clear`: `true` to clear logs after reading

**Note:** First call attaches debugger (shows banner to user). Logs accumulate until cleared.

---

## get_browser_dom

**Use this when the user asks you to look at, analyze, or work with page content.**

```js
// "What's on this page?"
get_browser_dom({})

// "Look at the navigation menu"
get_browser_dom({ selector: "nav" })

// "What does the footer say?"
get_browser_dom({ selector: "footer" })

// "Show me all the product cards"
get_browser_dom({ selector: ".product-card" })
```

**Parameters:**
- `selector`: CSS selector (default: `"body"`)

---

## get_browser_url

**Use this when you need to know what page the user is on.**

```js
get_browser_url({})
```

Returns URL and page title.

---

## get_browser_selection

**Use this when the user refers to highlighted/selected text.**

```js
// "Explain this text I selected"
get_browser_selection({})
```

---

## capture_browser_screenshot

**Use this when you need to see the visual appearance.**

```js
capture_browser_screenshot({})
```

Returns a PNG image.

---

## modify_dom

**Use this for ANY changes to the page.**

```js
// "Change the heading to say Hello"
modify_dom({ selector: "h1", action: "setText", value: "Hello" })

// "Hide all the ads"
modify_dom({ selector: ".ad", action: "remove", all: true })

// "Make the background red"
modify_dom({ selector: "body", action: "setAttribute", attributeName: "style", value: "background: red" })

// "Add a banner after the header"
modify_dom({ selector: "header", action: "insertAfter", value: "<div style='background:yellow;padding:10px'>Banner!</div>" })
```

**Actions:**
- `setText` - Change text content
- `setHTML` - Change inner HTML
- `setAttribute` - Set an attribute (use with `attributeName`)
- `removeAttribute` - Remove an attribute
- `addClass` / `removeClass` - Modify classes
- `remove` - Delete the element
- `insertBefore` / `insertAfter` - Add HTML adjacent to element

**Parameters:**
- `selector`: CSS selector (required)
- `action`: One of the actions above (required)
- `value`: The new content/value
- `attributeName`: For setAttribute/removeAttribute
- `all`: `true` to affect all matching elements

---

## execute_browser_script

**Only use this for reading JavaScript variables. Do NOT use for DOM changes.**

May fail on sites with strict CSP. If it fails, use `get_browser_dom` instead.

```js
// "What's the scroll position?"
execute_browser_script({ script: "window.scrollY" })
```

---

## Limitations

- Can't access `chrome://` pages or other extensions
- DOM changes are temporary (lost on page refresh)
- `execute_browser_script` may fail on sites with strict Content Security Policy
- Extension must be connected - if tools fail, tell user to open the extension side panel

## Best Practices

### 1. Check Page Size First
Always run `inspect_page` before `get_browser_dom` unless you are sure the page is small.

### 2. Targeted Extraction
If the user asks about a specific part of the page (e.g., "what's in the footer?"), prefer `get_browser_dom({ selector: "footer" })` over getting the whole body.

### 3. Visual vs Code
- Use `capture_browser_screenshot` for layout/visual questions.
- Use `get_browser_dom` for content/data questions.
- Use `inspect_page` to get a high-level summary without reading the whole DOM.
