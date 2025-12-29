# Chrome Browser Context Tools

You have direct access to the user's active Chrome browser tab. These tools let you read, analyze, and modify web pages in real-time.

## Quick Reference

| Tool | Best For |
|------|----------|
| `get_page_text` | **Reading content** - Gets visible text only (no HTML). Start here for summarization. |
| `save_page_to_file` | **Large pages** - Downloads to `~/Library/Application Support/ChromeGeminiSync/pages/` for analysis |
| `get_browser_dom` | Getting HTML structure, element attributes, page layout |
| `inspect_page` | Checking page size before fetching (use for large/complex sites) |
| `get_browser_url` | Getting current URL and title |
| `get_browser_selection` | Getting text the user highlighted |
| `capture_browser_screenshot` | Visual context, charts, images, layout issues |
| `execute_browser_script` | Running JavaScript and getting return values |
| `modify_dom` | Changing page content, removing elements, adding content |
| `get_console_logs` | Debugging, checking for JavaScript errors |

---

## Reading Page Content

### get_page_text (Recommended for content)

**Use this first for reading page content.** Returns only visible text, no HTML tags. Much smaller and cleaner than DOM.

```js
// Get all visible text
get_page_text({})

// Get text from specific section
get_page_text({ selector: "article" })
get_page_text({ selector: "main" })
get_page_text({ selector: ".content" })

// Limit length for very long pages
get_page_text({ maxLength: 20000 })
```

### get_browser_dom

Use when you need actual HTML structure, element attributes, or CSS classes.

```js
// Get specific element
get_browser_dom({ selector: "nav" })
get_browser_dom({ selector: "#main-content" })
get_browser_dom({ selector: ".product-card" })
```

**Note:** Returns first matching element only. For large pages, use `get_page_text` instead.

### inspect_page

Check page complexity before deciding how to fetch content.

```js
inspect_page({})
// Returns: { stats: { htmlLength, elementCount }, recommendation: "direct" | "download" }
```

### save_page_to_file

**For large/complex pages.** Downloads page content to a local file so you can analyze it with your standard file reading tools.

Files are saved to `~/Library/Application Support/ChromeGeminiSync/pages/` (within Gemini's allowed workspace).

```js
// Save as plain text (default, best for analysis)
save_page_to_file({ format: "text" })

// Save as markdown (preserves some formatting)
save_page_to_file({ format: "markdown" })

// Save as cleaned HTML (no scripts/styles)
save_page_to_file({ format: "html" })

// Custom filename
save_page_to_file({ format: "text", filename: "cnn-news.txt" })
```

**Returns:** `{ filePath: "/tmp/browser-pages/...", size: 12345, url: "...", title: "..." }`

**After calling this, use your file tools to read and analyze the content:**
```
Read the file at /tmp/browser-pages/Breaking News-1234567890.txt
```

---

## Executing JavaScript

### execute_browser_script

Runs JavaScript in the page and returns the result. **You must use `return` to get data back.**

```js
// CORRECT - returns data
execute_browser_script({ script: "return document.querySelectorAll('h2').length" })

execute_browser_script({ script: "return Array.from(document.querySelectorAll('a')).map(a => ({text: a.textContent, href: a.href}))" })

execute_browser_script({ script: "return document.body.innerText.length" })

// WRONG - no return, will return undefined
execute_browser_script({ script: "document.title" })
```

---

## Modifying Pages

### modify_dom

Change page content with these actions:

| Action | Description | Needs `value` | Needs `attributeName` |
|--------|-------------|---------------|----------------------|
| `setText` | Set text content | Yes | No |
| `setHTML` | Set inner HTML | Yes | No |
| `setAttribute` | Set attribute | Yes | Yes |
| `removeAttribute` | Remove attribute | No | Yes |
| `addClass` | Add CSS class | Yes (class name) | No |
| `removeClass` | Remove CSS class | Yes (class name) | No |
| `remove` | Delete element | No | No |
| `insertBefore` | Add HTML before | Yes (HTML) | No |
| `insertAfter` | Add HTML after | Yes (HTML) | No |

```js
// Change text
modify_dom({ selector: "h1", action: "setText", value: "New Title" })

// Remove all ads
modify_dom({ selector: ".ad, .advertisement", action: "remove", all: true })

// Add a class
modify_dom({ selector: "#header", action: "addClass", value: "sticky" })

// Set attribute
modify_dom({ selector: "img", action: "setAttribute", attributeName: "loading", value: "lazy", all: true })

// Insert banner
modify_dom({ selector: "header", action: "insertAfter", value: "<div style='background:yellow;padding:10px'>Notice!</div>" })
```

Set `all: true` to affect all matching elements, not just the first.

---

## Visual & Context

### capture_browser_screenshot

Returns PNG image of the visible viewport.

```js
capture_browser_screenshot({})
```

### get_browser_url

Get current page URL and title.

```js
get_browser_url({})
// Returns: { url: "...", title: "..." }
```

### get_browser_selection

Get text the user has highlighted.

```js
get_browser_selection({})
// Returns: { text: "selected text", url: "...", title: "..." }
```

---

## Debugging

### get_console_logs

Get console messages (first call attaches debugger).

```js
// Get errors only
get_console_logs({ level: "error" })

// Get all logs
get_console_logs({ level: "all" })

// Get and clear
get_console_logs({ level: "all", clear: true })
```

---

## Best Practices

1. **Start with `get_page_text`** for reading content - it's smaller and cleaner than DOM
2. **For large/complex pages** - use `save_page_to_file` to download, then analyze with your file tools
3. **Use specific selectors** - target `article`, `main`, `.content` instead of full body
4. **Scripts must return** - always use `return` in `execute_browser_script`
5. **Use `all: true`** when modifying multiple elements

### Workflow for Large Pages

1. Call `inspect_page({})` to check size
2. If `recommendation: "download"`, call `save_page_to_file({ format: "text" })`
3. Read the returned `filePath` with your standard file tools
4. Analyze the content normally

## Limitations

- Cannot access `chrome://` pages, extension pages, or `file://` URLs
- DOM changes are temporary (lost on page refresh)
- Some sites block script execution (CSP)
- Only works on the active tab
