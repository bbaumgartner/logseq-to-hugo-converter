package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---- isNearHome ----

func TestIsNearHome(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lng  float64
		want bool
	}{
		{"exact home coordinates", homeLat, homeLng, true},
		{"exactly on lat boundary", homeLat + homeExclusionDeg, homeLng, true},
		{"exactly on lng boundary", homeLat, homeLng + homeExclusionDeg, true},
		{"one tick outside lat boundary", homeLat + homeExclusionDeg + 0.001, homeLng, false},
		{"one tick outside lng boundary", homeLat, homeLng + homeExclusionDeg + 0.001, false},
		{"south of home within range", homeLat - 0.5, homeLng, true},
		{"west of home within range", homeLat, homeLng - 0.9, true},
		{"Portoroz Slovenia", 45.5127, 13.5954, false},
		{"Zagreb Croatia", 45.8150, 15.9819, false},
		{"Rome Italy", 41.9028, 12.4964, false},
		{"lat ok but lng too far east", homeLat, homeLng + 1.5, false},
		{"both axes outside", homeLat + 2.0, homeLng + 2.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNearHome(tt.lat, tt.lng)
			if got != tt.want {
				t.Errorf("isNearHome(%.5f, %.5f) = %v, want %v", tt.lat, tt.lng, got, tt.want)
			}
		})
	}
}

// ---- extractPositions helpers ----

// writeJournal creates a journal file named YYYY_MM_DD.md in dir with the given content.
func writeJournal(t *testing.T, dir, date, content string) {
	t.Helper()
	name := date[:4] + "_" + date[5:7] + "_" + date[8:10] + ".md"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writeJournal(%s): %v", date, err)
	}
}

// ---- extractPositions ----

func TestExtractPositions_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 0 {
		t.Errorf("expected 0 positions, got %d", len(j.Positions))
	}
}

func TestExtractPositions_NoPositionProperty(t *testing.T) {
	dir := t.TempDir()
	writeJournal(t, dir, "2025-09-13", "- [[Blog]]\n\t- type:: blog\n\t  title:: No position here\n")
	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 0 {
		t.Errorf("expected 0 positions, got %d", len(j.Positions))
	}
}

func TestExtractPositions_NonDateFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("- current-position:: 45.0,13.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 0 {
		t.Errorf("expected 0 positions from non-date file, got %d", len(j.Positions))
	}
}

func TestExtractPositions_SinglePosition(t *testing.T) {
	dir := t.TempDir()
	writeJournal(t, dir, "2025-09-13", "- current-position:: 45.5127,13.5954\n")

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(j.Positions))
	}

	p := j.Positions[0]
	if p.Date != "2025-09-13" {
		t.Errorf("Date = %q, want %q", p.Date, "2025-09-13")
	}
	if p.Lat != 45.5127 {
		t.Errorf("Lat = %v, want 45.5127", p.Lat)
	}
	if p.Lng != 13.5954 {
		t.Errorf("Lng = %v, want 13.5954", p.Lng)
	}
	if p.Days < 1 {
		t.Errorf("Days = %d, want >= 1", p.Days)
	}
}

func TestExtractPositions_MultiplePositions_SortedAndDaysCorrect(t *testing.T) {
	dir := t.TempDir()
	writeJournal(t, dir, "2026-01-17", "- current-position:: 43.5088,16.4402\n")
	writeJournal(t, dir, "2025-09-13", "- current-position:: 45.5127,13.5954\n")

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(j.Positions))
	}

	if j.Positions[0].Date != "2025-09-13" {
		t.Errorf("positions[0].Date = %q, want %q", j.Positions[0].Date, "2025-09-13")
	}
	if j.Positions[1].Date != "2026-01-17" {
		t.Errorf("positions[1].Date = %q, want %q", j.Positions[1].Date, "2026-01-17")
	}

	// 2025-09-13 to 2026-01-17 = 126 days.
	if j.Positions[0].Days != 126 {
		t.Errorf("positions[0].Days = %d, want 126", j.Positions[0].Days)
	}
	if j.Positions[1].Days < 1 {
		t.Errorf("positions[1].Days = %d, want >= 1", j.Positions[1].Days)
	}
}

