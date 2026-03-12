package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUISPConnectorPollParsesDevices(t *testing.T) {
	xAuthToken := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xAuthToken = r.Header.Get("X-Auth-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"identification":{"id":"uisp-gw-1","name":"UISP Gateway 1","role":"gateway","mac":"aa:bb:cc:dd:ee:ff"},
				"site":{"id":"site-a"},
				"overview":{"status":"online","latency":4}
			},
			{
				"identification":{"id":"uisp-ap-1","name":"UISP AP 1","role":"ap"},
				"site":{"id":"site-a"},
				"overview":{"status":"offline","latency":110}
			}
		]`))
	}))
	defer server.Close()

	connector := NewUISPConnector(server.URL, "uisp-token-123456", "/nms/api/v2.1/devices")
	batch, err := connector.Poll(context.Background(), SourcePollRequest{Limit: 10})
	if err != nil {
		t.Fatalf("expected no error, got=%v", err)
	}

	if xAuthToken != "uisp-token-123456" {
		t.Fatalf("expected uisp x-auth-token header, got=%q", xAuthToken)
	}
	if batch.Response.Source != "uisp" {
		t.Fatalf("unexpected source=%q", batch.Response.Source)
	}
	if batch.Response.Fetched != 2 || batch.Response.Normalized != 2 {
		t.Fatalf("unexpected fetched/normalized=%d/%d", batch.Response.Fetched, batch.Response.Normalized)
	}
	if len(batch.Events) != 1 {
		t.Fatalf("expected 1 event for first seen offline device, got=%d", len(batch.Events))
	}
	if batch.Events[0].Source != "uisp" || batch.Events[0].EventType != "device_down" {
		t.Fatalf("unexpected event payload: %#v", batch.Events[0])
	}
}

func TestVendorConnectorPollCiscoBearer(t *testing.T) {
	authHeader := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"cisco-core-1","name":"Cisco Core 1","siteId":"dc-a","status":"online","latencyMs":7},
			{"id":"cisco-edge-1","name":"Cisco Edge 1","siteId":"dc-a","status":"offline"}
		]`))
	}))
	defer server.Close()

	connector := NewVendorConnector("cisco", "Cisco", server.URL, "cisco-token-123456", "/devices", "bearer")
	batch, err := connector.Poll(context.Background(), SourcePollRequest{Limit: 10})
	if err != nil {
		t.Fatalf("expected no error, got=%v", err)
	}

	if authHeader != "Bearer cisco-token-123456" {
		t.Fatalf("expected bearer auth header, got=%q", authHeader)
	}
	if batch.Response.Source != "cisco" {
		t.Fatalf("unexpected source=%q", batch.Response.Source)
	}
	if batch.Response.Fetched != 2 || batch.Response.Normalized != 2 {
		t.Fatalf("unexpected fetched/normalized=%d/%d", batch.Response.Fetched, batch.Response.Normalized)
	}
	if batch.Response.Demo {
		t.Fatalf("expected non-demo poll")
	}
	if len(batch.Events) != 1 {
		t.Fatalf("expected 1 event for first seen offline device, got=%d", len(batch.Events))
	}
	if batch.Events[0].Source != "cisco" || batch.Events[0].EventType != "device_down" {
		t.Fatalf("unexpected event payload: %#v", batch.Events[0])
	}

	status := connector.Status()
	if status.Source != "cisco" {
		t.Fatalf("unexpected status source=%q", status.Source)
	}
	if status.LastFetched != 2 || status.LastNormalized != 2 {
		t.Fatalf("unexpected status counts fetched=%d normalized=%d", status.LastFetched, status.LastNormalized)
	}
	if status.LastError != "" {
		t.Fatalf("expected empty status error, got=%q", status.LastError)
	}
}

