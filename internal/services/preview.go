package services

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// PreviewService handles markdown preview functionality
type PreviewService struct {
	tempDir string
}

// NewPreviewService creates a new preview service
func NewPreviewService() *PreviewService {
	return &PreviewService{
		tempDir: os.TempDir(),
	}
}

// PreviewMarkdown converts markdown to HTML and opens it in the default browser
func (p *PreviewService) PreviewMarkdown(markdownPath, content string) error {
	// Convert markdown to HTML
	htmlContent, err := p.markdownToHTML(content, markdownPath)
	if err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}

	// Create temporary HTML file
	tempFile := filepath.Join(p.tempDir, "notetkr-preview.html")
	if err := os.WriteFile(tempFile, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Open in default browser
	if err := p.openInBrowser(tempFile); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

// markdownToHTML converts markdown content to styled HTML
func (p *PreviewService) markdownToHTML(markdown, sourcePath string) (string, error) {
	// Strip YAML front matter if present
	stripped := p.stripFrontMatter(markdown)

	// Configure goldmark with extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,   // GitHub Flavored Markdown
			extension.Table, // Tables
			extension.Strikethrough,
			extension.TaskList, // - [ ] checkboxes
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Auto-generate heading IDs
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // Respect line breaks
			html.WithXHTML(),     // XHTML-compliant output
		),
	)

	// Convert markdown to HTML
	var buf bytes.Buffer
	if err := md.Convert([]byte(stripped), &buf); err != nil {
		return "", err
	}

	// Get the directory of the source file for relative image paths
	sourceDir := filepath.Dir(sourcePath)

	// Wrap in full HTML document with styling
	html := p.wrapHTML(buf.String(), filepath.Base(sourcePath), sourceDir)
	return html, nil
}

// stripFrontMatter removes YAML front matter from markdown content
func (p *PreviewService) stripFrontMatter(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Check if first line is "---" (YAML front matter delimiter)
	firstLine := strings.TrimSpace(lines[0])
	if firstLine == "---" {
		// Find the closing "---"
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				// Found closing delimiter, return everything after it
				if i+1 < len(lines) {
					return strings.Join(lines[i+1:], "\n")
				}
				return ""
			}
		}
		// No closing delimiter found, return original
		return content
	}

	// Check if first line starts with "tags:" or "keywords:" (front matter without delimiters)
	if !strings.HasPrefix(firstLine, "tags:") && !strings.HasPrefix(firstLine, "keywords:") {
		// No front matter
		return content
	}

	// Skip all lines that are front matter (tags:, keywords:, or continuation values)
	startIdx := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Empty line marks end of front matter
		if trimmed == "" {
			startIdx = i + 1
			break
		}

		// Check if this is a front matter key (tags:, keywords:, etc.) or continuation
		isFrontMatterKey := strings.HasPrefix(trimmed, "tags:") ||
			strings.HasPrefix(trimmed, "keywords:") ||
			strings.HasPrefix(trimmed, "attendees:")

		// If it's not a front matter key and doesn't start with spaces/indentation,
		// it's the start of actual content
		if !isFrontMatterKey && i > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
			startIdx = i
			break
		}
	}

	// Return content without front matter
	if startIdx > 0 && startIdx < len(lines) {
		return strings.Join(lines[startIdx:], "\n")
	}

	return content
}

// wrapHTML wraps the markdown HTML in a complete HTML document with styling
func (p *PreviewService) wrapHTML(content, title, baseDir string) string {
	// Convert baseDir to file:// URL for proper image loading
	basePath := filepath.ToSlash(baseDir)

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Preview</title>
    <base href="file:///%s/">
    <style>
        :root {
            --bg-color: #ffffff;
            --text-color: #24292e;
            --border-color: #e1e4e8;
            --code-bg: #f6f8fa;
            --link-color: #0366d6;
        }
        
        @media (prefers-color-scheme: dark) {
            :root {
                --bg-color: #0d1117;
                --text-color: #c9d1d9;
                --border-color: #30363d;
                --code-bg: #161b22;
                --link-color: #58a6ff;
            }
        }
        
        * {
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
            font-size: 16px;
            line-height: 1.6;
            color: var(--text-color);
            background-color: var(--bg-color);
            max-width: 980px;
            margin: 0 auto;
            padding: 45px;
        }
        
        h1, h2, h3, h4, h5, h6 {
            margin-top: 24px;
            margin-bottom: 16px;
            font-weight: 600;
            line-height: 1.25;
        }
        
        h1 {
            font-size: 2em;
            padding-bottom: 0.3em;
            border-bottom: 1px solid var(--border-color);
        }
        
        h2 {
            font-size: 1.5em;
            padding-bottom: 0.3em;
            border-bottom: 1px solid var(--border-color);
        }
        
        h3 { font-size: 1.25em; }
        h4 { font-size: 1em; }
        h5 { font-size: 0.875em; }
        h6 { font-size: 0.85em; color: #6a737d; }
        
        p {
            margin-top: 0;
            margin-bottom: 16px;
        }
        
        a {
            color: var(--link-color);
            text-decoration: none;
        }
        
        a:hover {
            text-decoration: underline;
        }
        
        code {
            padding: 0.2em 0.4em;
            margin: 0;
            font-size: 85%%;
            background-color: var(--code-bg);
            border-radius: 6px;
            font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
        }
        
        pre {
            padding: 16px;
            overflow: auto;
            font-size: 85%%;
            line-height: 1.45;
            background-color: var(--code-bg);
            border-radius: 6px;
            margin-bottom: 16px;
        }
        
        pre code {
            padding: 0;
            margin: 0;
            background-color: transparent;
            border: 0;
            display: inline;
            font-size: 100%%;
        }
        
        blockquote {
            padding: 0 1em;
            color: #6a737d;
            border-left: 0.25em solid var(--border-color);
            margin: 0 0 16px 0;
        }
        
        ul, ol {
            padding-left: 2em;
            margin-top: 0;
            margin-bottom: 16px;
        }
        
        li + li {
            margin-top: 0.25em;
        }
        
        img {
            max-width: 100%%;
            height: auto;
            box-sizing: content-box;
            background-color: var(--bg-color);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            padding: 8px;
            margin: 16px 0;
        }
        
        table {
            border-spacing: 0;
            border-collapse: collapse;
            display: block;
            width: max-content;
            max-width: 100%%;
            overflow: auto;
            margin-bottom: 16px;
        }
        
        table th {
            font-weight: 600;
            padding: 6px 13px;
            border: 1px solid var(--border-color);
            background-color: var(--code-bg);
        }
        
        table td {
            padding: 6px 13px;
            border: 1px solid var(--border-color);
        }
        
        table tr {
            background-color: var(--bg-color);
            border-top: 1px solid var(--border-color);
        }
        
        hr {
            height: 0.25em;
            padding: 0;
            margin: 24px 0;
            background-color: var(--border-color);
            border: 0;
        }
        
        /* Task lists */
        input[type="checkbox"] {
            margin-right: 0.5em;
        }
        
        /* Strikethrough */
        del {
            text-decoration: line-through;
        }
    </style>
</head>
<body>
%s
</body>
</html>`, title, basePath, content)
}

// openInBrowser opens the file in the default browser using OS-specific commands
func (p *PreviewService) openInBrowser(filepath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use 'start' command on Windows
		cmd = exec.Command("cmd", "/c", "start", "", filepath)
	case "darwin":
		// Use 'open' command on macOS
		cmd = exec.Command("open", filepath)
	case "linux":
		// Use 'xdg-open' on Linux
		cmd = exec.Command("xdg-open", filepath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
