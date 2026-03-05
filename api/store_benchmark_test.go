package main

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkTelemetryRetentionCompaction(b *testing.B) {
	s := LoadStore("")
	nowMs := time.Now().UnixMilli()
	online := true

	buildSamples := func(prefix string, total int, stepMs int64) []TelemetrySample {
		out := make([]TelemetrySample, 0, total)
		for i := 0; i < total; i++ {
			out = append(out, TelemetrySample{
				SampleID:   fmt.Sprintf("%s-%d", prefix, i),
				DeviceID:   fmt.Sprintf("dev-%d", i%600),
				Source:     "bench",
				EventType:  "telemetry",
				Online:     &online,
				ObservedAt: nowMs - int64(i)*stepMs,
			})
		}
		return out
	}

	hotBaseline := buildSamples("hot", 25000, 1000)
	warmBaseline := buildSamples("warm", 25000, 2000)
	coldBaseline := buildSamples("cold", 25000, 3000)

	s.mu.Lock()
	s.TelemetryRetentionPolicy = TelemetryRetentionPolicy{
		HotRetentionMs:  int64((6 * time.Hour) / time.Millisecond),
		WarmRetentionMs: int64((7 * 24 * time.Hour) / time.Millisecond),
		ColdRetentionMs: int64((90 * 24 * time.Hour) / time.Millisecond),
		HotMaxSamples:   5000,
		WarmMaxSamples:  20000,
		ColdMaxSamples:  100000,
	}
	s.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.mu.Lock()
		s.TelemetryHot = append([]TelemetrySample(nil), hotBaseline...)
		s.TelemetryWarm = append([]TelemetrySample(nil), warmBaseline...)
		s.TelemetryCold = append([]TelemetrySample(nil), coldBaseline...)
		s.applyTelemetryRetentionLocked(nowMs + int64(i))
		s.mu.Unlock()
	}
}

func BenchmarkTelemetrySamplingGovernorIngest(b *testing.B) {
	s := LoadStore("")
	online := true

	s.mu.Lock()
	s.TelemetryGovernorRules = normalizeTelemetryGovernorRules([]TelemetryClassGovernorRule{
		{
			DeviceClass:         "core",
			MinSampleIntervalMs: int64((5 * time.Second) / time.Millisecond),
			QueuePriority:       0,
			Roles:               []string{"gateway"},
		},
		{
			DeviceClass:         "default",
			MinSampleIntervalMs: int64((20 * time.Second) / time.Millisecond),
			QueuePriority:       9,
			Roles:               []string{"device"},
		},
	})
	s.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := TelemetryIngestRequest{
			Source:   "bench_source",
			DeviceID: fmt.Sprintf("bench-device-%d", i%500),
			Device:   "Bench Device",
			Role:     "gateway",
			SiteID:   "bench-site",
			Online:   &online,
		}
		if _, _, ok := s.IngestTelemetry(req); !ok {
			b.Fatalf("ingest failed at iteration %d", i)
		}
	}
}
