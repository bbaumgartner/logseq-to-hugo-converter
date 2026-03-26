package main

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"
)

// ---- markerSize ----

func TestMarkerSize_Endpoints(t *testing.T) {
	if markerSize(1) != 30 {
		t.Errorf("markerSize(1) = %d, want 30 (minimum)", markerSize(1))
	}
	if markerSize(30) != 100 {
		t.Errorf("markerSize(30) = %d, want 100 (maximum)", markerSize(30))
	}
}

func TestMarkerSize_Clamped(t *testing.T) {
	if markerSize(0) != 30 {
		t.Errorf("markerSize(0) = %d, want 30 (clamped to minimum)", markerSize(0))
	}
	if markerSize(1000) != 100 {
		t.Errorf("markerSize(1000) = %d, want 100 (clamped to maximum)", markerSize(1000))
	}
}

func TestMarkerSize_Midpoint(t *testing.T) {
	// t = (15-1)/(30-1) = 14/29 ≈ 0.483, so size = 30 + 0.483*70 ≈ 64 (rounded).
	got := markerSize(15)
	if got < 60 || got > 68 {
		t.Errorf("markerSize(15) = %d, want ~64 (midpoint)", got)
	}
}

func TestMarkerSize_MonotonicallyIncreasing(t *testing.T) {
	prev := markerSize(1)
	for days := 2; days <= 30; days++ {
		curr := markerSize(days)
		if curr < prev {
			t.Errorf("markerSize not non-decreasing: markerSize(%d)=%d < markerSize(%d)=%d", days, curr, days-1, prev)
		}
		prev = curr
	}
}

// ---- loadLogos ----

func TestLoadLogos_Succeeds(t *testing.T) {
	logos, err := loadLogos()
	if err != nil {
		t.Fatalf("loadLogos() error: %v", err)
	}
	if len(logos) != 8 {
		t.Errorf("loadLogos() returned %d entries, want 8", len(logos))
	}
	for _, l := range logos {
		b := l.img.Bounds()
		if b.Dx() == 0 || b.Dy() == 0 {
			t.Errorf("logo_%dpx has zero-size bounds", l.size)
		}
	}
}

func TestLoadLogos_SizesMatch(t *testing.T) {
	logos, err := loadLogos()
	if err != nil {
		t.Fatalf("loadLogos() error: %v", err)
	}
	want := []int{30, 40, 50, 60, 70, 80, 90, 100}
	for i, l := range logos {
		if l.size != want[i] {
			t.Errorf("logos[%d].size = %d, want %d", i, l.size, want[i])
		}
	}
}

// ---- nearestLogo ----

func makeLogoEntries(sizes []int) []logoEntry {
	entries := make([]logoEntry, len(sizes))
	for i, s := range sizes {
		entries[i] = logoEntry{size: s, img: image.NewRGBA(image.Rect(0, 0, s, s))}
	}
	return entries
}

func TestNearestLogo_ExactMatch(t *testing.T) {
	logos := makeLogoEntries([]int{30, 40, 50, 60, 70, 80, 90, 100})
	for _, want := range []int{30, 40, 50, 60, 70, 80, 90, 100} {
		got := nearestLogo(logos, want)
		b := got.Bounds()
		if b.Dx() != want {
			t.Errorf("nearestLogo(logos, %d).Bounds().Dx() = %d, want %d", want, b.Dx(), want)
		}
	}
}

func TestNearestLogo_RoundsToNearest(t *testing.T) {
	logos := makeLogoEntries([]int{30, 40, 50, 60, 70, 80, 90, 100})
	cases := []struct {
		size int
		want int
	}{
		{10, 30},  // below 30 → 30
		{34, 30},  // closer to 30 than 40
		{35, 30},  // exactly midpoint (picks first on tie)
		{36, 40},  // closer to 40 than 30
		{55, 50},  // closer to 50 than 60
		{56, 60},  // closer to 60 than 50
		{110, 100}, // above 100 → 100
	}
	for _, tc := range cases {
		got := nearestLogo(logos, tc.size)
		b := got.Bounds()
		if b.Dx() != tc.want {
			t.Errorf("nearestLogo(logos, %d).Bounds().Dx() = %d, want %d", tc.size, b.Dx(), tc.want)
		}
	}
}

// ---- loadJourneyMap ----

func makeJourneyJSON(t *testing.T, journey JourneyMap) []byte {
	t.Helper()
	data, err := json.Marshal(journey)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return data
}

