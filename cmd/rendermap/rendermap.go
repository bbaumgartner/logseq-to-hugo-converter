// Package main reads a journey.json file and renders a static PNG map image
// using OpenStreetMap tiles with circles sized by duration and colored by age.
package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"

	sm "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
)

// Position mirrors the clustered stop structure written by cmd/journeymap.
type Position struct {
	Date string  `json:"date"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Days int     `json:"days"`
}

// LatLng is a coordinate pair used for the route polyline.
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// JourneyMap mirrors the top-level structure written by cmd/journeymap.
type JourneyMap struct {
	Positions []Position `json:"positions"`
	Route     []LatLng   `json:"route"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <journey-json-path> <output-png-path>\n", os.Args[0])
		os.Exit(1)
	}

	jsonPath := os.Args[1]
	outputPath := os.Args[2]

	journey, err := loadJourneyMap(jsonPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading journey map: %v\n", err)
		os.Exit(1)
	}

	if len(journey.Positions) == 0 {
		fmt.Println("No positions found, skipping map rendering")
		return
	}

	if err := renderMap(journey, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering map: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Journey map rendered to %s\n", outputPath)
}

func loadJourneyMap(path string) (JourneyMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return JourneyMap{}, err
	}
	var journey JourneyMap
	if err := json.Unmarshal(data, &journey); err != nil {
		return JourneyMap{}, err
	}
	return journey, nil
}

// lerpColor interpolates from gray (t=0, oldest) to sailing blue (t=1, newest).
func lerpColor(t float64) color.RGBA {
	r := uint8(math.Round(150 * (1 - t)))
	g := uint8(math.Round(150*(1-t) + 119*t))
	b := uint8(math.Round(150*(1-t) + 204*t))
	return color.RGBA{R: r, G: g, B: b, A: 200}
}

// strokeColor returns a darker, fully opaque version of a fill color for the circle border.
func strokeColor(fill color.RGBA) color.RGBA {
	return color.RGBA{
		R: fill.R / 2,
		G: fill.G / 2,
		B: fill.B / 2,
		A: 255,
	}
}

// circleRadiusMeters returns a geographic radius scaled by duration.
// Uses sqrt scaling so short stays are still distinguishable.
func circleRadiusMeters(days int) float64 {
	return math.Min(7000*math.Sqrt(float64(days)), 80000)
}

func renderMap(journey JourneyMap, outputPath string) error {
	ctx := sm.NewContext()
	ctx.SetSize(900, 500)
	ctx.OverrideAttribution("")

	positions := journey.Positions
	n := len(positions)

	// Route polyline — drawn first so circles render on top.
	// Uses journey.Route which preserves the actual travel order including
	// revisits to the same cluster (e.g. Portorož → Medulin → Portorož).
	if len(journey.Route) > 1 {
		latLngs := make([]s2.LatLng, len(journey.Route))
		for i, p := range journey.Route {
			latLngs[i] = s2.LatLngFromDegrees(p.Lat, p.Lng)
		}
		ctx.AddObject(sm.NewPath(latLngs, color.RGBA{R: 100, G: 100, B: 100, A: 180}, 2.0))
	}

	// Circles — oldest first so newest renders on top.
	for i, p := range positions {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		} else {
			t = 1.0
		}
		fill := lerpColor(t)
		border := strokeColor(fill)
		radius := circleRadiusMeters(p.Days)
		pos := s2.LatLngFromDegrees(p.Lat, p.Lng)
		ctx.AddObject(sm.NewCircle(pos, border, fill, radius, 2.0))
	}

	img, err := ctx.Render()
	if err != nil {
		return fmt.Errorf("rendering map: %w", err)
	}

	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return gg.SavePNG(outputPath, img)
}
