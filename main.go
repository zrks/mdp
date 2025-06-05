package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

const (
	defaultTemplate = `<!DOCTYPE html><html><head><meta http-equiv="content-type" content="text/html; charset=utf-8"> <title>{{ .Title }}</title> </head> <body> {{ .Body }} </body> </html>`
)

type content struct {
	Title string
	Body  template.HTML
}

func main() {
	filename := flag.String("file", "", "Markdown file to preview")
	skipPreview := flag.Bool("s", false, "Skip auto-preview")
	templateFile := flag.String("t", "", "Alternative HTML template file")
	flag.Parse()

	if *filename == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := run(*filename, os.Stdout, *skipPreview, *templateFile); err != nil {
		log.Fatalf("preview failed: %v", err)
	}

}

// run reads the Markdown file at filename, renders it using templateFile,
// writes the result to index.html in the same directory, prints the output
// path to w, and optionally invokes a preview command.
func run(filename string, w io.Writer, skipPreview bool, templateFile string) error {
	// Read source Markdown
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading markdown %q: %w", filename, err)
	}

	// Render HTML from template
	html, err := parseContent(data, templateFile)
	if err != nil {
		return fmt.Errorf("rendering content from template %q: %w", templateFile, err)
	}

	// Determine output path: same dir as source, named index.html
	outDir := filepath.Dir(filename)
	outPath := filepath.Join(outDir, "index.html")

	// Inform caller where file will be written
	fmt.Fprintln(w, outPath)

	// Write out HTML file
	if err := saveHTML(outPath, html); err != nil {
		return fmt.Errorf("writing HTML to %q: %w", outPath, err)
	}

	// Skip preview if requested
	if skipPreview {
		return nil
	}

	// Preview the generated file
	if err := preview(outPath); err != nil {
		return fmt.Errorf("preview failed for %q: %w", outPath, err)
	}

	return nil
}

// markdownToHTML converts basic Markdown to HTML
func markdownToHTML(markdown []byte) []byte {
	// Convert line endings to \n
	content := bytes.ReplaceAll(markdown, []byte("\r\n"), []byte("\n"))
	content = bytes.ReplaceAll(content, []byte("\r"), []byte("\n"))

	// Split into lines
	lines := bytes.Split(content, []byte("\n"))
	var result []byte
	var inCodeBlock bool
	var inList bool

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Handle code blocks
		if bytes.HasPrefix(line, []byte("```")) {
			if inCodeBlock {
				result = append(result, []byte("</pre></code>\n")...)
			} else {
				result = append(result, []byte("<pre><code>")...)
			}
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			result = append(result, line...)
			result = append(result, '\n')
			continue
		}

		// Handle headers
		if bytes.HasPrefix(line, []byte("# ")) {
			result = append(result, []byte("<h1>")...)
			result = append(result, line[2:]...)
			result = append(result, []byte("</h1>\n")...)
			continue
		}

		// Handle lists
		if bytes.HasPrefix(line, []byte("- ")) {
			if !inList {
				result = append(result, []byte("<ul>\n")...)
				inList = true
			}
			result = append(result, []byte("<li>")...)
			result = append(result, line[2:]...)
			result = append(result, []byte("</li>\n")...)
			continue
		} else if inList {
			result = append(result, []byte("</ul>\n")...)
			inList = false
		}

		// Handle paragraphs
		if len(line) > 0 {
			result = append(result, []byte("<p>")...)
			result = append(result, line...)
			result = append(result, []byte("</p>\n")...)
		} else {
			result = append(result, '\n')
		}
	}

	if inList {
		result = append(result, []byte("</ul>\n")...)
	}

	return result
}

// sanitizeHTML removes potentially dangerous HTML elements and attributes
func sanitizeHTML(html []byte) []byte {
	// Convert to string for easier manipulation
	content := string(html)

	// Remove script tags and their contents
	scriptRegex := regexp.MustCompile(`<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")

	// Remove event handlers
	eventRegex := regexp.MustCompile(`\s(on\w+)="[^"]*"`)
	content = eventRegex.ReplaceAllString(content, "")

	// Remove javascript: URLs
	jsUrlRegex := regexp.MustCompile(`javascript:[^"'\s]*`)
	content = jsUrlRegex.ReplaceAllString(content, "")

	// Remove style attributes
	styleRegex := regexp.MustCompile(`\sstyle="[^"]*"`)
	content = styleRegex.ReplaceAllString(content, "")

	// Remove iframe tags
	iframeRegex := regexp.MustCompile(`<iframe[^>]*>.*?</iframe>`)
	content = iframeRegex.ReplaceAllString(content, "")

	// Remove object tags
	objectRegex := regexp.MustCompile(`<object[^>]*>.*?</object>`)
	content = objectRegex.ReplaceAllString(content, "")

	// Remove embed tags
	embedRegex := regexp.MustCompile(`<embed[^>]*>`)
	content = embedRegex.ReplaceAllString(content, "")

	// Remove base tags
	baseRegex := regexp.MustCompile(`<base[^>]*>`)
	content = baseRegex.ReplaceAllString(content, "")

	// Remove meta refresh
	metaRefreshRegex := regexp.MustCompile(`<meta[^>]*http-equiv=["']refresh["'][^>]*>`)
	content = metaRefreshRegex.ReplaceAllString(content, "")

	return []byte(content)
}

// parseContent converts the given Markdown input into sanitized HTML by applying
// either the built-in defaultTemplate or an optional external template file.
// It returns the rendered HTML bytes or an error.
func parseContent(markdown []byte, templateFileName string) ([]byte, error) {
	// 1. Convert Markdown → HTML
	rendered := markdownToHTML(markdown)

	// 2. Sanitize HTML for safe output
	sanitized := sanitizeHTML(rendered)

	// 3. Load the base template
	tmpl, err := template.New("mdp").Parse(defaultTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing default template: %w", err)
	}

	// 4. If a custom template is provided, override the base template
	if templateFileName != "" {
		tmpl, err = template.ParseFiles(templateFileName)
		if err != nil {
			return nil, fmt.Errorf("parsing template file %q: %w", templateFileName, err)
		}
	}

	// 5. Prepare data for the template
	data := content{
		Title: "zrks", // adjust as appropriate
		Body:  template.HTML(sanitized),
	}

	// 6. Execute the template into a buffer
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

func saveHTML(filename string, data []byte) error {
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("writing HTML file %q: %w", filename, err)
	}
	return nil
}

// preview opens the given file in the user's default viewer/browser,
// based on the host operating system.
// It locates the appropriate open command, runs it, and waits briefly
// to ensure the viewer has time to launch.
func preview(filePath string) error {
	// Select the appropriate command and initial arguments per OS
	var cmdName string
	var cmdArgs []string

	switch runtime.GOOS {
	case "linux":
		cmdName = "xdg-open"
	case "windows":
		cmdName = "cmd.exe"
		cmdArgs = []string{"/C", "start"}
	case "darwin":
		cmdName = "open"
	default:
		return fmt.Errorf("unsupported OS: %q", runtime.GOOS)
	}

	// Append the file to open
	cmdArgs = append(cmdArgs, filePath)

	// Resolve the full path to the executable
	exePath, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("executable %q not found: %w", cmdName, err)
	}

	// Execute the command
	cmd := exec.Command(exePath, cmdArgs...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %q with args %v: %w", cmdName, cmdArgs, err)
	}

	// Allow viewer time to start
	time.Sleep(2 * time.Second)

	return nil
}
