// Package main reads a journey.json file and renders a static PNG map image
// using OpenStreetMap tiles with the Sailing Nomads logo sized by duration of stay.
// Logo markers are drawn at one of eight pre-prepared sizes (30–100px in 10px steps);
// markerSize linearly interpolates between 30px (1 day) and 100px (≥30 days),
// then nearestLogo selects the closest pre-prepared image without runtime scaling.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
	"path/filepath"

	sm "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
)

//go:embed logo_30px.png
var logo30 []byte

//go:embed logo_40px.png
var logo40 []byte

//go:embed logo_50px.png
var logo50 []byte

//go:embed logo_60px.png
var logo60 []byte

//go:embed logo_70px.png
var logo70 []byte

//go:embed logo_80px.png
var logo80 []byte

//go:embed logo_90px.png
var logo90 []byte

//go:embed logo_100px.png
var logo100 []byte

var routeColor = color.RGBA{R: 100, G: 100, B: 100, A: 180}

// logoEntry pairs a pre-prepared logo image with its pixel size.
type logoEntry struct {
	size int
	img  image.Image
}

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

	journey, err := loadJourneyMap(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading journey map: %v\n", err)
		os.Exit(1)
	}

	if len(journey.Positions) == 0 {
		fmt.Println("No positions found, skipping map rendering")
		return
	}

	if err := renderMap(journey, os.Args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering map: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Journey map rendered to %s\n", os.Args[2])
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

// loadLogos decodes all embedded logo PNGs and returns them sorted by size.
func loadLogos() ([]logoEntry, error) {
	candidates := []struct {
		size int
		data []byte
	}{
		{30, logo30},
		{40, logo40},
		{50, logo50},
		{60, logo60},
		{70, logo70},
		{80, logo80},
		{90, logo90},
		{100, logo100},
	}
	logos := make([]logoEntry, 0, len(candidates))
	for _, c := range candidates {
		img, _, err := image.Decode(bytes.NewReader(c.data))
		if err != nil {
			return nil, fmt.Errorf("decoding logo_%dpx.png: %w", c.size, err)
		}
		logos = append(logos, logoEntry{size: c.size, img: img})
	}
	return logos, nil
}

// nearestLogo returns the logo whose size is closest to the requested pixel size.
func nearestLogo(logos []logoEntry, size int) image.Image {
	best := logos[0]
	for _, l := range logos[1:] {
		if abs(l.size-size) < abs(best.size-size) {
			best = l
		}
	}
	return best.img
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// markerSize returns the ideal pixel size for a logo marker scaled by duration.
// Linearly interpolates between 20px (1 day) and 100px (30+ days).
// The result is then quantized to the nearest pre-prepared logo size by nearestLogo.
func markerSize(days int) int {
	const minSize, maxSize = 30, 100
	const minDays, maxDays = 1, 30
	if days <= minDays {
		return minSize
	}
	if days >= maxDays {
		return maxSize
	}
	t := float64(days-minDays) / float64(maxDays-minDays)
	return int(math.Round(float64(minSize) + t*float64(maxSize-minSize)))
}

func renderMap(journey JourneyMap, outputPath string) error {
	logos, err := loadLogos()
	if err != nil {
		return fmt.Errorf("loading logos: %w", err)
	}

	ctx := sm.NewContext()
	ctx.SetSize(900, 500)
	ctx.OverrideAttribution("")

	// Route polyline — drawn first so markers render on top.
	// Uses journey.Route which preserves the actual travel order including
	// revisits to the same cluster (e.g. Portorož → Medulin → Portorož).
	if len(journey.Route) > 1 {
		latLngs := make([]s2.LatLng, len(journey.Route))
		for i, p := range journey.Route {
			latLngs[i] = s2.LatLngFromDegrees(p.Lat, p.Lng)
		}
		ctx.AddObject(sm.NewPath(latLngs, routeColor, 2.0))
	}

	// Logo markers — oldest first so newest renders on top.
	for _, p := range journey.Positions {
		size := markerSize(p.Days)
		logo := nearestLogo(logos, size)
		offset := float64(size) / 2
		pos := s2.LatLngFromDegrees(p.Lat, p.Lng)
		ctx.AddObject(sm.NewImageMarker(pos, logo, offset, offset))
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