func TestExtractPositions_HomePositionFiltered(t *testing.T) {
	dir := t.TempDir()
	writeJournal(t, dir, "2025-09-13", "- current-position:: 45.5127,13.5954\n")
	writeJournal(t, dir, "2025-12-01", "- current-position:: 47.13826,8.60032\n") // exact home

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 1 {
		t.Fatalf("expected 1 position after filtering home, got %d", len(j.Positions))
	}
	if j.Positions[0].Lat != 45.5127 {
		t.Errorf("expected the remote position to survive, got lat %v", j.Positions[0].Lat)
	}
}

func TestExtractPositions_HomePositionNearbyFiltered(t *testing.T) {
	dir := t.TempDir()
	writeJournal(t, dir, "2025-06-01", "- current-position:: 46.50000,8.10000\n")

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 0 {
		t.Errorf("expected 0 positions (all near home), got %d", len(j.Positions))
	}
}

func TestExtractPositions_PropertyInsideBlogBlock(t *testing.T) {
	dir := t.TempDir()
	content := "- [[Blog]]\n\t- type:: blog\n\t  current-position:: 45.5127,13.5954\n\t  title:: Test\n"
	writeJournal(t, dir, "2025-09-13", content)

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 1 {
		t.Fatalf("expected 1 position from indented property, got %d", len(j.Positions))
	}
	if j.Positions[0].Lat != 45.5127 {
		t.Errorf("Lat = %v, want 45.5127", j.Positions[0].Lat)
	}
}

func TestExtractPositions_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	writeJournal(t, dir, "2025-09-13", "- Current-Position:: 45.5127,13.5954\n")

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 1 {
		t.Errorf("expected 1 position with mixed-case property, got %d", len(j.Positions))
	}
}

func TestExtractPositions_NegativeCoordinates(t *testing.T) {
	dir := t.TempDir()
	writeJournal(t, dir, "2027-03-01", "- current-position:: -33.8688,151.2093\n") // Sydney

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(j.Positions))
	}
	if j.Positions[0].Lat != -33.8688 {
		t.Errorf("Lat = %v, want -33.8688", j.Positions[0].Lat)
	}
	if j.Positions[0].Lng != 151.2093 {
		t.Errorf("Lng = %v, want 151.2093", j.Positions[0].Lng)
	}
}

func TestExtractPositions_SpacesAroundComma(t *testing.T) {
	dir := t.TempDir()
	writeJournal(t, dir, "2025-09-13", "- current-position:: 45.5127 , 13.5954\n")

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 1 {
		t.Errorf("expected 1 position with spaced comma, got %d", len(j.Positions))
	}
}

func TestExtractPositions_MinimumOneDayEnforced(t *testing.T) {
	dir := t.TempDir()
	today := time.Now().Truncate(24 * time.Hour)
	writeJournal(t, dir, today.Format("2006-01-02"), "- current-position:: 45.0,13.0\n")

	j, err := extractPositions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j.Positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(j.Positions))
	}
	if j.Positions[0].Days < 1 {
		t.Errorf("Days = %d, want >= 1", j.Positions[0].Days)
	}
}

// ---- clusterPositions ----

func makeEntry(dateStr string, lat, lng float64, days int) entryWithDays {
	d, _ := time.Parse("2006-01-02", dateStr)
	return entryWithDays{journalEntry{date: d, lat: lat, lng: lng}, days}
}

func TestClusterPositions_NoEntries(t *testing.T) {
	got := clusterPositions(nil)
	if len(got.Positions) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(got.Positions))
	}
	if len(got.Route) != 0 {
		t.Errorf("expected empty route, got %d entries", len(got.Route))
	}
}

