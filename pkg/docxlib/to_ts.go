// Package docxlib provides DOCX to TypeScript conversion utilities.
package docxlib

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/gomutex/godocx/dml"
	"github.com/gomutex/godocx/docx"
	"github.com/gomutex/godocx/wml/ctypes"
)

// ToTypeScript converts a godocx RootDoc to a TypeScript source string.
func ToTypeScript(rootDoc *docx.RootDoc, imagesDir string) string {
	if rootDoc == nil || rootDoc.Document == nil {
		return "// Empty or invalid document\n"
	}

	var b strings.Builder
	b.WriteString(importsHeader())

	var sectionChildren []string
	if rootDoc.Document.Body != nil {
		for _, child := range rootDoc.Document.Body.Children {
			if child.Para != nil {
				sectionChildren = append(sectionChildren, paragraphToTS(child.Para.GetCT(), rootDoc))
			} else if child.Table != nil {
				sectionChildren = append(sectionChildren, tableToTS(unsafeGetTableCT(child.Table), rootDoc))
			}
		}
	}

	headers := extractHeaderFooterTS(rootDoc, true)
	footers := extractHeaderFooterTS(rootDoc, false)

	b.WriteString("const doc = new Document({\n")
	b.WriteString("  sections: [\n")
	b.WriteString("    {\n")

	if len(headers) > 0 {
		b.WriteString("      headers: {\n")
		for k, v := range headers {
			b.WriteString(fmt.Sprintf("        %s: %s,\n", k, v))
		}
		b.WriteString("      },\n")
	}
	if len(footers) > 0 {
		b.WriteString("      footers: {\n")
		for k, v := range footers {
			b.WriteString(fmt.Sprintf("        %s: %s,\n", k, v))
		}
		b.WriteString("      },\n")
	}

	b.WriteString("      children: [\n")
	for _, child := range sectionChildren {
		b.WriteString(indent(child, 8) + ",\n")
	}
	b.WriteString("      ],\n")
	b.WriteString("    },\n")
	b.WriteString("  ],\n")
	b.WriteString("});\n\n")
	b.WriteString(exportFooter())
	return b.String()
}

func importsHeader() string {
	return `import * as fs from "fs";
import {
  Document,
  Packer,
  Paragraph,
  TextRun,
  Table,
  TableCell,
  TableRow,
  Header,
  Footer,
  ImageRun,
  AlignmentType,
  HeadingLevel,
  BorderStyle,
  WidthType,
  PageOrientation,
  convertInchesToTwip,
  convertMillimetersToTwip,
  ShadingType,
  VerticalAlign,
} from "docx";

`
}

func exportFooter() string {
	return `async function generate() {
  const buffer = await Packer.toBuffer(doc);
  fs.writeFileSync("output.docx", buffer);
  console.log("Document generated: output.docx");
}

generate().catch(console.error);
`
}

func extractHeaderFooterTS(rootDoc *docx.RootDoc, isHeader bool) map[string]string {
	result := make(map[string]string)
	if rootDoc == nil {
		return result
	}

	hasRef := false
	if rootDoc.Document.Body != nil && rootDoc.Document.Body.SectPr != nil {
		sp := rootDoc.Document.Body.SectPr
		if isHeader && sp.HeaderReference != nil && sp.HeaderReference.ID != "" {
			hasRef = true
		}
		if !isHeader && sp.FooterReference != nil && sp.FooterReference.ID != "" {
			hasRef = true
		}
	}
	if !hasRef {
		return result
	}

	kind := "word/header"
	if !isHeader {
		kind = "word/footer"
	}

	rootDoc.FileMap.Range(func(key, value any) bool {
		path := key.(string)
		if strings.Contains(path, kind) && strings.HasSuffix(path, ".xml") {
			content := value.([]byte)
			paras := parseXMLParagraphs(content, rootDoc)
			if len(paras) > 0 {
				var children []string
				for _, p := range paras {
					children = append(children, paragraphToTS(p, rootDoc))
				}
				className := "Header"
				if !isHeader {
					className = "Footer"
				}
				tsObj := fmt.Sprintf("new %s({ children: [\n%s\n      ] })",
					className,
					indent(strings.Join(children, ",\n"), 8))
				result["default"] = tsObj
			}
		}
		return true
	})

	return result
}

