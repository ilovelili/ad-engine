package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
)

const (
	pageWidth    = 612
	pageHeight   = 792
	leftMargin   = 48
	topMargin    = 54
	lineHeight   = 14
	linesPerPage = 48
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: go run ./tools/whitepaper_pdf <input.md> <output.pdf>\n")
		os.Exit(1)
	}

	lines, err := readMarkdown(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read markdown: %v\n", err)
		os.Exit(1)
	}

	pdf, err := buildPDF(lines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build pdf: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(os.Args[2], pdf, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write pdf: %v\n", err)
		os.Exit(1)
	}
}

func readMarkdown(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := normalizeLine(scanner.Text())
		if line == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, wrapLine(line, 88)...)
	}

	return lines, scanner.Err()
}

func normalizeLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(line, "### "):
		line = strings.TrimPrefix(line, "### ")
		line = strings.ToUpper(line)
	case strings.HasPrefix(line, "## "):
		line = strings.TrimPrefix(line, "## ")
		line = strings.ToUpper(line)
	case strings.HasPrefix(line, "# "):
		line = strings.TrimPrefix(line, "# ")
		line = strings.ToUpper(line)
	case strings.HasPrefix(line, "- "):
		line = "• " + strings.TrimPrefix(line, "- ")
	case strings.HasPrefix(line, "1. "), strings.HasPrefix(line, "2. "), strings.HasPrefix(line, "3. "), strings.HasPrefix(line, "4. "), strings.HasPrefix(line, "5. "), strings.HasPrefix(line, "6. "), strings.HasPrefix(line, "7. "), strings.HasPrefix(line, "8. "), strings.HasPrefix(line, "9. "):
	default:
	}

	replacer := strings.NewReplacer(
		"`", "",
		"**", "",
		"###", "",
		"##", "",
		"#", "",
		"[", "",
		"](", " (",
		")", ")",
	)
	line = replacer.Replace(line)
	line = strings.ReplaceAll(line, "]( /", " (/")
	line = strings.ReplaceAll(line, "](/", " (/")
	line = strings.TrimSpace(line)
	return line
}

func wrapLine(line string, limit int) []string {
	if len(line) <= limit {
		return []string{line}
	}

	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}

	var wrapped []string
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) <= limit {
			current += " " + word
			continue
		}
		wrapped = append(wrapped, current)
		current = word
	}
	wrapped = append(wrapped, current)
	return wrapped
}

func buildPDF(lines []string) ([]byte, error) {
	pages := paginate(lines)

	var objects []string
	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")

	kids := make([]string, 0, len(pages))
	for i := range pages {
		pageObjNum := 4 + (i * 2)
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObjNum))
	}
	objects = append(objects, fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(kids), strings.Join(kids, " ")))
	objects = append(objects, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	for _, page := range pages {
		content := pageContent(page)
		pageObjNum := len(objects) + 1
		contentObjNum := pageObjNum + 1

		pageObject := fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", pageWidth, pageHeight, contentObjNum)
		contentObject := fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)

		objects = append(objects, pageObject, contentObject)
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xrefOffset)

	return buf.Bytes(), nil
}

func paginate(lines []string) [][]string {
	var pages [][]string
	var page []string

	for _, line := range lines {
		if len(page) == linesPerPage {
			pages = append(pages, page)
			page = nil
		}
		page = append(page, line)
	}

	if len(page) > 0 {
		pages = append(pages, page)
	}
	if len(pages) == 0 {
		pages = append(pages, []string{""})
	}
	return pages
}

func pageContent(lines []string) string {
	var buf strings.Builder
	y := pageHeight - topMargin
	buf.WriteString("BT\n/F1 11 Tf\n")
	for _, line := range lines {
		fmt.Fprintf(&buf, "1 0 0 1 %d %d Tm (%s) Tj\n", leftMargin, y, escapePDFText(line))
		y -= lineHeight
	}
	buf.WriteString("ET")
	return buf.String()
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
