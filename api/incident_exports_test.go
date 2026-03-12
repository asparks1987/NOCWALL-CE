package main

import (
	"strings"
	"testing"
)

func TestIncidentTimelineExportBuildsMarkdownAndPDF(t *testing.T) {
	s := LoadStore("")
	s.mu.Lock()
	s.Devices = nil
	s.Incidents = nil
	s.mu.Unlock()

	offline := false
	_, created, ok := s.IngestTelemetry(TelemetryIngestRequest{
		Source:    "incident_export_test",
		EventType: "offline",
		DeviceID:  "export-node-1",
		Device:    "Export Node 1",
		Role:      "gateway",
		SiteID:    "export-site",
		Online:    &offline,
		Message:   "primary uplink offline",
	})
	if !ok || created == nil {
		t.Fatalf("expected incident creation for export test")
	}

	incidentID := created.ID
	if _, ok := s.SetIncidentCommander(incidentID, "alice", "alice"); !ok {
		t.Fatalf("expected commander assignment")
	}
	if _, ok := s.AddIncidentTimelineEntry(incidentID, "note", "waiting on field dispatch", "alice"); !ok {
		t.Fatalf("expected timeline note append")
	}

	doc, ok := s.IncidentTimelineExport(incidentID)
	if !ok {
		t.Fatalf("expected export document for incident")
	}
	if doc.Incident.ID != incidentID {
		t.Fatalf("expected incident id=%s, got=%s", incidentID, doc.Incident.ID)
	}

	markdown := BuildIncidentTimelineMarkdown(doc)
	for _, expected := range []string{
		"# NOCWALL Incident Timeline Export",
		"Incident ID: " + incidentID,
		"Commander: alice",
		"waiting on field dispatch",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected markdown export to contain %q", expected)
		}
	}

	pdf := BuildIncidentTimelinePDF(doc)
	if len(pdf) == 0 {
		t.Fatalf("expected non-empty pdf export")
	}
	if !strings.HasPrefix(string(pdf), "%PDF-1.4") {
		t.Fatalf("expected pdf header, got=%q", string(pdf[:min(8, len(pdf))]))
	}
	if !strings.Contains(string(pdf), "NOCWALL Incident Timeline Export") {
		t.Fatalf("expected pdf content stream to include export title")
	}
	if !strings.Contains(string(pdf), "waiting on field dispatch") {
		t.Fatalf("expected pdf content stream to include timeline note")
	}
}

func TestIncidentTimelineExportFilenameNormalizesStemAndFormat(t *testing.T) {
	name := IncidentTimelineExportFilename(Incident{ID: "INC 42/Main"}, "markdown")
	if name != "nocwall-incident-inc-42main-timeline.md" {
		t.Fatalf("unexpected markdown filename: %s", name)
	}

	pdfName := IncidentTimelineExportFilename(Incident{DeviceID: "Edge Switch 7"}, "pdf")
	if pdfName != "nocwall-incident-edge-switch-7-timeline.pdf" {
		t.Fatalf("unexpected pdf filename: %s", pdfName)
	}
}
