package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// PDFService handles PDF generation for documents
type PDFService struct {
	Format      string
	Orientation string
	Unit        string
	FontFamily  string
	FontSize    float64
}

// NewPDFService creates a new PDF service instance
func NewPDFService() *PDFService {
	return &PDFService{
		Format:      "A4",
		Orientation: "P",
		Unit:        "mm",
		FontFamily:  "Arial",
		FontSize:    12,
	}
}

// DocumentPDFRequest represents the request data for PDF generation
type DocumentPDFRequest struct {
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	Status      string  `json:"status"`
	Version     float64 `json:"version"` // Back to float64 as the correct type
	DocNo       string  `json:"docNo"`
	Priority    string  `json:"priority"`
	Author      string  `json:"author"`
	CreatedDate string  `json:"createdDate"`
}

// GenerateDocumentPDF creates a PDF from document data
func (p *PDFService) GenerateDocumentPDF(docData DocumentPDFRequest) ([]byte, error) {
	// Create new PDF document
	pdf := gofpdf.New(p.Orientation, p.Unit, p.Format, "")

	// Set document properties
	pdf.SetTitle(docData.Title, true)
	pdf.SetAuthor(docData.Author, true)
	pdf.SetCreator("MDP System", true)

	// Add a page
	pdf.AddPage()

	// Add header
	p.addHeader(pdf, docData)

	// Add watermark based on status
	p.addWatermark(pdf, docData.Status)

	// Add content directly (preserving JSON structure for rich formatting)
	p.addContent(pdf, docData.Content)

	// Add footer
	p.addFooter(pdf, docData)

	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)

	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %v", err)
	}

	log.Printf("PDF generated successfully for document: %s (Status: %s)", docData.Title, docData.Status)
	return buf.Bytes(), nil
}

// addHeader adds the document header
func (p *PDFService) addHeader(pdf *gofpdf.Fpdf, docData DocumentPDFRequest) {
	// Header background
	pdf.SetFillColor(240, 248, 255)
	pdf.Rect(10, 10, 190, 30, "F")

	// MDP System Title
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(25, 25, 112)
	pdf.SetXY(15, 15)
	pdf.Cell(0, 8, "MDP System - Document Management")

	// Document Title
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(15, 25)
	title := docData.Title
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	pdf.Cell(0, 8, title)

	// Document Information (Right side)
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(100, 100, 100)

	pdf.SetXY(120, 15)
	pdf.Cell(0, 4, fmt.Sprintf("Doc No: %s", docData.DocNo))

	pdf.SetXY(120, 20)
	pdf.Cell(0, 4, fmt.Sprintf("Version: %s", docData.Version))

	pdf.SetXY(120, 25)
	statusColor := p.getStatusColor(docData.Status)
	pdf.SetTextColor(statusColor.R, statusColor.G, statusColor.B)
	pdf.Cell(0, 4, fmt.Sprintf("Status: %s", strings.ToUpper(docData.Status)))

	pdf.SetTextColor(100, 100, 100)
	pdf.SetXY(120, 30)
	pdf.Cell(0, 4, fmt.Sprintf("Priority: %s", docData.Priority))

	// Reset text color and position
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(50)
}

// addWatermark adds status-based watermark
func (p *PDFService) addWatermark(pdf *gofpdf.Fpdf, status string) {
	// Get current position to restore later
	x, y := pdf.GetXY()

	// Set watermark properties
	pdf.SetFont("Arial", "B", 60)

	// Determine watermark text and color
	watermarkText := ""
	switch strings.ToLower(status) {
	case "draft":
		watermarkText = "DRAFT"
		pdf.SetTextColor(255, 200, 200)
	case "approved":
		watermarkText = "APPROVED"
		pdf.SetTextColor(200, 255, 200)
	case "in review", "review":
		watermarkText = "UNDER REVIEW"
		pdf.SetTextColor(255, 255, 200)
	case "rejected":
		watermarkText = "REJECTED"
		pdf.SetTextColor(255, 180, 180)
	default:
		watermarkText = strings.ToUpper(status)
		pdf.SetTextColor(220, 220, 220)
	}

	if watermarkText != "" {
		// Position watermark in center of page
		pdf.SetXY(50, 150)

		// Rotate text for diagonal watermark effect
		pdf.TransformBegin()
		pdf.TransformRotate(45, 105, 150)
		pdf.Cell(0, 0, watermarkText)
		pdf.TransformEnd()
	}

	// Reset text properties and position
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 12)
	pdf.SetXY(x, y)
}

