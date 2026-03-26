// Package main reads a journey.json file and renders an animated MP4 showing
// the journey building up stop by stop. For each new position the route line
// reveals up to that stop and the logo marker flies in (starts large, eases
// down to its final size). Frames are written to a temp directory and assembled
// into an H.264 MP4 via ffmpeg.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	stddraw "image/draw"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	sm "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
	xdraw "golang.org/x/image/draw"
)

//go:embed logo.png
var logoBytes []byte

// Animation timing (all in frames at fps).
const (
	imgWidth    = 900
	imgHeight   = 500
	fps         = 24
	flyInFrames = 15   // frames for the fly-in shrink animation
	flyInScale  = 4.0  // starting size multiplier relative to final size
	holdFrames  = 10   // frames to hold after each fly-in
	finalHold   = 60   // frames for the completed map at the end (~2.5 s)
)

var routeColor = color.RGBA{R: 100, G: 100, B: 100, A: 180}

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
		fmt.Fprintf(os.Stderr, "Usage: %s <journey-json-path> <output-mp4-path>\n", os.Args[0])
		os.Exit(1)
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: ffmpeg not found on $PATH. Install it with: brew install ffmpeg")
		os.Exit(1)
	}

	journey, err := loadJourneyMap(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading journey map: %v\n", err)
		os.Exit(1)
	}

	if len(journey.Positions) == 0 {
		fmt.Println("No positions found, skipping animation")
		return
	}

	if err := generateAnimation(journey, os.Args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating animation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Journey animation written to %s\n", os.Args[2])
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

// markerSize linearly interpolates between 30px (1 day) and 100px (≥30 days),
// matching the sizing used by cmd/rendermap.
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

// mercatorY converts latitude (degrees) to a Web Mercator y coordinate in [0,1].
// y=0 is the north pole, y=1 is the south pole.
func mercatorY(lat float64) float64 {
	latRad := lat * math.Pi / 180
	return (1 - math.Log(math.Tan(latRad)+1/math.Cos(latRad))/math.Pi) / 2
}

// latLngToPixel converts a lat/lng coordinate to pixel (x, y) on the rendered
// map image given the explicit zoom level and centre point.
func latLngToPixel(lat, lng float64, zoom int, centerLat, centerLng float64, imgW, imgH int) (float64, float64) {
	worldSize := 256.0 * math.Pow(2, float64(zoom))
	x := (lng+180)/360*worldSize - (centerLng+180)/360*worldSize + float64(imgW)/2
	y := mercatorY(lat)*worldSize - mercatorY(centerLat)*worldSize + float64(imgH)/2
	return x, y
}

// chooseBoundsAndZoom computes the arithmetic centre of all positions and
// selects the largest zoom level (up to 15) where every position falls at
// least padding pixels inside the image edges.
func chooseBoundsAndZoom(positions []Position, imgW, imgH int) (centerLat, centerLng float64, zoom int) {
	minLat, maxLat := positions[0].Lat, positions[0].Lat
	minLng, maxLng := positions[0].Lng, positions[0].Lng
	for _, p := range positions[1:] {
		if p.Lat < minLat {
			minLat = p.Lat
		}
		if p.Lat > maxLat {
			maxLat = p.Lat
		}
		if p.Lng < minLng {
			minLng = p.Lng
		}
		if p.Lng > maxLng {
			maxLng = p.Lng
		}
	}
	centerLat = (minLat + maxLat) / 2
	centerLng = (minLng + maxLng) / 2

	const padding = 80.0
	for z := 15; z >= 1; z-- {
		allFit := true
		for _, p := range positions {
			x, y := latLngToPixel(p.Lat, p.Lng, z, centerLat, centerLng, imgW, imgH)
			if x < padding || x > float64(imgW)-padding || y < padding || y > float64(imgH)-padding {
				allFit = false
				break
			}
		}
		if allFit {
			return centerLat, centerLng, z
		}
	}
	return centerLat, centerLng, 1
}

// renderBaseMap fetches OSM tiles for the given centre/zoom and returns the
// resulting image with no markers or routes drawn on it.
func renderBaseMap(centerLat, centerLng float64, zoom int) (image.Image, error) {
	ctx := sm.NewContext()
	ctx.SetSize(imgWidth, imgHeight)
	ctx.SetZoom(zoom)
	ctx.SetCenter(s2.LatLngFromDegrees(centerLat, centerLng))
	ctx.OverrideAttribution("")
	return ctx.Render()
}

// cloneImage returns a mutable *image.RGBA copy of src.
func cloneImage(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	stddraw.Draw(dst, b, src, b.Min, stddraw.Src)
	return dst
}

// scaleImage scales src to a square of the given pixel size using the
// Catmull-Rom bicubic interpolator, which preserves sharpness and detail
// better than bilinear when downscaling the high-resolution logo.
func scaleImage(src image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

// drawRoute renders the route polyline from route[0] to route[upTo-1] onto
// frame using an antialiased gg context.
func drawRoute(frame *image.RGBA, route []LatLng, upTo int, zoom int, centerLat, centerLng float64) {
	if upTo < 2 {
		return
	}
	dc := gg.NewContextForRGBA(frame)
	dc.SetRGBA(
		float64(routeColor.R)/255,
		float64(routeColor.G)/255,
		float64(routeColor.B)/255,
		float64(routeColor.A)/255,
	)
	dc.SetLineWidth(2)
	x0, y0 := latLngToPixel(route[0].Lat, route[0].Lng, zoom, centerLat, centerLng, imgWidth, imgHeight)
	dc.MoveTo(x0, y0)
	for i := 1; i < upTo; i++ {
		xi, yi := latLngToPixel(route[i].Lat, route[i].Lng, zoom, centerLat, centerLng, imgWidth, imgHeight)
		dc.LineTo(xi, yi)
	}
	dc.Stroke()
}

// drawMarker composites a pre-scaled logo image centred at pixel (px, py)
// onto frame using alpha compositing.
func drawMarker(frame *image.RGBA, scaled image.Image, px, py float64) {
	b := scaled.Bounds()
	dx := int(math.Round(px)) - b.Dx()/2
	dy := int(math.Round(py)) - b.Dy()/2
	dest := image.Rect(dx, dy, dx+b.Dx(), dy+b.Dy())
	stddraw.Draw(frame, dest, scaled, b.Min, stddraw.Over)
}

// positionRouteIndex returns the first index in route where coordinates match
// pos within clusterThreshold degrees (the same tolerance used by journeymap).
// Returns the last index if no match is found.
const clusterThreshold = 0.01

func positionRouteIndex(pos Position, route []LatLng) int {
	for i, r := range route {
		if math.Abs(r.Lat-pos.Lat) <= clusterThreshold &&
			math.Abs(r.Lng-pos.Lng) <= clusterThreshold {
			return i
		}
	}
	return len(route) - 1
}

// totalFrames returns the total number of frames the animation will produce.
// Useful for progress reporting and testing.
func totalFrames(numPositions int) int {
	return numPositions*(flyInFrames+holdFrames) + finalHold
}

// markerState groups precomputed pixel position, final size, and
// pre-scaled logo image for a single journey position.
type markerState struct {
	px, py    float64
	finalSize int
	finalLogo *image.RGBA
}

func generateAnimation(journey JourneyMap, outputPath string) error {
	logo, _, err := image.Decode(bytes.NewReader(logoBytes))
	if err != nil {
		return fmt.Errorf("decoding embedded logo: %w", err)
	}

	centerLat, centerLng, zoom := chooseBoundsAndZoom(journey.Positions, imgWidth, imgHeight)

	fmt.Printf("Rendering base map (zoom %d, centre %.4f,%.4f)...\n", zoom, centerLat, centerLng)
	baseMap, err := renderBaseMap(centerLat, centerLng, zoom)
	if err != nil {
		return fmt.Errorf("rendering base map: %w", err)
	}

	// Precompute pixel positions and pre-scaled logos for all positions.
	states := make([]markerState, len(journey.Positions))
	for i, p := range journey.Positions {
		px, py := latLngToPixel(p.Lat, p.Lng, zoom, centerLat, centerLng, imgWidth, imgHeight)
		fs := markerSize(p.Days)
		states[i] = markerState{
			px:        px,
			py:        py,
			finalSize: fs,
			finalLogo: scaleImage(logo, fs),
		}
	}

	// Find where each position first appears in the route.
	routeIndices := make([]int, len(journey.Positions))
	for i, p := range journey.Positions {
		routeIndices[i] = positionRouteIndex(p, journey.Route)
	}

	tmpDir, err := os.MkdirTemp("", "animatemap_*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	total := totalFrames(len(journey.Positions))
	frameIdx := 0

	writeFrame := func(img *image.RGBA) error {
		if frameIdx%30 == 0 {
			fmt.Printf("  frame %d / %d\n", frameIdx, total)
		}
		path := filepath.Join(tmpDir, fmt.Sprintf("frame_%04d.png", frameIdx))
		frameIdx++
		return gg.SavePNG(path, img)
	}

	routeShown := 0

	for eventIdx := range journey.Positions {
		routeShown = routeIndices[eventIdx] + 1
		st := states[eventIdx]

		// Fly-in: logo shrinks from flyInScale × finalSize to finalSize.
		for f := 0; f < flyInFrames; f++ {
			t := float64(f) / float64(flyInFrames-1) // 0→1
			easedT := 1 - (1-t)*(1-t)               // ease-out quadratic
			scale := flyInScale - easedT*(flyInScale-1)
			size := int(math.Round(float64(st.finalSize) * scale))
			if size < 1 {
				size = 1
			}

			frame := cloneImage(baseMap)
			drawRoute(frame, journey.Route, routeShown, zoom, centerLat, centerLng)
			for j := 0; j < eventIdx; j++ {
				drawMarker(frame, states[j].finalLogo, states[j].px, states[j].py)
			}
			drawMarker(frame, scaleImage(logo, size), st.px, st.py)
			if err := writeFrame(frame); err != nil {
				return err
			}
		}

		// Hold: all positions up to and including eventIdx at final size.
		for f := 0; f < holdFrames; f++ {
			frame := cloneImage(baseMap)
			drawRoute(frame, journey.Route, routeShown, zoom, centerLat, centerLng)
			for j := 0; j <= eventIdx; j++ {
				drawMarker(frame, states[j].finalLogo, states[j].px, states[j].py)
			}
			if err := writeFrame(frame); err != nil {
				return err
			}
		}
	}

	// Final hold: everything including any remaining route after the last position.
	for f := 0; f < finalHold; f++ {
		frame := cloneImage(baseMap)
		drawRoute(frame, journey.Route, len(journey.Route), zoom, centerLat, centerLng)
		for j := range journey.Positions {
			drawMarker(frame, states[j].finalLogo, states[j].px, states[j].py)
		}
		if err := writeFrame(frame); err != nil {
			return err
		}
	}

	fmt.Printf("Assembling %d frames into %s...\n", frameIdx, outputPath)
	return runFFmpeg(tmpDir, outputPath)
}

func runFFmpeg(framesDir, outputPath string) error {
	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	pattern := filepath.Join(framesDir, "frame_%04d.png")
	cmd := exec.Command("ffmpeg",
		"-y",
		"-framerate", fmt.Sprintf("%d", fps),
		"-i", pattern,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-crf", "23",
		outputPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\n%s", err, string(out))
	}
	return nil
}