func paragraphToTS(para *ctypes.Paragraph, rootDoc *docx.RootDoc) string {
	if para == nil {
		return `new Paragraph("")`
	}

	var props []string
	ppr := para.Property

	if ppr != nil && ppr.PageBreakBefore != nil && ppr.PageBreakBefore.Val != nil && *ppr.PageBreakBefore.Val == "true" {
		props = append(props, "pageBreakBefore: true")
	}
	if ppr != nil && ppr.Justification != nil && ppr.Justification.Val != "" {
		if al := alignmentToTS(string(ppr.Justification.Val)); al != "" {
			props = append(props, fmt.Sprintf("alignment: %s", al))
		}
	}
	if ppr != nil && ppr.Style != nil && ppr.Style.Val != "" {
		if hl := headingLevelToTS(ppr.Style.Val); hl != "" {
			props = append(props, fmt.Sprintf("heading: %s", hl))
		}
	}
	if ppr != nil && ppr.Spacing != nil {
		if sp := spacingToTS(ppr.Spacing); sp != "" {
			props = append(props, fmt.Sprintf("spacing: %s", sp))
		}
	}
	if ppr != nil && ppr.Indent != nil {
		if ind := indentToTS(ppr.Indent); ind != "" {
			props = append(props, fmt.Sprintf("indent: %s", ind))
		}
	}

	var childExprs []string
	for _, child := range para.Children {
		if child.Run != nil {
			if expr := runToTS(child.Run, rootDoc); expr != "" {
				childExprs = append(childExprs, expr)
			}
		}
	}

	if len(childExprs) == 0 && len(props) == 0 {
		return `new Paragraph("")`
	}

	var parts []string
	if len(childExprs) > 0 {
		parts = append(parts, fmt.Sprintf("children: [\n%s\n      ]", indent(strings.Join(childExprs, ",\n"), 8)))
	}
	parts = append(parts, props...)

	return fmt.Sprintf("new Paragraph({\n%s\n    })", indent(strings.Join(parts, ",\n"), 4))
}

func runToTS(run *ctypes.Run, rootDoc *docx.RootDoc) string {
	if run == nil {
		return ""
	}
	var texts []string
	var props []string
	var imageExpr string

	for _, rc := range run.Children {
		if rc.Text != nil && rc.Text.Text != "" {
			texts = append(texts, rc.Text.Text)
		}
		if rc.Drawing != nil && rootDoc != nil {
			if img := drawingToImageTS(rc.Drawing, rootDoc); img != "" {
				imageExpr = img
			}
		}
	}

	content := strings.Join(texts, "")
	if content == "" && imageExpr == "" {
		return ""
	}

	if content != "" {
		props = append(props, fmt.Sprintf("text: %s", quote(content)))
	}
	if run.Property != nil {
		rp := run.Property
		if rp.Bold != nil && rp.Bold.Val != nil && *rp.Bold.Val == "true" {
			props = append(props, "bold: true")
		}
		if rp.Italic != nil && rp.Italic.Val != nil && *rp.Italic.Val == "true" {
			props = append(props, "italics: true")
		}
		if rp.Strike != nil && rp.Strike.Val != nil && *rp.Strike.Val == "true" {
			props = append(props, "strike: true")
		}
		if rp.DoubleStrike != nil && rp.DoubleStrike.Val != nil && *rp.DoubleStrike.Val == "true" {
			props = append(props, "doubleStrike: true")
		}
		if rp.Caps != nil && rp.Caps.Val != nil && *rp.Caps.Val == "true" {
			props = append(props, "allCaps: true")
		}
		if rp.SmallCaps != nil && rp.SmallCaps.Val != nil && *rp.SmallCaps.Val == "true" {
			props = append(props, "smallCaps: true")
		}
		if rp.Color != nil && rp.Color.Val != "" {
			props = append(props, fmt.Sprintf("color: %s", quote(rp.Color.Val)))
		}
		if rp.Size != nil {
			props = append(props, fmt.Sprintf("size: %d", rp.Size.Value))
		}
		if rp.Fonts != nil && rp.Fonts.Ascii != "" {
			props = append(props, fmt.Sprintf("font: %s", quote(rp.Fonts.Ascii)))
		}
		if rp.Underline != nil && rp.Underline.Val != "" {
			props = append(props, fmt.Sprintf("underline: { type: %s }", quote(string(rp.Underline.Val))))
		}
		if rp.Highlight != nil && rp.Highlight.Val != "" {
			props = append(props, fmt.Sprintf("highlight: %s", quote(rp.Highlight.Val)))
		}
		if rp.Shading != nil && rp.Shading.Fill != nil && *rp.Shading.Fill != "" {
			props = append(props, fmt.Sprintf("shading: { type: ShadingType.CLEAR, fill: %s }", quote(*rp.Shading.Fill)))
		}
		if rp.VertAlign != nil && rp.VertAlign.Val != "" {
			va := string(rp.VertAlign.Val)
			if va == "subscript" {
				props = append(props, "subScript: true")
			} else if va == "superscript" {
				props = append(props, "superScript: true")
			}
		}
	}

	var exprs []string
	if content != "" {
		exprs = append(exprs, fmt.Sprintf("new TextRun({ %s })", strings.Join(props, ", ")))
	}
	if imageExpr != "" {
		exprs = append(exprs, imageExpr)
	}
	if len(exprs) == 1 {
		return exprs[0]
	}
	return strings.Join(exprs, ",\n")
}

