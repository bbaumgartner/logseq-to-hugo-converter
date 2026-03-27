// Package main scans Logseq journal files for current-position:: properties
// and writes a journey.json file with positions and durations.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Position represents a single (clustered) stop on the journey.
type Position struct {
	Date string  `json:"date"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Days int     `json:"days"`
}

// JourneyMap is the structure written to journey.json.
// Positions holds one entry per clustered stop in chronological order.
type JourneyMap struct {
	Positions []Position `json:"positions"`
}

var positionRegex = regexp.MustCompile(`(?i)current-position::\s*(-?\d+(?:\.\d+)?)\s*,\s*(-?\d+(?:\.\d+)?)`)

// homeLat/homeLng is the home base location. Positions within homeExclusionDeg
// degrees in both axes are excluded from the journey map.
// clusterThresholdDeg controls how close two positions must be (in degrees, both
// axes) to be considered the same location and merged into one stop.
const (
	homeLat             = 47.13826
	homeLng             = 8.60032
	homeExclusionDeg    = 1.0
	clusterThresholdDeg = 0.01 // ~0.8-1.1 km — covers same harbour / adjacent anchorages
)

func isNearHome(lat, lng float64) bool {
	return math.Abs(lat-homeLat) <= homeExclusionDeg && math.Abs(lng-homeLng) <= homeExclusionDeg
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <journals-dir> <output-json-path>\n", os.Args[0])
		os.Exit(1)
	}

	journalsDir := os.Args[1]
	outputPath := os.Args[2]

	journey, err := extractPositions(journalsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting positions: %v\n", err)
		os.Exit(1)
	}

	if len(journey.Positions) == 0 {
		fmt.Println("No current-position entries found, skipping journey map generation")
		return
	}

	if err := writeJSON(journey, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %d journey positions to %s\n", len(journey.Positions), outputPath)
}

type journalEntry struct {
	date time.Time
	lat  float64
	lng  float64
}

// entryWithDays pairs a journal entry with the raw number of days at that location
// (the difference in calendar days to the next journal entry, or to today).
type entryWithDays struct {
	journalEntry
	days int
}

func extractPositions(dir string) (JourneyMap, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return JourneyMap{}, err
	}

	var entries []journalEntry
	for _, f := range files {
		base := strings.TrimSuffix(filepath.Base(f), ".md")
		dateStr := strings.ReplaceAll(base, "_", "-")
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read %s: %v\n", f, err)
			continue
		}

		matches := positionRegex.FindSubmatch(content)
		if matches == nil {
			continue
		}

		lat, err := strconv.ParseFloat(string(matches[1]), 64)
		if err != nil {
			continue
		}
		lng, err := strconv.ParseFloat(string(matches[2]), 64)
		if err != nil {
			continue
		}

		if isNearHome(lat, lng) {
			fmt.Printf("Skipping home position (%.5f, %.5f) from %s\n", lat, lng, date.Format("2006-01-02"))
			continue
		}

		entries = append(entries, journalEntry{date: date, lat: lat, lng: lng})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].date.Before(entries[j].date)
	})

	// Phase 1: compute raw days per entry (diff to next entry, or to today).
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	withDays := make([]entryWithDays, len(entries))
	for i, e := range entries {
		var nextDate time.Time
		if i+1 < len(entries) {
			nextDate = entries[i+1].date
		} else {
			nextDate = today
		}
		days := int(nextDate.Sub(e.date).Hours() / 24)
		if days < 1 {
			days = 1
		}
		withDays[i] = entryWithDays{e, days}
	}

	// Phase 2: merge entries that are geographically close into clusters.
	return clusterPositions(withDays), nil
}

// clusterPositions merges *consecutive* entries within clusterThresholdDeg of
// each other into a single Position, summing their days. Only the immediately
// preceding cluster is considered for merging, so returning to a location after
// leaving it always creates a new stop rather than silently collapsing into the
// earlier visit. The representative position and date are taken from the first
// (earliest) entry that started the run.
func clusterPositions(entries []entryWithDays) JourneyMap {
	type cluster struct {
		lat, lng  float64
		date      time.Time
		totalDays int
	}

	var clusters []cluster

	for _, e := range entries {
		last := len(clusters) - 1
		if last >= 0 &&
			math.Abs(e.lat-clusters[last].lat) <= clusterThresholdDeg &&
			math.Abs(e.lng-clusters[last].lng) <= clusterThresholdDeg {
			clusters[last].totalDays += e.days
			fmt.Printf("Merging (%.5f, %.5f) on %s into cluster at (%.5f, %.5f) — combined %d days\n",
				e.lat, e.lng, e.date.Format("2006-01-02"),
				clusters[last].lat, clusters[last].lng, clusters[last].totalDays)
		} else {
			clusters = append(clusters, cluster{
				lat:       e.lat,
				lng:       e.lng,
				date:      e.date,
				totalDays: e.days,
			})
		}
	}

	// Clusters are already in date order because entries were sorted before clustering.
	positions := make([]Position, len(clusters))
	for i, c := range clusters {
		positions[i] = Position{
			Date: c.date.Format("2006-01-02"),
			Lat:  c.lat,
			Lng:  c.lng,
			Days: c.totalDays,
		}
	}
	return JourneyMap{Positions: positions}
}

func writeJSON(journey JourneyMap, outputPath string) error {
	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(journey, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