func TestClusterPositions_AllDistinct(t *testing.T) {
	entries := []entryWithDays{
		makeEntry("2025-09-13", 45.5127, 13.5954, 10), // Portoroz
		makeEntry("2025-10-01", 43.5088, 16.4402, 5),  // Split (~300km away)
		makeEntry("2025-10-15", 42.6507, 18.0944, 7),  // Dubrovnik (~200km away)
	}
	got := clusterPositions(entries)
	if len(got.Positions) != 3 {
		t.Fatalf("expected 3 clusters (all distinct), got %d", len(got.Positions))
	}
	if got.Positions[0].Days != 10 || got.Positions[1].Days != 5 || got.Positions[2].Days != 7 {
		t.Errorf("days mismatch: got %v", got.Positions)
	}
	if len(got.Route) != 3 {
		t.Errorf("expected route length 3, got %d", len(got.Route))
	}
}

func TestClusterPositions_ConsecutiveMerge(t *testing.T) {
	// Two entries in essentially the same harbour.
	entries := []entryWithDays{
		makeEntry("2026-03-14", 45.50543, 13.59597, 4), // Portoroz berth A
		makeEntry("2026-03-18", 45.50591, 13.59765, 3), // Portoroz berth B (0.002° apart)
	}
	got := clusterPositions(entries)
	if len(got.Positions) != 1 {
		t.Fatalf("expected 1 cluster after merging same harbour, got %d", len(got.Positions))
	}
	if got.Positions[0].Days != 7 {
		t.Errorf("merged days = %d, want 7", got.Positions[0].Days)
	}
	if got.Positions[0].Date != "2026-03-14" {
		t.Errorf("representative date = %q, want %q", got.Positions[0].Date, "2026-03-14")
	}
	if got.Positions[0].Lat != 45.50543 {
		t.Errorf("representative lat = %v, want 45.50543", got.Positions[0].Lat)
	}
	// Route should still have 2 entries (both berths in order).
	if len(got.Route) != 2 {
		t.Errorf("expected route length 2, got %d", len(got.Route))
	}
}

func TestClusterPositions_LeaveAndReturn(t *testing.T) {
	// Go to Portoroz, leave for a day trip south, return to Portoroz.
	// Both Portoroz entries should merge into one cluster; Medulin stays separate.
	// The route should be: Portoroz → Medulin → Portoroz (3 entries).
	entries := []entryWithDays{
		makeEntry("2026-03-14", 45.50543, 13.59597, 4), // Portoroz
		makeEntry("2026-03-18", 45.15039, 13.59877, 1), // Medulin (~40km south)
		makeEntry("2026-03-19", 45.50591, 13.59765, 1), // back to Portoroz
	}
	got := clusterPositions(entries)

	if len(got.Positions) != 2 {
		t.Fatalf("expected 2 clusters (Portoroz + Medulin), got %d", len(got.Positions))
	}
	if got.Positions[0].Days != 5 {
		t.Errorf("Portoroz cluster days = %d, want 5 (4+1)", got.Positions[0].Days)
	}
	if got.Positions[1].Days != 1 {
		t.Errorf("Medulin cluster days = %d, want 1", got.Positions[1].Days)
	}

	// Route must have 3 entries reflecting the actual travel order.
	if len(got.Route) != 3 {
		t.Fatalf("expected route length 3 (Portoroz→Medulin→Portoroz), got %d", len(got.Route))
	}
	// First and third route entries should point to the Portoroz cluster representative.
	if got.Route[0].Lat != 45.50543 || got.Route[2].Lat != 45.50543 {
		t.Errorf("first and third route entries should be Portoroz cluster (lat 45.50543), got %v", got.Route)
	}
	// Second route entry should point to Medulin.
	if got.Route[1].Lat != 45.15039 {
		t.Errorf("second route entry should be Medulin (lat 45.15039), got %v", got.Route[1])
	}
}