func drawingToImageTS(drawing *dml.Drawing, rootDoc *docx.RootDoc) string {
	b64 := getImageBase64FromDrawing(drawing, rootDoc)
	if b64 == "" {
		return ""
	}
	return fmt.Sprintf("new ImageRun({ data: Buffer.from(%s, \"base64\"), transformation: { width: 200, height: 200 } })", quote(b64))
}

func tableToTS(tbl *ctypes.Table, rootDoc *docx.RootDoc) string {
	if tbl == nil {
		return "new Table({ rows: [] })"
	}

	var rows []string
	for _, rc := range tbl.RowContents {
		if rc.Row == nil {
			continue
		}
		if expr := tableRowToTS(rc.Row, rootDoc); expr != "" {
			rows = append(rows, expr)
		}
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("rows: [\n%s\n      ]", indent(strings.Join(rows, ",\n"), 8)))

	if tbl.TableProp.Width != nil && tbl.TableProp.Width.Width != nil && tbl.TableProp.Width.WidthType != nil {
		tw := *tbl.TableProp.Width.Width
		typ := string(*tbl.TableProp.Width.WidthType)
		if typ == "pct" {
			parts = append(parts, fmt.Sprintf("width: { size: %d, type: WidthType.PERCENTAGE }", tw/50))
		} else if typ == "dxa" {
			parts = append(parts, fmt.Sprintf("width: { size: %d, type: WidthType.DXA }", tw))
		} else {
			parts = append(parts, fmt.Sprintf("width: { size: %d, type: WidthType.AUTO }", tw))
		}
	}

	if tbl.TableProp.Borders != nil {
		if bts := bordersToTS(tbl.TableProp.Borders); bts != "" {
			parts = append(parts, fmt.Sprintf("borders: %s", bts))
		}
	}

	return fmt.Sprintf("new Table({\n%s\n    })", indent(strings.Join(parts, ",\n"), 4))
}

func tableRowToTS(row *ctypes.Row, rootDoc *docx.RootDoc) string {
	if row == nil {
		return ""
	}
	var cells []string
	for _, cc := range row.Contents {
		if cc.Cell == nil {
			continue
		}
		if expr := tableCellToTS(cc.Cell, rootDoc); expr != "" {
			cells = append(cells, expr)
		}
	}
	if len(cells) == 0 {
		return ""
	}
	return fmt.Sprintf("new TableRow({\n%s\n      })", indent(fmt.Sprintf("children: [\n%s\n      ]", indent(strings.Join(cells, ",\n"), 8)), 4))
}