// addContent processes and adds document content with rich formatting
func (p *PDFService) addContent(pdf *gofpdf.Fpdf, rawContent string) {
	// Set content styling
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(50, 50, 50)

	// Add content section header
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "Document Content")
	pdf.Ln(10)

	// Reset font for content
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(50, 50, 50)

	// Check if content is JSON (from Lexical editor)
	if strings.TrimSpace(rawContent) != "" && strings.HasPrefix(strings.TrimSpace(rawContent), "{") {
		p.processLexicalJSON(pdf, rawContent)
	} else {
		// Fallback to plain text
		lines := strings.Split(rawContent, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				pdf.Ln(3)
				continue
			}
			p.addWrappedText(pdf, line, 180)
			pdf.Ln(5)
		}
	}
}

// processLexicalJSON processes Lexical editor JSON content
func (p *PDFService) processLexicalJSON(pdf *gofpdf.Fpdf, jsonContent string) {
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &doc); err != nil {
		log.Printf("Failed to parse JSON content: %v", err)
		pdf.Cell(0, 6, "Error: Unable to parse document content")
		return
	}

	// Process the document root
	if content, ok := doc["content"].([]interface{}); ok {
		for _, node := range content {
			p.processNode(pdf, node)
		}
	}
}

// processNode processes individual nodes in the document
func (p *PDFService) processNode(pdf *gofpdf.Fpdf, node interface{}) {
	nodeMap, ok := node.(map[string]interface{})
	if !ok {
		return
	}

	nodeType, ok := nodeMap["type"].(string)
	if !ok {
		return
	}

	switch nodeType {
	case "heading":
		p.processHeading(pdf, nodeMap)
	case "paragraph":
		p.processParagraph(pdf, nodeMap)
	case "table":
		p.processTable(pdf, nodeMap)
	default:
		// For unknown node types, try to process their content
		if content, ok := nodeMap["content"].([]interface{}); ok {
			for _, child := range content {
				p.processNode(pdf, child)
			}
		}
	}
}

// processHeading processes heading nodes
func (p *PDFService) processHeading(pdf *gofpdf.Fpdf, node map[string]interface{}) {
	// Get heading level
	level := 1
	if attrs, ok := node["attrs"].(map[string]interface{}); ok {
		if levelFloat, ok := attrs["level"].(float64); ok {
			level = int(levelFloat)
		}
	}

	// Set font size based on level
	fontSize := 16.0 - float64(level-1)*2
	if fontSize < 12 {
		fontSize = 12
	}

	pdf.Ln(3)
	pdf.SetFont("Arial", "B", fontSize)
	pdf.SetTextColor(0, 0, 0)

	// Process heading content
	if content, ok := node["content"].([]interface{}); ok {
		text := p.extractTextFromNodes(content)
		p.addWrappedText(pdf, text, 180)
	}

	pdf.Ln(5)
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(50, 50, 50)
}

// processParagraph processes paragraph nodes
func (p *PDFService) processParagraph(pdf *gofpdf.Fpdf, node map[string]interface{}) {
	if content, ok := node["content"].([]interface{}); ok {
		text := p.extractTextFromNodes(content)
		if strings.TrimSpace(text) != "" {
			p.addWrappedText(pdf, text, 180)
		}
	}
	pdf.Ln(5)
}

// processTable processes table nodes
func (p *PDFService) processTable(pdf *gofpdf.Fpdf, node map[string]interface{}) {
	pdf.Ln(5) // Space before table

	if content, ok := node["content"].([]interface{}); ok {
		// Extract table data
		var tableData [][]string

		for _, rowNode := range content {
			if rowMap, ok := rowNode.(map[string]interface{}); ok {
				if rowType, ok := rowMap["type"].(string); ok && rowType == "tableRow" {
					var rowData []string
					if rowContent, ok := rowMap["content"].([]interface{}); ok {
						for _, cellNode := range rowContent {
							if cellMap, ok := cellNode.(map[string]interface{}); ok {
								if cellType, ok := cellMap["type"].(string); ok && cellType == "tableCell" {
									cellText := ""
									if cellContent, ok := cellMap["content"].([]interface{}); ok {
										cellText = p.extractTextFromNodes(cellContent)
									}
									rowData = append(rowData, cellText)
								}
							}
						}
					}
					if len(rowData) > 0 {
						tableData = append(tableData, rowData)
					}
				}
			}
		}

		// Render table if we have data
		if len(tableData) > 0 {
			p.renderSimpleTable(pdf, tableData)
		}
	}

	pdf.Ln(5) // Space after table
}

