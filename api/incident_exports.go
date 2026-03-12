package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"
)

type IncidentTimelineExport struct {
	GeneratedAt string   `json:"generated_at"`
	Incident    Incident `json:"incident"`
}

type incidentExportLine struct {
	Text string
	Size int
}

func (s *Store) IncidentTimelineExport(id string) (IncidentTimelineExport, bool) {
	incidentID := strings.TrimSpace(id)
	if incidentID == "" {
		return IncidentTimelineExport{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, incident := range s.Incidents {
		if strings.TrimSpace(incident.ID) != incidentID {
			continue
		}
		return IncidentTimelineExport{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Incident:    cloneIncident(incident),
		}, true
	}

	return IncidentTimelineExport{}, false
}

func BuildIncidentTimelineMarkdown(doc IncidentTimelineExport) string {
	incident := doc.Incident
	lines := []string{
		"# NOCWALL Incident Timeline Export",
		"",
		"- Generated At: " + exportValueOrFallback(doc.GeneratedAt, "--"),
		"- Incident ID: " + exportValueOrFallback(incident.ID, "--"),
		"- Device ID: " + exportValueOrFallback(incident.DeviceID, "--"),
		"- Type: " + exportValueOrFallback(incident.Type, "--"),
		"- Severity: " + exportValueOrFallback(incident.Severity, "--"),
		"- Status: " + incidentStatusLabel(incident),
		"- Source: " + exportValueOrFallback(incident.Source, "--"),
		"- Started At: " + exportValueOrFallback(incident.Started, "--"),
		"- Resolved At: " + exportValueOrFallback(stringPtrOrEmpty(incident.Resolved), "--"),
		"- Acknowledged Until: " + exportValueOrFallback(stringPtrOrEmpty(incident.AckUntil), "--"),
		"- Commander: " + exportValueOrFallback(incident.Commander, "unassigned"),
		"",
	}

	message := strings.TrimSpace(incident.Message)
	if message != "" {
		lines = append(lines,
			"## Summary",
			"",
			message,
			"",
		)
	}

	lines = append(lines, "## Timeline", "")
	if len(incident.CommandTimeline) == 0 {
		lines = append(lines, "No timeline entries recorded.")
		return strings.Join(lines, "\n") + "\n"
	}

	for _, entry := range incident.CommandTimeline {
		at := exportValueOrFallback(entry.At, "--")
		eventType := exportValueOrFallback(entry.EventType, "incident_event")
		actor := exportValueOrFallback(entry.Actor, "system")
		text := strings.TrimSpace(entry.Message)
		if text == "" {
			text = defaultIncidentTimelineMessage(eventType)
		}
		lines = append(lines, fmt.Sprintf("1. `%s` `%s` `%s`", at, eventType, actor))
		lines = append(lines, "   "+text)
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func BuildIncidentTimelinePDF(doc IncidentTimelineExport) []byte {
	lines := buildIncidentExportLines(doc)
	pages := buildIncidentExportPages(lines)
	return renderMinimalPDF(pages)
}

func IncidentTimelineExportFilename(incident Incident, format string) string {
	stem := sanitizeExportFilenamePart(strings.TrimSpace(incident.ID))
	if stem == "" {
		stem = sanitizeExportFilenamePart(strings.TrimSpace(incident.DeviceID))
	}
	if stem == "" {
		stem = "incident"
	}
	suffix := "txt"
	switch normalizeIncidentExportFormat(format) {
	case "markdown":
		suffix = "md"
	case "pdf":
		suffix = "pdf"
	}
	return "nocwall-incident-" + stem + "-timeline." + suffix
}

func normalizeIncidentExportFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "markdown", "md":
		return "markdown"
	case "pdf":
		return "pdf"
	default:
		return ""
	}
}

func buildIncidentExportLines(doc IncidentTimelineExport) []incidentExportLine {
	incident := doc.Incident
	lines := []incidentExportLine{
		{Text: "NOCWALL Incident Timeline Export", Size: 18},
		{Text: "Generated At: " + exportValueOrFallback(doc.GeneratedAt, "--"), Size: 10},
		{Text: "", Size: 10},
		{Text: "Incident Summary", Size: 13},
		{Text: "Incident ID: " + exportValueOrFallback(incident.ID, "--"), Size: 10},
		{Text: "Device ID: " + exportValueOrFallback(incident.DeviceID, "--"), Size: 10},
		{Text: "Type: " + exportValueOrFallback(incident.Type, "--"), Size: 10},
		{Text: "Severity: " + exportValueOrFallback(incident.Severity, "--"), Size: 10},
		{Text: "Status: " + incidentStatusLabel(incident), Size: 10},
		{Text: "Source: " + exportValueOrFallback(incident.Source, "--"), Size: 10},
		{Text: "Started At: " + exportValueOrFallback(incident.Started, "--"), Size: 10},
		{Text: "Resolved At: " + exportValueOrFallback(stringPtrOrEmpty(incident.Resolved), "--"), Size: 10},
		{Text: "Acknowledged Until: " + exportValueOrFallback(stringPtrOrEmpty(incident.AckUntil), "--"), Size: 10},
		{Text: "Commander: " + exportValueOrFallback(incident.Commander, "unassigned"), Size: 10},
	}

	if message := strings.TrimSpace(incident.Message); message != "" {
		lines = append(lines,
			incidentExportLine{Text: "", Size: 10},
			incidentExportLine{Text: "Summary", Size: 13},
		)
		lines = appendWrappedIncidentExportLines(lines, message, 10, 92)
	}

	lines = append(lines,
		incidentExportLine{Text: "", Size: 10},
		incidentExportLine{Text: "Timeline", Size: 13},
	)

	if len(incident.CommandTimeline) == 0 {
		lines = append(lines, incidentExportLine{Text: "No timeline entries recorded.", Size: 10})
		return lines
	}

	for idx, entry := range incident.CommandTimeline {
		at := exportValueOrFallback(entry.At, "--")
		eventType := exportValueOrFallback(entry.EventType, "incident_event")
		actor := exportValueOrFallback(entry.Actor, "system")
		message := strings.TrimSpace(entry.Message)
		if message == "" {
			message = defaultIncidentTimelineMessage(eventType)
		}
		header := fmt.Sprintf("%d. %s | %s | %s", idx+1, at, eventType, actor)
		lines = appendWrappedIncidentExportLines(lines, header, 10, 90)
		lines = appendWrappedIncidentExportLines(lines, "   "+message, 10, 90)
		lines = append(lines, incidentExportLine{Text: "", Size: 10})
	}

	return lines
}

func appendWrappedIncidentExportLines(lines []incidentExportLine, text string, size, width int) []incidentExportLine {
	for _, line := range wrapIncidentExportText(text, width) {
		lines = append(lines, incidentExportLine{Text: line, Size: size})
	}
	return lines
}

func wrapIncidentExportText(text string, width int) []string {
	normalized := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"))
	if normalized == "" {
		return []string{""}
	}
	if width <= 0 {
		width = 88
	}

	paragraphs := strings.Split(normalized, "\n")
	out := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		trimmed := strings.TrimSpace(paragraph)
		if trimmed == "" {
			out = append(out, "")
			continue
		}
		words := strings.Fields(trimmed)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		current := words[0]
		for _, word := range words[1:] {
			candidate := current + " " + word
			if len(candidate) <= width {
				current = candidate
				continue
			}
			out = append(out, current)
			current = word
		}
		out = append(out, current)
	}
	return out
}

func buildIncidentExportPages(lines []incidentExportLine) []string {
	const (
		pageHeight = 792.0
		topMargin  = 48.0
		botMargin  = 48.0
		leftMargin = 48.0
	)

	pages := []string{}
	var page strings.Builder
	y := pageHeight - topMargin
	writeLine := func(text string, size int) {
		escaped := escapePDFText(text)
		if escaped == "" {
			return
		}
		page.WriteString(fmt.Sprintf("BT /F1 %d Tf %.2f %.2f Td (%s) Tj ET\n", size, leftMargin, y, escaped))
	}

	for _, line := range lines {
		size := line.Size
		if size <= 0 {
			size = 10
		}
		lineHeight := incidentExportLineHeight(size)
		if y-lineHeight < botMargin {
			pages = append(pages, page.String())
			page.Reset()
			y = pageHeight - topMargin
		}
		if strings.TrimSpace(line.Text) != "" {
			writeLine(line.Text, size)
		}
		y -= lineHeight
	}

	if page.Len() > 0 || len(pages) == 0 {
		pages = append(pages, page.String())
	}
	return pages
}

func incidentExportLineHeight(size int) float64 {
	switch {
	case size >= 18:
		return 24
	case size >= 13:
		return 18
	default:
		return 14
	}
}

func renderMinimalPDF(pages []string) []byte {
	if len(pages) == 0 {
		pages = []string{""}
	}

	type pdfObject struct {
		Number  int
		Payload string
	}

	objects := []pdfObject{
		{Number: 1, Payload: "<< /Type /Catalog /Pages 2 0 R >>"},
		{Number: 2, Payload: ""},
		{Number: 3, Payload: "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"},
	}

	pageRefs := make([]string, 0, len(pages))
	pageObjectNumbers := make([]int, 0, len(pages))
	contentObjectNumbers := make([]int, 0, len(pages))
	nextNumber := 4
	for range pages {
		pageObjectNumbers = append(pageObjectNumbers, nextNumber)
		contentObjectNumbers = append(contentObjectNumbers, nextNumber+1)
		pageRefs = append(pageRefs, fmt.Sprintf("%d 0 R", nextNumber))
		nextNumber += 2
	}

	objects[1].Payload = fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(pageRefs), strings.Join(pageRefs, " "))

	for idx, content := range pages {
		pageObject := fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", contentObjectNumbers[idx])
		contentBytes := []byte(content)
		contentObject := fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(contentBytes), content)
		objects = append(objects,
			pdfObject{Number: pageObjectNumbers[idx], Payload: pageObject},
			pdfObject{Number: contentObjectNumbers[idx], Payload: contentObject},
		)
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Number < objects[j].Number
	})

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	out.Write([]byte("%\xe2\xe3\xcf\xd3\n"))

	offsets := make([]int, nextNumber)
	for _, object := range objects {
		offsets[object.Number] = out.Len()
		out.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", object.Number, object.Payload))
	}

	xrefOffset := out.Len()
	out.WriteString(fmt.Sprintf("xref\n0 %d\n", nextNumber))
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i < nextNumber; i++ {
		out.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	out.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", nextNumber, xrefOffset))
	return out.Bytes()
}

func escapePDFText(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range text {
		switch r {
		case '\\', '(', ')':
			builder.WriteByte('\\')
			builder.WriteRune(r)
		case '\n', '\r', '\t':
			builder.WriteByte(' ')
		default:
			if r < 32 {
				continue
			}
			if r > 255 {
				builder.WriteByte('?')
				continue
			}
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func incidentStatusLabel(incident Incident) string {
	if incident.Resolved != nil && strings.TrimSpace(*incident.Resolved) != "" {
		return "resolved"
	}
	return "active"
}

func stringPtrOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func exportValueOrFallback(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return fallback
}

func sanitizeExportFilenamePart(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range trimmed {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}