func tableCellToTS(cell *ctypes.Cell, rootDoc *docx.RootDoc) string {
	if cell == nil {
		return ""
	}
	var children []string
	for _, block := range cell.Contents {
		if block.Paragraph != nil {
			children = append(children, paragraphToTS(block.Paragraph, rootDoc))
		}
	}
	if len(children) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("children: [\n%s\n      ]", indent(strings.Join(children, ",\n"), 8)))

	if cell.Property != nil {
		cp := cell.Property
		if cp.Width != nil && cp.Width.Width != nil {
			parts = append(parts, fmt.Sprintf("width: { size: %d, type: WidthType.DXA }", *cp.Width.Width))
		}
		if cp.GridSpan != nil && cp.GridSpan.Val > 0 {
			parts = append(parts, fmt.Sprintf("columnSpan: %d", cp.GridSpan.Val))
		}
		if cp.Shading != nil && cp.Shading.Fill != nil && *cp.Shading.Fill != "" {
			parts = append(parts, fmt.Sprintf("shading: { fill: %s, type: ShadingType.CLEAR }", quote(*cp.Shading.Fill)))
		}
		if cp.VAlign != nil && cp.VAlign.Val != "" {
			va := string(cp.VAlign.Val)
			if va == "center" {
				parts = append(parts, "verticalAlign: VerticalAlign.CENTER")
			} else if va == "top" {
				parts = append(parts, "verticalAlign: VerticalAlign.TOP")
			} else if va == "bottom" {
				parts = append(parts, "verticalAlign: VerticalAlign.BOTTOM")
			}
		}
		if cp.Borders != nil {
			if bts := cellBordersToTS(cp.Borders); bts != "" {
				parts = append(parts, fmt.Sprintf("borders: %s", bts))
			}
		}
	}

	return fmt.Sprintf("new TableCell({\n%s\n      })", indent(strings.Join(parts, ",\n"), 4))
}