// extractTextFromNodes extracts text from an array of content nodes
func (p *PDFService) extractTextFromNodes(nodes []interface{}) string {
	var result strings.Builder

	for _, node := range nodes {
		if nodeMap, ok := node.(map[string]interface{}); ok {
			if text, ok := nodeMap["text"].(string); ok {
				// Check for formatting marks
				isBold := false
				if marks, ok := nodeMap["marks"].([]interface{}); ok {
					for _, mark := range marks {
						if markMap, ok := mark.(map[string]interface{}); ok {
							if markType, ok := markMap["type"].(string); ok && markType == "bold" {
								isBold = true
								break
							}
						}
					}
				}

				// For now, we'll extract the text and note if it should be bold
				// The bold formatting will be handled during rendering
				if isBold {
					result.WriteString("**" + text + "**") // Mark bold text
				} else {
					result.WriteString(text)
				}
			} else if content, ok := nodeMap["content"].([]interface{}); ok {
				// Recursive call for nested content
				result.WriteString(p.extractTextFromNodes(content))
			}
		}
	}

	return result.String()
}

// renderSimpleTable renders a table with simple formatting
func (p *PDFService) renderSimpleTable(pdf *gofpdf.Fpdf, tableData [][]string) {
	if len(tableData) == 0 {
		return
	}

	// Calculate column width
	numCols := len(tableData[0])
	colWidth := 170.0 / float64(numCols) // Total width divided by columns

	pdf.SetFont("Arial", "", 10)
	pdf.SetDrawColor(0, 0, 0)

	for rowIndex, row := range tableData {
		// Calculate row height needed
		maxHeight := 10.0

		// Check if we need a new page
		if pdf.GetY() > 250 {
			pdf.AddPage()
		}

		startY := pdf.GetY()

		// Draw cells for this row
		for colIndex, cellText := range row {
			if colIndex >= numCols {
				break
			}

			cellX := 10.0 + float64(colIndex)*colWidth

			// Set font style for header row
			if rowIndex == 0 {
				pdf.SetFont("Arial", "B", 10)
				pdf.SetTextColor(0, 0, 0)
			} else {
				pdf.SetFont("Arial", "", 10)
				pdf.SetTextColor(50, 50, 50)
			}

			// Draw cell border
			pdf.Rect(cellX, startY, colWidth, maxHeight, "D")

			// Position text in cell
			pdf.SetXY(cellX+1, startY+2)

			// Clean text and render
			cleanText := strings.ReplaceAll(cellText, "**", "") // Remove bold markers
			cleanText = strings.TrimSpace(cleanText)

			// Simple text rendering (truncate if too long)
			if pdf.GetStringWidth(cleanText) > colWidth-2 {
				// Truncate text if too long
				for len(cleanText) > 0 && pdf.GetStringWidth(cleanText+"...") > colWidth-2 {
					cleanText = cleanText[:len(cleanText)-1]
				}
				cleanText += "..."
			}

			pdf.Cell(colWidth-2, 6, cleanText)
		}

		// Move to next row
		pdf.SetY(startY + maxHeight)
	}

	// Reset font
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(50, 50, 50)
}

// addWrappedText adds text with word wrapping
func (p *PDFService) addWrappedText(pdf *gofpdf.Fpdf, text string, maxWidth float64) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return
	}

	currentLine := ""
	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		lineWidth := pdf.GetStringWidth(testLine)
		if lineWidth <= maxWidth {
			currentLine = testLine
		} else {
			if currentLine != "" {
				pdf.Cell(0, 6, currentLine)
				pdf.Ln(6)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		pdf.Cell(0, 6, currentLine)
	}
}

// addFooter adds the document footer
func (p *PDFService) addFooter(pdf *gofpdf.Fpdf, docData DocumentPDFRequest) {
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)

		pdf.SetDrawColor(200, 200, 200)
		pdf.Line(10, pdf.GetY(), 200, pdf.GetY())

		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(100, 100, 100)

		pdf.SetX(10)
		footerLeft := fmt.Sprintf("Document: %s | Author: %s", docData.Title, docData.Author)
		if len(footerLeft) > 60 {
			footerLeft = footerLeft[:57] + "..."
		}
		pdf.Cell(0, 5, footerLeft)

		printDate := time.Now().Format("Printed on January 2, 2006 at 3:04 PM")
		pdf.CellFormat(190, 5, printDate, "", 0, "C", false, 0, "")

		pdf.SetX(-30)
		pdf.Cell(0, 5, fmt.Sprintf("Page %d", pdf.PageNo()))
	})
}

// getStatusColor returns color for status display
func (p *PDFService) getStatusColor(status string) struct{ R, G, B int } {
	switch strings.ToLower(status) {
	case "draft":
		return struct{ R, G, B int }{128, 128, 128}
	case "approved":
		return struct{ R, G, B int }{34, 139, 34}
	case "in review", "review":
		return struct{ R, G, B int }{255, 140, 0}
	case "rejected":
		return struct{ R, G, B int }{220, 20, 60}
	default:
		return struct{ R, G, B int }{0, 0, 0}
	}
}
