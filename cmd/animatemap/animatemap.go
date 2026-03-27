// Package main reads a journey.json file and renders an animated MP4 showing
// the journey building up stop by stop. For each new position the logo marker
// flies in (starts large, eases down to its final size) and bounces on
// landing. Frames are written to a temp directory and assembled into an
// H.264 MP4 via ffmpeg.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	stddraw "image/draw"
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
	imgWidth     = 900
	imgHeight    = 500
	fps          = 24
	flyInFrames  = 20 // frames for the fly-in shrink animation
	flyInOverlap = 5  // fly-in frames shared with the NEXT position's fly-in;
	//                      next position starts (flyInFrames-flyInOverlap) frames after current
	flyInScale    = 4.0  // starting size multiplier relative to final size
	bounceFrames  = 12   // frames for 3 post-landing bounces (~0.5 s)
	bounceAmp     = 0.25 // amplitude of each bounce (0.25 = 25% undershoot/overshoot)
	minHoldFrames = 6    // hold frames for a 1-day stay  (~0.25 s)
	maxHoldFrames = 48   // hold frames for a 30+ day stay (~2 s)
	finalHold     = 60   // frames for the completed map at the end (~2.5 s)
)


// Position mirrors the clustered stop structure written by cmd/journeymap.
type Position struct {
	Date string  `json:"date"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Days int     `json:"days"`
}

// JourneyMap mirrors the top-level structure written by cmd/journeymap.
type JourneyMap struct {
	Positions []Position `json:"positions"`
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
	return linearInterp(days, minSize, maxSize)
}

// holdFramesForDays linearly interpolates between minHoldFrames (1 day) and
// maxHoldFrames (≥30 days) so that longer stays feel proportionally longer
// in the animation.
func holdFramesForDays(days int) int {
	return linearInterp(days, minHoldFrames, maxHoldFrames)
}

// bounceMultiplier returns a size multiplier for bounce frame f out of total
// bounceFrames, producing nBounces oscillations that decay to 1 by the end.
// At f=0 the multiplier is 1 (seamlessly joins the end of the fly-in).
// At f=bounceFrames the multiplier is also 1 (fully settled).
// Between those points the logo squishes below and springs above its final
// size nBounces times, with linearly decreasing amplitude.
//
//	size(f) = finalSize × bounceMultiplier(f, ...)
func bounceMultiplier(f, total, nBounces int, amplitude float64) float64 {
	if total <= 0 {
		return 1
	}
	t := float64(f) / float64(total)
	// nBounces half-cycles of sine give nBounces visible squish/spring events.
	// (1-t) damps amplitude linearly to zero so the logo settles cleanly.
	dampedSine := math.Sin(float64(nBounces)*math.Pi*t) * (1 - t)
	return 1 - amplitude*dampedSine
}

// linearInterp maps days in [1, 30] to [minVal, maxVal], clamping outside that range.
func linearInterp(days, minVal, maxVal int) int {
	const minDays, maxDays = 1, 30
	if days <= minDays {
		return minVal
	}
	if days >= maxDays {
		return maxVal
	}
	t := float64(days-minDays) / float64(maxDays-minDays)
	return int(math.Round(float64(minVal) + t*float64(maxVal-minVal)))
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
	if len(positions) == 0 {
		return 0, 0, 1
	}
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

// drawMarker composites a pre-scaled logo image centred at pixel (px, py)
// onto frame using alpha compositing.
func drawMarker(frame *image.RGBA, scaled image.Image, px, py float64) {
	b := scaled.Bounds()
	dx := int(math.Round(px)) - b.Dx()/2
	dy := int(math.Round(py)) - b.Dy()/2
	dest := image.Rect(dx, dy, dx+b.Dx(), dy+b.Dy())
	stddraw.Draw(frame, dest, scaled, b.Min, stddraw.Over)
}

// positionStartFrames returns the global frame at which each position's fly-in
// begins. Consecutive positions are staggered by (flyInFrames - flyInOverlap)
// so that their fly-in animations partially overlap.
func positionStartFrames(positions []Position) []int {
	starts := make([]int, len(positions))
	if len(positions) == 0 {
		return starts
	}
	offset := flyInFrames - flyInOverlap
	for i := 1; i < len(positions); i++ {
		starts[i] = starts[i-1] + offset
	}
	return starts
}

// totalFrames returns the total number of frames the animation will produce.
// Useful for progress reporting and testing.
func totalFrames(positions []Position) int {
	if len(positions) == 0 {
		return finalHold
	}
	starts := positionStartFrames(positions)
	last := len(positions) - 1
	lastEnd := starts[last] + flyInFrames + bounceFrames + holdFramesForDays(positions[last].Days)
	return lastEnd + finalHold
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

	// Each position's fly-in starts flyInOverlap frames before the current
	// position's fly-in ends, so multiple markers animate simultaneously.
	starts := positionStartFrames(journey.Positions)

	tmpDir, err := os.MkdirTemp("", "animatemap_*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	total := totalFrames(journey.Positions)
	frameIdx := 0

	writeFrame := func(img *image.RGBA) error {
		if frameIdx%30 == 0 {
			fmt.Printf("  frame %d / %d\n", frameIdx, total)
		}
		path := filepath.Join(tmpDir, fmt.Sprintf("frame_%04d.png", frameIdx))
		frameIdx++
		return gg.SavePNG(path, img)
	}

	frame := image.NewRGBA(baseMap.Bounds())
	scaledCache := make(map[int]*image.RGBA)
	cachedScale := func(size int) *image.RGBA {
		if img, ok := scaledCache[size]; ok {
			return img
		}
		img := scaleImage(logo, size)
		scaledCache[size] = img
		return img
	}

	for globalF := 0; globalF < total; globalF++ {
		stddraw.Draw(frame, frame.Bounds(), baseMap, baseMap.Bounds().Min, stddraw.Src)

		// Draw each position in arrival order (earlier positions rendered below
		// later ones so newly arriving markers appear on top).
		for i, start := range starts {
			localF := globalF - start
			if localF < 0 {
				continue // not started yet
			}
			st := states[i]
			animLen := flyInFrames + bounceFrames + holdFramesForDays(journey.Positions[i].Days)

			var size int
			switch {
			case localF >= animLen:
				// Animation finished: stays at final size for the rest of the video.
				drawMarker(frame, st.finalLogo, st.px, st.py)
				continue
			case localF < flyInFrames:
				// Fly-in: ease-in quadratic (starts slow, ends fast — gravity-like)
				// so the logo arrives with velocity and flows directly into the bounce.
				t := 1.0
				if flyInFrames > 1 {
					t = float64(localF) / float64(flyInFrames-1)
				}
				scale := flyInScale - t*t*(flyInScale-1)
				size = int(math.Round(float64(st.finalSize) * scale))
			case localF < flyInFrames+bounceFrames:
				// Bounce: 3 damped oscillations at final size.
				mult := bounceMultiplier(localF-flyInFrames, bounceFrames, 3, bounceAmp)
				size = int(math.Round(float64(st.finalSize) * mult))
			default:
				// Hold: static at final size.
				drawMarker(frame, st.finalLogo, st.px, st.py)
				continue
			}

			if size < 1 {
				size = 1
			}
			drawMarker(frame, cachedScale(size), st.px, st.py)
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
