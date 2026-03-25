package main

import (
	"encoding/json"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// ---- lerpColor ----

func TestLerpColor_Endpoints(t *testing.T) {
	oldest := lerpColor(0.0)
	if oldest.R != 150 || oldest.G != 150 || oldest.B != 150 {
		t.Errorf("lerpColor(0) = %v, want R=150 G=150 B=150 (gray)", oldest)
	}

	newest := lerpColor(1.0)
	if newest.R != 0 || newest.G != 119 || newest.B != 204 {
		t.Errorf("lerpColor(1) = %v, want R=0 G=119 B=204 (sailing blue)", newest)
	}
}

func TestLerpColor_Midpoint(t *testing.T) {
	mid := lerpColor(0.5)
	wantR := uint8(math.Round(150 * 0.5))
	wantG := uint8(math.Round(150*0.5 + 119*0.5))
	wantB := uint8(math.Round(150*0.5 + 204*0.5))
	if mid.R != wantR || mid.G != wantG || mid.B != wantB {
		t.Errorf("lerpColor(0.5) = %v, want R=%d G=%d B=%d", mid, wantR, wantG, wantB)
	}
}

func TestLerpColor_AlphaConstant(t *testing.T) {
	for _, t2 := range []float64{0, 0.25, 0.5, 0.75, 1.0} {
		c := lerpColor(t2)
		if c.A != 200 {
			t.Errorf("lerpColor(%v).A = %d, want 200", t2, c.A)
		}
	}
}

func TestLerpColor_MonotonicallyIncreasingBlue(t *testing.T) {
	// Blue channel should increase as t increases (older=gray has less blue).
	steps := []float64{0, 0.25, 0.5, 0.75, 1.0}
	prev := lerpColor(steps[0])
	for _, t2 := range steps[1:] {
		curr := lerpColor(t2)
		if curr.B < prev.B {
			t.Errorf("Blue channel decreased from t=%.2f (B=%d) to t=%.2f (B=%d)",
				t2-0.25, prev.B, t2, curr.B)
		}
		prev = curr
	}
}

// ---- strokeColor ----

func TestStrokeColor_DarkerThanFill(t *testing.T) {
	fill := color.RGBA{R: 100, G: 80, B: 60, A: 200}
	stroke := strokeColor(fill)

	if stroke.R != 50 {
		t.Errorf("strokeColor R = %d, want %d", stroke.R, 50)
	}
	if stroke.G != 40 {
		t.Errorf("strokeColor G = %d, want %d", stroke.G, 40)
	}
	if stroke.B != 30 {
		t.Errorf("strokeColor B = %d, want %d", stroke.B, 30)
	}
	if stroke.A != 255 {
		t.Errorf("strokeColor A = %d, want 255 (fully opaque)", stroke.A)
	}
}

func TestStrokeColor_BlackFillStaysBlack(t *testing.T) {
	stroke := strokeColor(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	if stroke.R != 0 || stroke.G != 0 || stroke.B != 0 {
		t.Errorf("stroke of black fill should be black, got %v", stroke)
	}
}

// ---- circleRadiusMeters ----

func TestCircleRadiusMeters_ScalesWithDays(t *testing.T) {
	r1 := circleRadiusMeters(1)
	r7 := circleRadiusMeters(7)
	r30 := circleRadiusMeters(30)

	if r1 >= r7 {
		t.Errorf("expected r(1) < r(7), got r(1)=%.0f r(7)=%.0f", r1, r7)
	}
	if r7 >= r30 {
		t.Errorf("expected r(7) < r(30), got r(7)=%.0f r(30)=%.0f", r7, r30)
	}
}

func TestCircleRadiusMeters_MinimumIsPositive(t *testing.T) {
	r := circleRadiusMeters(1)
	if r <= 0 {
		t.Errorf("circleRadiusMeters(1) = %.0f, want > 0", r)
	}
}

func TestCircleRadiusMeters_CappedAtMaximum(t *testing.T) {
	// Very long stay should not produce a radius larger than 80 000 m.
	r := circleRadiusMeters(100000)
	if r > 80000 {
		t.Errorf("circleRadiusMeters(100000) = %.0f, exceeds cap of 80000", r)
	}
}

func TestCircleRadiusMeters_SqrtScaling(t *testing.T) {
	// radius should follow 7000 * sqrt(days) up to the cap.
	for _, days := range []int{1, 4, 9, 16} {
		got := circleRadiusMeters(days)
		want := 7000 * math.Sqrt(float64(days))
		if math.Abs(got-want) > 1 {
			t.Errorf("circleRadiusMeters(%d) = %.2f, want %.2f", days, got, want)
		}
	}
}

// ---- loadPositions ----

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

// TestRenderMap_ProducesFile verifies that renderMap creates a valid PNG file.
// This test downloads OSM tiles and requires network access.
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
	// Verify that a leave-and-return journey (route longer than positions) renders without error.
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "journey-map.png")

	journey := JourneyMap{
		Positions: []Position{
			{Date: "2026-03-14", Lat: 45.50543, Lng: 13.59597, Days: 5}, // Portoroz (merged)
			{Date: "2026-03-18", Lat: 45.15039, Lng: 13.59877, Days: 1}, // Medulin
		},
		Route: []LatLng{
			{Lat: 45.50543, Lng: 13.59597}, // Portoroz (first visit)
			{Lat: 45.15039, Lng: 13.59877}, // Medulin
			{Lat: 45.50543, Lng: 13.59597}, // Portoroz (return)
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
