package ingestion

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

type ParsedDocument struct {
	Filename   string
	TotalPages int
	Pages      []ParsedPage
	RawText    string
}

type ParsedPage struct {
	PageNumber int
	Text       string
}

type PDFParser struct{}

var Verbose = true

func logInfo(format string, args ...any) {
	if Verbose {
		fmt.Printf(format, args...)
	}
}

func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

func (p *PDFParser) ParsePDF(pdfPath string) (*ParsedDocument, error) {
	// Check file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", pdfPath)
	}

	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	totalPages := r.NumPage()
	logInfo("📄 Parsing %s (%d pages)...\n", pdfPath, totalPages)

	doc := &ParsedDocument{
		Filename:   pdfPath,
		TotalPages: totalPages,
		Pages:      make([]ParsedPage, 0, totalPages),
	}

	var fullText bytes.Buffer

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			logInfo("  ⚠️  Warning: failed to extract page %d: %v\n", pageNum, err)
			continue
		}

		// Clean up whitespace
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		doc.Pages = append(doc.Pages, ParsedPage{
			PageNumber: pageNum,
			Text:       text,
		})

		fullText.WriteString(text)
		fullText.WriteString("\n\n")
	}

	doc.RawText = fullText.String()
	logInfo("✅ Parsed %d pages, extracted %d characters\n", len(doc.Pages), len(doc.RawText))

	return doc, nil
}
