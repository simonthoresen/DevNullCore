package main

import (
	"image"
	"image/color"
	"math"
)

// launcherScene draws a wireframe-like 3D tunnel/wormhole that the camera
// flies through. The tunnel centerline wobbles in (x, y) along a
// pseudo-Lissajous curve so the path bends and twists; each ring spins
// slowly around the path tangent so the longitudinal grid spirals.
//
// Rendering is a point cloud — for every (s, θ) sample we plot a single
// pixel into the backing RGBA image. With dense enough sampling the
// points merge visually into the grid lines you'd expect from a wireframe
// tube, and at the near end they remain visibly discrete (the retro
// "point cloud" look). The result is then quadrant-blocked by the
// launcher renderer through the shared client pipeline.
type launcherScene struct {
	img  *image.RGBA
	w, h int
}

func newLauncherScene() *launcherScene { return &launcherScene{} }

var (
	launcherBgColor = color.RGBA{R: 0x03, G: 0x05, B: 0x0d, A: 0xff}
	tunnelLineColor = color.RGBA{R: 180, G: 230, B: 240, A: 0xff}
)

// Render produces a w×h RGBA frame for the given elapsed time (seconds).
func (s *launcherScene) Render(w, h int, t float64) *image.RGBA {
	if w <= 0 || h <= 0 {
		return nil
	}
	if s.img == nil || s.w != w || s.h != h {
		s.img = image.NewRGBA(image.Rect(0, 0, w, h))
		s.w, s.h = w, h
	}
	s.fillBackground()
	s.drawTunnel(t)
	return s.img
}

func (s *launcherScene) fillBackground() {
	pix := s.img.Pix
	bg := launcherBgColor
	for i := 0; i < len(pix); i += 4 {
		pix[i+0] = bg.R
		pix[i+1] = bg.G
		pix[i+2] = bg.B
		pix[i+3] = bg.A
	}
}

func (s *launcherScene) drawTunnel(t float64) {
	const (
		pathSamples = 360
		numSegs     = 96

		nearD  = 1.0
		farD   = 32.0
		radius = 1.0

		// Path wobble (world-space curvature of the tunnel centerline).
		ax      = 1.4
		ay      = 1.0
		axOmega = 0.18
		ayOmega = 0.23

		flowSpeed = 0.06 // fraction of the path that flows toward camera per second
		twistRate = 0.12 // ring rotation per unit of path
		rollRate  = 0.20 // camera roll around its z axis (radians per second)
	)

	cx := float64(s.w) * 0.5
	cy := float64(s.h) * 0.5
	focal := math.Min(float64(s.w), float64(s.h)) * 0.55

	flow := math.Mod(t*flowSpeed, 1.0)
	if flow < 0 {
		flow += 1
	}

	span := farD - nearD
	camPhaseX := math.Sin(t * axOmega)
	camPhaseY := math.Sin(t * ayOmega)

	// Camera roll: rotate the whole projection around the screen center.
	rollAngle := t * rollRate
	rollCos := math.Cos(rollAngle)
	rollSin := math.Sin(rollAngle)

	for k := 0; k < pathSamples; k++ {
		// sFrac wraps as time advances, so each k slot drifts toward the
		// camera and reappears at the far end on wrap. With a point cloud
		// the wrap is invisible — points just pop in at the far horizon.
		sFrac := float64(k)/float64(pathSamples) - flow
		sFrac -= math.Floor(sFrac)
		d := nearD + sFrac*span

		// Path-coordinate of this ring (advances with time so the wobble
		// pattern translates and we feel like we're moving along the tube).
		sw := t + d

		// Tunnel centerline in camera space: difference between the wobble
		// at the ring's path coord and at the camera's path coord.
		cxw := ax * (math.Sin(sw*axOmega) - camPhaseX)
		cyw := ay * (math.Sin(sw*ayOmega) - camPhaseY)
		twist := sw * twistRate

		// Brightness/color falloff with depth.
		bright := 1.0 - sFrac
		bright = bright * bright
		col := scaleRGB(tunnelLineColor, 0.18+0.82*bright)

		invD := focal / d

		for j := 0; j < numSegs; j++ {
			a := 2*math.Pi*float64(j)/float64(numSegs) + twist
			wx := cxw + radius*math.Cos(a)
			wy := cyw + radius*math.Sin(a)
			// Apply camera roll around the optical axis before projection.
			rwx := wx*rollCos - wy*rollSin
			rwy := wx*rollSin + wy*rollCos
			px := cx + rwx*invD
			py := cy + rwy*invD
			s.plotPoint(int(px+0.5), int(py+0.5), col)
		}
	}
}

func (s *launcherScene) plotPoint(x, y int, c color.RGBA) {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	off := y*s.img.Stride + x*4
	p := s.img.Pix
	// "Max" blend so overlapping points stay neon-bright instead of
	// degrading to whatever color happens to be drawn last.
	if c.R > p[off+0] {
		p[off+0] = c.R
	}
	if c.G > p[off+1] {
		p[off+1] = c.G
	}
	if c.B > p[off+2] {
		p[off+2] = c.B
	}
	p[off+3] = 0xff
}

func scaleRGB(c color.RGBA, k float64) color.RGBA {
	return color.RGBA{
		R: clampByte(float64(c.R) * k),
		G: clampByte(float64(c.G) * k),
		B: clampByte(float64(c.B) * k),
		A: 0xff,
	}
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