func bordersToTS(b *ctypes.TableBorders) string {
	if b == nil {
		return ""
	}
	var parts []string
	if b.Top != nil {
		parts = append(parts, fmt.Sprintf("top: %s", borderToTS(b.Top)))
	}
	if b.Bottom != nil {
		parts = append(parts, fmt.Sprintf("bottom: %s", borderToTS(b.Bottom)))
	}
	if b.Left != nil {
		parts = append(parts, fmt.Sprintf("left: %s", borderToTS(b.Left)))
	}
	if b.Right != nil {
		parts = append(parts, fmt.Sprintf("right: %s", borderToTS(b.Right)))
	}
	if b.InsideH != nil {
		parts = append(parts, fmt.Sprintf("insideHorizontal: %s", borderToTS(b.InsideH)))
	}
	if b.InsideV != nil {
		parts = append(parts, fmt.Sprintf("insideVertical: %s", borderToTS(b.InsideV)))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("{\n%s\n      }", indent(strings.Join(parts, ",\n"), 8))
}

func cellBordersToTS(b *ctypes.CellBorders) string {
	if b == nil {
		return ""
	}
	var parts []string
	if b.Top != nil {
		parts = append(parts, fmt.Sprintf("top: %s", borderToTS(b.Top)))
	}
	if b.Bottom != nil {
		parts = append(parts, fmt.Sprintf("bottom: %s", borderToTS(b.Bottom)))
	}
	if b.Left != nil {
		parts = append(parts, fmt.Sprintf("left: %s", borderToTS(b.Left)))
	}
	if b.Right != nil {
		parts = append(parts, fmt.Sprintf("right: %s", borderToTS(b.Right)))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("{\n%s\n      }", indent(strings.Join(parts, ",\n"), 8))
}

func borderToTS(b *ctypes.Border) string {
	if b == nil {
		return ""
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("style: BorderStyle.%s", borderStyleToTS(string(b.Val))))
	if b.Color != nil && *b.Color != "" {
		parts = append(parts, fmt.Sprintf("color: %s", quote(*b.Color)))
	}
	if b.Space != nil && *b.Space != "" {
		parts = append(parts, fmt.Sprintf("space: %s", *b.Space))
	}
	return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
}

func borderStyleToTS(val string) string {
	switch val {
	case "single":
		return "SINGLE"
	case "double":
		return "DOUBLE"
	case "dashed":
		return "DASHED"
	case "dotted":
		return "DOTTED"
	case "dotDash":
		return "DOT_DASH"
	case "dotDotDash":
		return "DOT_DOT_DASH"
	case "triple":
		return "TRIPLE"
	case "thinThickSmallGap":
		return "THIN_THICK_SMALL_GAP"
	case "thickThinSmallGap":
		return "THICK_THIN_SMALL_GAP"
	case "thinThickThinSmallGap":
		return "THIN_THICK_THIN_SMALL_GAP"
	case "thinThickMediumGap":
		return "THIN_THICK_MEDIUM_GAP"
	case "thickThinMediumGap":
		return "THICK_THIN_MEDIUM_GAP"
	case "thinThickThinMediumGap":
		return "THIN_THICK_THIN_MEDIUM_GAP"
	case "thinThickLargeGap":
		return "THIN_THICK_LARGE_GAP"
	case "thickThinLargeGap":
		return "THICK_THIN_LARGE_GAP"
	case "thinThickThinLargeGap":
		return "THIN_THICK_THIN_LARGE_GAP"
	case "wave":
		return "WAVE"
	case "doubleWave":
		return "DOUBLE_WAVE"
	case "none":
		return "NIL"
	default:
		return "SINGLE"
	}
}

func alignmentToTS(val string) string {
	switch val {
	case "left":
		return "AlignmentType.LEFT"
	case "center":
		return "AlignmentType.CENTER"
	case "right":
		return "AlignmentType.RIGHT"
	case "both":
		return "AlignmentType.JUSTIFIED"
	case "distribute":
		return "AlignmentType.DISTRIBUTED"
	}
	return ""
}

func headingLevelToTS(val string) string {
	switch val {
	case "Heading1":
		return "HeadingLevel.HEADING_1"
	case "Heading2":
		return "HeadingLevel.HEADING_2"
	case "Heading3":
		return "HeadingLevel.HEADING_3"
	case "Heading4":
		return "HeadingLevel.HEADING_4"
	case "Heading5":
		return "HeadingLevel.HEADING_5"
	case "Heading6":
		return "HeadingLevel.HEADING_6"
	}
	return ""
}

func spacingToTS(sp *ctypes.Spacing) string {
	var parts []string
	if sp.Before != nil {
		parts = append(parts, fmt.Sprintf("before: %d", *sp.Before))
	}
	if sp.After != nil {
		parts = append(parts, fmt.Sprintf("after: %d", *sp.After))
	}
	if sp.Line != nil {
		parts = append(parts, fmt.Sprintf("line: %d", *sp.Line))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
}

func indentToTS(ind *ctypes.Indent) string {
	var parts []string
	if ind.Left != nil {
		parts = append(parts, fmt.Sprintf("left: %d", *ind.Left))
	}
	if ind.Right != nil {
		parts = append(parts, fmt.Sprintf("right: %d", *ind.Right))
	}
	if ind.FirstLine != nil {
		parts = append(parts, fmt.Sprintf("firstLine: %d", *ind.FirstLine))
	}
	if ind.Hanging != nil {
		parts = append(parts, fmt.Sprintf("hanging: %d", *ind.Hanging))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
}

func quote(s string) string {
	return fmt.Sprintf("%q", s)
}

func indent(s string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = prefix + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func getImageBase64FromDrawing(drawing *dml.Drawing, rootDoc *docx.RootDoc) string {
	if drawing == nil || rootDoc == nil {
		return ""
	}

	var imgBytes []byte
	rootDoc.FileMap.Range(func(key, value any) bool {
		path := key.(string)
		if strings.Contains(path, "word/media/") {
			imgBytes = value.([]byte)
			return false
		}
		return true
	})

	if imgBytes == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(imgBytes)
}

// parseXMLParagraphs parses a header/footer XML byte slice and returns a slice of Paragraph pointers.
func parseXMLParagraphs(xmlContent []byte, rootDoc *docx.RootDoc) []*ctypes.Paragraph {
	var result []*ctypes.Paragraph
	if len(xmlContent) == 0 {
		return result
	}

	type wPara struct {
		XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main p"`
		ctypes.Paragraph
	}
	type wrapper struct {
		XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main hdr"`
		Paras   []wPara  `xml:"p"`
	}

	var w wrapper
	if err := xml.Unmarshal(xmlContent, &w); err == nil && len(w.Paras) > 0 {
		for _, p := range w.Paras {
			cp := p.Paragraph
			result = append(result, &cp)
		}
		return result
	}

	var fWrapper struct {
		XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main ftr"`
		Paras   []wPara  `xml:"p"`
	}
	if err := xml.Unmarshal(xmlContent, &fWrapper); err == nil && len(fWrapper.Paras) > 0 {
		for _, p := range fWrapper.Paras {
			cp := p.Paragraph
			result = append(result, &cp)
		}
		return result
	}

	contentStr := string(xmlContent)
	start := 0
	for {
		idx := strings.Index(contentStr[start:], "<w:p")
		if idx == -1 {
			break
		}
		idx += start
		endIdx := strings.Index(contentStr[idx:], "</w:p>")
		if endIdx == -1 {
			break
		}
		endIdx += idx + len("</w:p>")
		paraXML := contentStr[idx:endIdx]
		var p wPara
		if err := xml.Unmarshal([]byte(paraXML), &p); err == nil {
			cp := p.Paragraph
			result = append(result, &cp)
		}
		start = endIdx
	}

	return result
}