func TestVendorConnectorPollJuniperXAuthToken(t *testing.T) {
	xAuthToken := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xAuthToken = r.Header.Get("X-Auth-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{"deviceId":"juniper-core-1","displayName":"Juniper Core 1","networkId":"site-1","online":true},
				{"deviceId":"juniper-edge-1","displayName":"Juniper Edge 1","networkId":"site-1","online":false}
			]
		}`))
	}))
	defer server.Close()

	connector := NewVendorConnector("juniper", "Juniper", server.URL, "juniper-token-123456", "/inventory/devices", "x-auth-token")
	batch, err := connector.Poll(context.Background(), SourcePollRequest{Limit: 10})
	if err != nil {
		t.Fatalf("expected no error, got=%v", err)
	}
	if xAuthToken != "juniper-token-123456" {
		t.Fatalf("expected x-auth-token header, got=%q", xAuthToken)
	}
	if batch.Response.Source != "juniper" {
		t.Fatalf("unexpected source=%q", batch.Response.Source)
	}
	if batch.Response.Fetched != 2 {
		t.Fatalf("expected fetched=2, got=%d", batch.Response.Fetched)
	}
	if len(batch.Events) != 1 {
		t.Fatalf("expected 1 event for first seen offline device, got=%d", len(batch.Events))
	}
	if batch.Events[0].Source != "juniper" || batch.Events[0].EventType != "device_down" {
		t.Fatalf("unexpected event payload: %#v", batch.Events[0])
	}
}

func TestVendorConnectorPollMerakiBearer(t *testing.T) {
	authHeader := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"Meraki MX 1","serial":"Q2XX-AAAA-0001","networkId":"net-1","status":"online","productType":"appliance","model":"MX95"},
			{"name":"Meraki MR 1","serial":"Q2XX-BBBB-0002","networkId":"net-1","status":"alerting","productType":"wireless","model":"MR46"},
			{"name":"Meraki MS 1","serial":"Q2XX-CCCC-0003","networkId":"net-1","status":"offline","productType":"switch","model":"MS250"}
		]`))
	}))
	defer server.Close()

	connector := NewVendorConnector("meraki", "Meraki", server.URL, "meraki-token-123456", "/devices/statuses", "bearer")
	batch, err := connector.Poll(context.Background(), SourcePollRequest{Limit: 10})
	if err != nil {
		t.Fatalf("expected no error, got=%v", err)
	}
	if authHeader != "Bearer meraki-token-123456" {
		t.Fatalf("expected bearer auth header, got=%q", authHeader)
	}
	if batch.Response.Source != "meraki" {
		t.Fatalf("unexpected source=%q", batch.Response.Source)
	}
	if batch.Response.Fetched != 3 || batch.Response.Normalized != 3 {
		t.Fatalf("unexpected fetched/normalized=%d/%d", batch.Response.Fetched, batch.Response.Normalized)
	}
	if len(batch.Events) != 1 {
		t.Fatalf("expected 1 event for first seen offline device, got=%d", len(batch.Events))
	}
	if batch.Events[0].Source != "meraki" || batch.Events[0].EventType != "device_down" {
		t.Fatalf("unexpected event payload: %#v", batch.Events[0])
	}
	if batch.Events[0].Role != "switch" {
		t.Fatalf("expected normalized switch role, got=%q", batch.Events[0].Role)
	}
}

func TestVendorConnectorPollErrorUpdatesStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"backend unavailable"}`))
	}))
	defer server.Close()

	connector := NewVendorConnector("cisco", "Cisco", server.URL, "cisco-token-123456", "/devices", "bearer")
	batch, err := connector.Poll(context.Background(), SourcePollRequest{Retries: 0})
	if err == nil {
		t.Fatalf("expected poll error")
	}
	if !strings.Contains(err.Error(), "cisco status 503") {
		t.Fatalf("unexpected error=%v", err)
	}
	if !strings.Contains(batch.Response.Error, "cisco status 503") {
		t.Fatalf("unexpected response error=%q", batch.Response.Error)
	}

	status := connector.Status()
	if !strings.Contains(status.LastError, "cisco status 503") {
		t.Fatalf("expected status error to include http code, got=%q", status.LastError)
	}
}

func TestVendorConnectorPollDemoWithoutCredentials(t *testing.T) {
	connector := NewVendorConnector("juniper", "Juniper", "", "", "/api/v1/devices", "bearer")
	batch, err := connector.Poll(context.Background(), SourcePollRequest{})
	if err != nil {
		t.Fatalf("expected demo poll success, got=%v", err)
	}
	if !batch.Response.Demo {
		t.Fatalf("expected demo mode response")
	}
	if batch.Response.Fetched == 0 || batch.Response.Normalized == 0 {
		t.Fatalf("expected demo records to be returned, got fetched=%d normalized=%d", batch.Response.Fetched, batch.Response.Normalized)
	}

	status := connector.Status()
	if !status.Demo {
		t.Fatalf("expected status to reflect demo mode")
	}
}
