package main

import (
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ---- mercatorY ----

func TestMercatorY_Equator(t *testing.T) {
	got := mercatorY(0)
	if math.Abs(got-0.5) > 1e-9 {
		t.Errorf("mercatorY(0) = %v, want 0.5", got)
	}
}

func TestMercatorY_Monotonicity(t *testing.T) {
	// Higher latitude → smaller y (north is up).
	if mercatorY(45) >= mercatorY(0) {
		t.Errorf("mercatorY(45) should be < mercatorY(0)")
	}
	if mercatorY(-45) <= mercatorY(0) {
		t.Errorf("mercatorY(-45) should be > mercatorY(0)")
	}
}

func TestMercatorY_Symmetry(t *testing.T) {
	// mercatorY(-lat) should be symmetric around 0.5.
	lat := 45.0
	y1 := mercatorY(lat)
	y2 := mercatorY(-lat)
	if math.Abs((y1+y2)-1.0) > 1e-9 {
		t.Errorf("mercatorY(%v) + mercatorY(%v) = %v, want 1.0", lat, -lat, y1+y2)
	}
}

// ---- latLngToPixel ----

func TestLatLngToPixel_CenterMapsToImageCenter(t *testing.T) {
	centerLat, centerLng := 45.0, 13.0
	zoom := 7
	x, y := latLngToPixel(centerLat, centerLng, zoom, centerLat, centerLng, imgWidth, imgHeight)
	if math.Abs(x-float64(imgWidth)/2) > 0.5 {
		t.Errorf("x = %v, want %v (image centre)", x, float64(imgWidth)/2)
	}
	if math.Abs(y-float64(imgHeight)/2) > 0.5 {
		t.Errorf("y = %v, want %v (image centre)", y, float64(imgHeight)/2)
	}
}

func TestLatLngToPixel_EastIsRight(t *testing.T) {
	centerLat, centerLng := 0.0, 0.0
	zoom := 5
	x1, _ := latLngToPixel(0, 0, zoom, centerLat, centerLng, imgWidth, imgHeight)
	x2, _ := latLngToPixel(0, 10, zoom, centerLat, centerLng, imgWidth, imgHeight)
	if x2 <= x1 {
		t.Errorf("east point should have larger x: x(0°)=%v, x(10°E)=%v", x1, x2)
	}
}

func TestLatLngToPixel_NorthIsUp(t *testing.T) {
	centerLat, centerLng := 0.0, 0.0
	zoom := 5
	_, y1 := latLngToPixel(0, 0, zoom, centerLat, centerLng, imgWidth, imgHeight)
	_, y2 := latLngToPixel(10, 0, zoom, centerLat, centerLng, imgWidth, imgHeight)
	if y2 >= y1 {
		t.Errorf("north point should have smaller y: y(0°)=%v, y(10°N)=%v", y1, y2)
	}
}

// ---- chooseBoundsAndZoom ----

func TestChooseBoundsAndZoom_SinglePoint(t *testing.T) {
	positions := []Position{{Lat: 45.5, Lng: 13.6, Days: 10}}
	lat, lng, zoom := chooseBoundsAndZoom(positions, imgWidth, imgHeight)
	if math.Abs(lat-45.5) > 1e-9 || math.Abs(lng-13.6) > 1e-9 {
		t.Errorf("centre = (%.4f, %.4f), want (45.5, 13.6)", lat, lng)
	}
	if zoom < 1 || zoom > 15 {
		t.Errorf("zoom = %d, want 1–15", zoom)
	}
}

func TestChooseBoundsAndZoom_AllPositionsFitWithPadding(t *testing.T) {
	positions := []Position{
		{Lat: 45.5, Lng: 13.6, Days: 10},
		{Lat: 43.5, Lng: 16.4, Days: 5},
		{Lat: 44.8, Lng: 14.0, Days: 3},
	}
	const padding = 80.0
	centerLat, centerLng, zoom := chooseBoundsAndZoom(positions, imgWidth, imgHeight)
	for _, p := range positions {
		x, y := latLngToPixel(p.Lat, p.Lng, zoom, centerLat, centerLng, imgWidth, imgHeight)
		if x < padding || x > float64(imgWidth)-padding {
			t.Errorf("position (%.4f,%.4f) x=%v outside padding at zoom %d", p.Lat, p.Lng, x, zoom)
		}
		if y < padding || y > float64(imgHeight)-padding {
			t.Errorf("position (%.4f,%.4f) y=%v outside padding at zoom %d", p.Lat, p.Lng, y, zoom)
		}
	}
}

func TestChooseBoundsAndZoom_CentreIsArithmeticMidpoint(t *testing.T) {
	positions := []Position{
		{Lat: 40.0, Lng: 10.0},
		{Lat: 50.0, Lng: 20.0},
	}
	lat, lng, _ := chooseBoundsAndZoom(positions, imgWidth, imgHeight)
	if math.Abs(lat-45.0) > 1e-9 {
		t.Errorf("centre lat = %v, want 45.0", lat)
	}
	if math.Abs(lng-15.0) > 1e-9 {
		t.Errorf("centre lng = %v, want 15.0", lng)
	}
}

// ---- markerSize ----

func TestMarkerSize_Endpoints(t *testing.T) {
	if markerSize(1) != 30 {
		t.Errorf("markerSize(1) = %d, want 30", markerSize(1))
	}
	if markerSize(30) != 100 {
		t.Errorf("markerSize(30) = %d, want 100", markerSize(30))
	}
}

func TestMarkerSize_Clamped(t *testing.T) {
	if markerSize(0) != 30 {
		t.Errorf("markerSize(0) = %d, want 30 (clamped)", markerSize(0))
	}
	if markerSize(1000) != 100 {
		t.Errorf("markerSize(1000) = %d, want 100 (clamped)", markerSize(1000))
	}
}

func TestMarkerSize_MonotonicallyNonDecreasing(t *testing.T) {
	prev := markerSize(1)
	for days := 2; days <= 30; days++ {
		curr := markerSize(days)
		if curr < prev {
			t.Errorf("markerSize(%d)=%d < markerSize(%d)=%d", days, curr, days-1, prev)
		}
		prev = curr
	}
}

// ---- bounceMultiplier ----

func TestBounceMultiplier_Endpoints(t *testing.T) {
	// At f=0 the multiplier must be 1 (seamless join with end of fly-in).
	if got := bounceMultiplier(0, 12, 3, bounceAmp); math.Abs(got-1) > 1e-9 {
		t.Errorf("bounceMultiplier(f=0) = %v, want 1", got)
	}
	// At f=total the multiplier must be 1 (fully settled).
	if got := bounceMultiplier(12, 12, 3, bounceAmp); math.Abs(got-1) > 1e-9 {
		t.Errorf("bounceMultiplier(f=total) = %v, want 1", got)
	}
}

func TestBounceMultiplier_ThreeExcursions(t *testing.T) {
	// With nBounces=3 there must be 3 distinct regions where the multiplier
	// deviates noticeably from 1. We use a region-counting approach because
	// with 12 frames and 3 half-cycles the sine hits exactly 0 at the
	// crossing frames, so a strict sign-change check would miss them.
	const (
		total     = 12
		amp       = 0.25
		threshold = 0.02 // minimum deviation to count as a bounce
	)
	inExcursion := false
	excursions := 0
	for f := 0; f <= total; f++ {
		if math.Abs(bounceMultiplier(f, total, 3, amp)-1) > threshold {
			if !inExcursion {
				excursions++
				inExcursion = true
			}
		} else {
			inExcursion = false
		}
	}
	if excursions < 3 {
		t.Errorf("expected ≥3 bounce excursions, got %d", excursions)
	}
}

func TestBounceMultiplier_DecayingAmplitude(t *testing.T) {
	// The magnitude of the excursion should decrease over time (damping).
	const (
		total = 24
		amp   = 0.25
	)
	firstPeak := math.Abs(bounceMultiplier(total/6, total, 3, amp) - 1)
	lastPeak := math.Abs(bounceMultiplier(5*total/6, total, 3, amp) - 1)
	if firstPeak <= lastPeak {
		t.Errorf("bounce amplitude should decrease: first=%v, last=%v", firstPeak, lastPeak)
	}
}

// ---- holdFramesForDays ----

func TestHoldFramesForDays_Endpoints(t *testing.T) {
	if holdFramesForDays(1) != minHoldFrames {
		t.Errorf("holdFramesForDays(1) = %d, want %d", holdFramesForDays(1), minHoldFrames)
	}
	if holdFramesForDays(30) != maxHoldFrames {
		t.Errorf("holdFramesForDays(30) = %d, want %d", holdFramesForDays(30), maxHoldFrames)
	}
}

func TestHoldFramesForDays_Clamped(t *testing.T) {
	if holdFramesForDays(0) != minHoldFrames {
		t.Errorf("holdFramesForDays(0) = %d, want %d (clamped)", holdFramesForDays(0), minHoldFrames)
	}
	if holdFramesForDays(1000) != maxHoldFrames {
		t.Errorf("holdFramesForDays(1000) = %d, want %d (clamped)", holdFramesForDays(1000), maxHoldFrames)
	}
}

func TestHoldFramesForDays_MonotonicallyNonDecreasing(t *testing.T) {
	prev := holdFramesForDays(1)
	for days := 2; days <= 30; days++ {
		curr := holdFramesForDays(days)
		if curr < prev {
			t.Errorf("holdFramesForDays(%d)=%d < holdFramesForDays(%d)=%d", days, curr, days-1, prev)
		}
		prev = curr
	}
}

// ---- positionStartFrames ----

func TestPositionStartFrames_Empty(t *testing.T) {
	starts := positionStartFrames([]Position{})
	if len(starts) != 0 {
		t.Errorf("positionStartFrames(empty) len = %d, want 0", len(starts))
	}
}

func TestPositionStartFrames_Single(t *testing.T) {
	starts := positionStartFrames([]Position{{Days: 1}})
	if len(starts) != 1 || starts[0] != 0 {
		t.Errorf("positionStartFrames(single) = %v, want [0]", starts)
	}
}

func TestPositionStartFrames_Offset(t *testing.T) {
	positions := []Position{{Days: 1}, {Days: 1}, {Days: 1}}
	starts := positionStartFrames(positions)
	offset := flyInFrames - flyInOverlap
	for i, got := range starts {
		if want := i * offset; got != want {
			t.Errorf("starts[%d] = %d, want %d", i, got, want)
		}
	}
}

func TestPositionStartFrames_OverlapLessThanFlyIn(t *testing.T) {
	// Each position must start before the previous one's fly-in ends.
	positions := []Position{{Days: 1}, {Days: 1}}
	starts := positionStartFrames(positions)
	if starts[1] >= flyInFrames {
		t.Errorf("starts[1]=%d should be < flyInFrames=%d for overlap to occur", starts[1], flyInFrames)
	}
}

// ---- totalFrames ----

func TestTotalFrames(t *testing.T) {
	// A single position is unaffected by overlap logic.
	single := []Position{{Days: 1}}
	want := flyInFrames + bounceFrames + holdFramesForDays(1) + finalHold
	if got := totalFrames(single); got != want {
		t.Errorf("totalFrames(1-day) = %d, want %d", got, want)
	}

	empty := []Position{}
	if got := totalFrames(empty); got != finalHold {
		t.Errorf("totalFrames(empty) = %d, want %d", got, finalHold)
	}

	// With overlap, two positions take fewer frames than 2× a single position.
	twoPositions := []Position{{Days: 1}, {Days: 1}}
	if got := totalFrames(twoPositions); got >= 2*totalFrames(single) {
		t.Errorf("overlapping two positions should take fewer frames than 2× single: got %d, 2×single=%d",
			got, 2*totalFrames(single))
	}

	// Longer stays produce more total frames than shorter ones (hold dominates).
	short := []Position{{Days: 1}, {Days: 1}}
	long := []Position{{Days: 30}, {Days: 30}}
	if totalFrames(short) >= totalFrames(long) {
		t.Errorf("long stays should produce more frames: short=%d, long=%d",
			totalFrames(short), totalFrames(long))
	}
}

// ---- scaleImage ----

func TestScaleImage_OutputSize(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for _, size := range []int{30, 50, 80, 100} {
		got := scaleImage(src, size)
		b := got.Bounds()
		if b.Dx() != size || b.Dy() != size {
			t.Errorf("scaleImage(src, %d) size = %dx%d, want %dx%d", size, b.Dx(), b.Dy(), size, size)
		}
	}
}

// ---- cloneImage ----

func TestCloneImage_IsIndependent(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	src.SetRGBA(0, 0, struct{ R, G, B, A uint8 }{255, 0, 0, 255})
	clone := cloneImage(src)
	clone.SetRGBA(0, 0, struct{ R, G, B, A uint8 }{0, 255, 0, 255})
	// Mutating the clone must not affect src.
	if src.RGBAAt(0, 0).R != 255 {
		t.Error("cloneImage: modifying clone affected source")
	}
}

// ---- generateAnimation integration ----

func TestGenerateAnimation_SkipsIfFFmpegMissing(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		t.Skip("ffmpeg is available; skipping no-ffmpeg path")
	}
	journey := JourneyMap{
		Positions: []Position{{Lat: 45.5, Lng: 13.6, Days: 5}},
		Route:     []LatLng{{Lat: 45.5, Lng: 13.6}},
	}
	err := generateAnimation(journey, filepath.Join(t.TempDir(), "out.mp4"))
	if err == nil {
		t.Error("expected error when ffmpeg is missing, got nil")
	}
}

func TestGenerateAnimation_ProducesFile(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "journey.mp4")

	journey := JourneyMap{
		Positions: []Position{
			{Date: "2025-09-13", Lat: 45.5127, Lng: 13.5954, Days: 10},
			{Date: "2026-01-17", Lat: 43.5088, Lng: 16.4402, Days: 5},
		},
		Route: []LatLng{
			{Lat: 45.5127, Lng: 13.5954},
			{Lat: 43.5088, Lng: 16.4402},
		},
	}

	if err := generateAnimation(journey, outputPath); err != nil {
		t.Fatalf("generateAnimation() error: %v", err)
	}

	info, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		t.Fatal("generateAnimation() did not create output file")
	}
	if info.Size() == 0 {
		t.Error("generateAnimation() created an empty file")
	}
}