func TestLoadJourneyMap_Valid(t *testing.T) {
	dir := t.TempDir()
	input := JourneyMap{
		Positions: []Position{
			{Date: "2025-09-13", Lat: 45.5127, Lng: 13.5954, Days: 126},
			{Date: "2026-01-17", Lat: 43.5088, Lng: 16.4402, Days: 10},
		},
		Route: []LatLng{
			{Lat: 45.5127, Lng: 13.5954},
			{Lat: 43.5088, Lng: 16.4402},
		},
	}
	jsonPath := filepath.Join(dir, "journey.json")
	os.WriteFile(jsonPath, makeJourneyJSON(t, input), 0644)

	got, err := loadJourneyMap(jsonPath)
	if err != nil {
		t.Fatalf("loadJourneyMap() error: %v", err)
	}
	if len(got.Positions) != len(input.Positions) {
		t.Fatalf("got %d positions, want %d", len(got.Positions), len(input.Positions))
	}
	if len(got.Route) != len(input.Route) {
		t.Fatalf("got %d route entries, want %d", len(got.Route), len(input.Route))
	}
	for i := range input.Positions {
		if got.Positions[i] != input.Positions[i] {
			t.Errorf("positions[%d] = %+v, want %+v", i, got.Positions[i], input.Positions[i])
		}
	}
}

func TestLoadJourneyMap_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "journey.json")
	os.WriteFile(jsonPath, []byte(`{"positions":[],"route":[]}`), 0644)

	got, err := loadJourneyMap(jsonPath)
	if err != nil {
		t.Fatalf("loadJourneyMap() error: %v", err)
	}
	if len(got.Positions) != 0 || len(got.Route) != 0 {
		t.Errorf("expected empty JourneyMap, got %+v", got)
	}
}

func TestLoadJourneyMap_FileNotFound(t *testing.T) {
	_, err := loadJourneyMap("/nonexistent/path/journey.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadJourneyMap_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "journey.json")
	os.WriteFile(jsonPath, []byte("{not valid json}"), 0644)

	_, err := loadJourneyMap(jsonPath)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ---- renderMap integration ----

// These tests download OSM tiles and require network access.

func TestRenderMap_ProducesFile(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "journey-map.png")

	journey := JourneyMap{
		Positions: []Position{
			{Date: "2025-09-13", Lat: 45.5127, Lng: 13.5954, Days: 126},
			{Date: "2026-01-17", Lat: 43.5088, Lng: 16.4402, Days: 67},
		},
		Route: []LatLng{
			{Lat: 45.5127, Lng: 13.5954},
			{Lat: 43.5088, Lng: 16.4402},
		},
	}

	if err := renderMap(journey, outputPath); err != nil {
		t.Fatalf("renderMap() error: %v", err)
	}

	info, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		t.Fatal("renderMap() did not create output file")
	}
	if info.Size() == 0 {
		t.Error("renderMap() created an empty file")
	}
}

func TestRenderMap_RouteRevisit(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "journey-map.png")

	journey := JourneyMap{
		Positions: []Position{
			{Date: "2026-03-14", Lat: 45.50543, Lng: 13.59597, Days: 5},
			{Date: "2026-03-18", Lat: 45.15039, Lng: 13.59877, Days: 1},
		},
		Route: []LatLng{
			{Lat: 45.50543, Lng: 13.59597},
			{Lat: 45.15039, Lng: 13.59877},
			{Lat: 45.50543, Lng: 13.59597},
		},
	}

	if err := renderMap(journey, outputPath); err != nil {
		t.Fatalf("renderMap() error with revisit route: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("renderMap() did not create output file")
	}
}

func TestRenderMap_SinglePosition(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "journey-map.png")

	journey := JourneyMap{
		Positions: []Position{{Date: "2025-09-13", Lat: 45.5127, Lng: 13.5954, Days: 30}},
		Route:     []LatLng{{Lat: 45.5127, Lng: 13.5954}},
	}

	if err := renderMap(journey, outputPath); err != nil {
		t.Fatalf("renderMap() error with single position: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("renderMap() did not create output file for single position")
	}
}

func TestRenderMap_CreatesIntermediateDirectories(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "sub", "dir", "map.png")

	journey := JourneyMap{
		Positions: []Position{{Date: "2025-09-13", Lat: 45.5127, Lng: 13.5954, Days: 10}},
		Route:     []LatLng{{Lat: 45.5127, Lng: 13.5954}},
	}

	if err := renderMap(journey, outputPath); err != nil {
		t.Fatalf("renderMap() error: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("renderMap() did not create output file in nested directory")
	}
}