func TestClusterPositions_WithinThreshold(t *testing.T) {
	entries := []entryWithDays{
		makeEntry("2025-09-13", 45.0, 13.0, 5),
		makeEntry("2025-09-20", 45.0+clusterThresholdDeg*0.9, 13.0+clusterThresholdDeg*0.9, 3),
	}
	got := clusterPositions(entries)
	if len(got.Positions) != 1 {
		t.Fatalf("expected 1 cluster for positions within threshold, got %d", len(got.Positions))
	}
}

func TestClusterPositions_JustOutsideThreshold(t *testing.T) {
	entries := []entryWithDays{
		makeEntry("2025-09-13", 45.0, 13.0, 5),
		makeEntry("2025-09-20", 45.0+clusterThresholdDeg+0.001, 13.0, 3),
	}
	got := clusterPositions(entries)
	if len(got.Positions) != 2 {
		t.Fatalf("expected 2 clusters for positions just outside threshold, got %d", len(got.Positions))
	}
}

func TestClusterPositions_PreservesChronologicalOrder(t *testing.T) {
	entries := []entryWithDays{
		makeEntry("2025-09-13", 45.5, 13.6, 10),
		makeEntry("2025-10-01", 43.5, 16.4, 5),
		makeEntry("2025-10-15", 40.0, 18.0, 3),
	}
	got := clusterPositions(entries)
	for i := 1; i < len(got.Positions); i++ {
		t1, _ := time.Parse("2006-01-02", got.Positions[i-1].Date)
		t2, _ := time.Parse("2006-01-02", got.Positions[i].Date)
		if !t1.Before(t2) {
			t.Errorf("clusters not in chronological order at index %d: %s >= %s",
				i, got.Positions[i-1].Date, got.Positions[i].Date)
		}
	}
}

// ---- writeJSON ----

func TestWriteJSON_CreatesValidFile(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "journey.json")
	journey := JourneyMap{
		Positions: []Position{
			{Date: "2025-09-13", Lat: 45.5127, Lng: 13.5954, Days: 126},
			{Date: "2026-01-17", Lat: 43.5088, Lng: 16.4402, Days: 10},
		},
		Route: []LatLng{
			{Lat: 45.5127, Lng: 13.5954},
			{Lat: 43.5088, Lng: 16.4402},
		},
	}

	if err := writeJSON(journey, outputPath); err != nil {
		t.Fatalf("writeJSON() error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("could not read output file: %v", err)
	}

	var got JourneyMap
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(got.Positions) != len(journey.Positions) {
		t.Fatalf("got %d positions, want %d", len(got.Positions), len(journey.Positions))
	}
	if len(got.Route) != len(journey.Route) {
		t.Fatalf("got %d route entries, want %d", len(got.Route), len(journey.Route))
	}
	for i := range journey.Positions {
		if got.Positions[i] != journey.Positions[i] {
			t.Errorf("positions[%d] = %+v, want %+v", i, got.Positions[i], journey.Positions[i])
		}
	}
}

func TestWriteJSON_CreatesIntermediateDirectories(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "subdir", "nested", "journey.json")
	journey := JourneyMap{
		Positions: []Position{{Date: "2025-09-13", Lat: 1, Lng: 2, Days: 3}},
		Route:     []LatLng{{Lat: 1, Lng: 2}},
	}

	if err := writeJSON(journey, outputPath); err != nil {
		t.Fatalf("writeJSON() error: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to exist after creating intermediate dirs")
	}
}

func TestWriteJSON_EmptyJourneyMap(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "journey.json")

	if err := writeJSON(JourneyMap{}, outputPath); err != nil {
		t.Fatalf("writeJSON() error for empty JourneyMap: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var got JourneyMap
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(got.Positions) != 0 || len(got.Route) != 0 {
		t.Errorf("expected empty JourneyMap in JSON, got %+v", got)
	}
}
