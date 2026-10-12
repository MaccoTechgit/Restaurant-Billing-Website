package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"sync"
)

// The MacBill logo, drawn from code: an orange tile, a bold "M", and a torn-receipt edge underneath.
// The same shape is produced as SVG (favicon.svg) and as PNG/ICO at every size browsers and Google ask for.

// markSVG is the vector logo (also used inline on the website through <symbol id="mb">).
const markInner = `<defs><linearGradient id="mbg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#ff9d52"/><stop offset="1" stop-color="#e8590c"/></linearGradient></defs>` +
	`<rect width="32" height="32" rx="9" fill="url(#mbg)"/>` +
	`<path d="M9 19.5V9l7 6.5L23 9v10.5" fill="none" stroke="#fff" stroke-width="2.9" stroke-linecap="round" stroke-linejoin="round"/>` +
	`<path d="M9 24.2l2-1.7 2 1.7 2-1.7 2 1.7 2-1.7 2 1.7" fill="none" stroke="#fff" stroke-opacity=".62" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>`

func faviconSVG() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">` + markInner + `</svg>`
}

var (
	imgMu    sync.Mutex
	imgCache = map[string][]byte{}
)

func servePNG(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(b)
}

type rgb struct{ r, g, b float64 }

func mix(a, b rgb, t float64) rgb {
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}
func clamp(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
func blend(img *image.RGBA, x, y int, c rgb, a float64) {
	if a <= 0 || !(image.Point{X: x, Y: y}).In(img.Rect) {
		return
	}
	o := img.RGBAAt(x, y)
	f := func(old uint8, n float64) uint8 { return uint8(float64(old)*(1-a) + n*a + .5) }
	img.SetRGBA(x, y, color.RGBA{f(o.R, c.r), f(o.G, c.g), f(o.B, c.b), 255})
}

var (
	navy   = rgb{7, 11, 22}
	navy2  = rgb{27, 31, 70}
	orange = rgb{249, 115, 22}
	white  = rgb{255, 255, 255}
	grayB  = rgb{229, 232, 239}
	ink    = rgb{10, 15, 30}
)

// roundCov is the anti-aliased coverage of a rounded rectangle at pixel (x,y).
func roundCov(x, y int, x0, y0, w, h, r float64) float64 {
	px, py := float64(x)+.5-(x0+w/2), float64(y)+.5-(y0+h/2)
	qx, qy := math.Abs(px)-(w/2-r), math.Abs(py)-(h/2-r)
	d := math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
	return clamp(.5 - d)
}

func fillRound(img *image.RGBA, x0, y0, w, h, r float64, c rgb, alpha float64) {
	for y := int(y0) - 1; y <= int(y0+h)+1; y++ {
		for x := int(x0) - 1; x <= int(x0+w)+1; x++ {
			blend(img, x, y, c, roundCov(x, y, x0, y0, w, h, r)*alpha)
		}
	}
}

func glow(img *image.RGBA, cx, cy, rad float64, c rgb, strength float64) {
	for y := int(cy - rad); y < int(cy+rad); y++ {
		for x := int(cx - rad); x < int(cx+rad); x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / rad
			if d < 1 {
				blend(img, x, y, c, (1-d)*(1-d)*strength)
			}
		}
	}
}

func segDist(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	t := clamp(((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

type pt struct{ x, y float64 }

// stroke draws a round-capped polyline given in the 32x32 logo grid.
func stroke(img *image.RGBA, pts []pt, ox, oy, scale, hw float64, c rgb, alpha float64) {
	x0, y0, x1, y1 := 1e9, 1e9, -1e9, -1e9
	for _, p := range pts {
		x0, y0, x1, y1 = math.Min(x0, p.x), math.Min(y0, p.y), math.Max(x1, p.x), math.Max(y1, p.y)
	}
	pad := hw + 1
	for y := int(oy + (y0-pad)*scale); y <= int(oy+(y1+pad)*scale); y++ {
		for x := int(ox + (x0-pad)*scale); x <= int(ox+(x1+pad)*scale); x++ {
			px, py := (float64(x)+.5-ox)/scale, (float64(y)+.5-oy)/scale
			best := 1e9
			for i := 0; i+1 < len(pts); i++ {
				best = math.Min(best, segDist(px, py, pts[i].x, pts[i].y, pts[i+1].x, pts[i+1].y))
			}
			blend(img, x, y, c, clamp((hw-best)*scale+.5)*alpha)
		}
	}
}

// drawMark paints the logo into a square of the given size at (ox,oy).
// detail=false (tiny sizes) drops the receipt edge and thickens the M; full=true fills the whole square (maskable icons);
// inner scales the artwork inside the tile.
func drawMark(img *image.RGBA, ox, oy, size float64, detail, full bool, inner float64) {
	s := size / 32
	r := 9 * s
	if full {
		r = 0
	}
	for y := int(oy) - 1; y <= int(oy+size)+1; y++ {
		for x := int(ox) - 1; x <= int(ox+size)+1; x++ {
			cov := roundCov(x, y, ox, oy, size, size, r)
			if cov <= 0 {
				continue
			}
			t := clamp(((float64(x) - ox) + (float64(y) - oy)) / (2 * size))
			c := mix(rgb{255, 157, 82}, rgb{232, 89, 12}, t)
			// soft top highlight
			c = mix(c, white, .10*clamp(1-(float64(y)-oy)/(size*.55)))
			blend(img, x, y, c, cov)
		}
	}
	tf := func(p pt) pt { return pt{16 + (p.x-16)*inner, 16 + (p.y-16)*inner} }
	hw := 1.45 * inner
	m := []pt{{9, 19.5}, {9, 9}, {16, 15.5}, {23, 9}, {23, 19.5}}
	if !detail {
		hw = 2.05 * inner
		m = []pt{{8.5, 22}, {8.5, 9.5}, {16, 17}, {23.5, 9.5}, {23.5, 22}}
	}
	for i := range m {
		m[i] = tf(m[i])
	}
	stroke(img, m, ox, oy, s, hw, white, 1)
	if detail {
		z := []pt{{9, 24.2}, {11, 22.5}, {13, 24.2}, {15, 22.5}, {17, 24.2}, {19, 22.5}, {21, 24.2}, {23, 22.5}}
		for i := range z {
			z[i] = tf(z[i])
		}
		stroke(img, z, ox, oy, s, .75*inner, white, .62)
	}
}

func square(size int, full bool, inner float64) []byte {
	key := fmt.Sprint("sq", size, full, inner)
	imgMu.Lock()
	defer imgMu.Unlock()
	if b, ok := imgCache[key]; ok {
		return b
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	if full { // opaque background for maskable/apple icons
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				img.SetRGBA(x, y, color.RGBA{255, 140, 60, 255})
			}
		}
	}
	drawMark(img, 0, 0, float64(size), size >= 40, full, inner)
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imgCache[key] = buf.Bytes()
	return imgCache[key]
}

// favicon.ico with 16, 32 and 48 pixel PNG images inside.
func faviconICO() []byte {
	imgMu.Lock()
	if b, ok := imgCache["ico"]; ok {
		imgMu.Unlock()
		return b
	}
	imgMu.Unlock()
	sizes := []int{16, 32, 48}
	var datas [][]byte
	for _, s := range sizes {
		datas = append(datas, square(s, false, 1))
	}
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})
	off := 6 + 16*len(sizes)
	for i, s := range sizes {
		buf.Write([]byte{byte(s), byte(s), 0, 0})
		binary.Write(&buf, binary.LittleEndian, [2]uint16{1, 32})
		binary.Write(&buf, binary.LittleEndian, [2]uint32{uint32(len(datas[i])), uint32(off)})
		off += len(datas[i])
	}
	for _, d := range datas {
		buf.Write(d)
	}
	imgMu.Lock()
	imgCache["ico"] = buf.Bytes()
	imgMu.Unlock()
	return buf.Bytes()
}

// 1200x630 social card.
func socialCard() []byte {
	imgMu.Lock()
	defer imgMu.Unlock()
	if b, ok := imgCache["og"]; ok {
		return b
	}
	img := image.NewRGBA(image.Rect(0, 0, 1200, 630))
	w, h := 1200.0, 630.0
	for y := 0; y < 630; y++ {
		for x := 0; x < 1200; x++ {
			blend(img, x, y, mix(navy, navy2, clamp((float64(x)/w)*.6+(float64(y)/h)*.4)), 1)
		}
	}
	glow(img, w*.78, h*.45, 420, orange, .38)
	glow(img, w*.12, h*1.05, 380, rgb{99, 102, 241}, .22)
	drawMark(img, 96, 96, 132, true, false, 1)
	fillRound(img, 96, 290, 430, 34, 17, white, .95)
	fillRound(img, 96, 346, 360, 34, 17, white, .95)
	fillRound(img, 96, 402, 250, 34, 17, orange, 1)
	fillRound(img, 96, 478, 300, 14, 7, white, .35)
	wx, wy, ww, wh := 640.0, 120.0, 460.0, 390.0
	fillRound(img, wx+6, wy+18, ww, wh, 26, rgb{0, 0, 0}, .35)
	fillRound(img, wx, wy, ww, wh, 26, white, 1)
	fillRound(img, wx, wy, ww, 44, 26, rgb{246, 247, 251}, 1)
	for i := 0; i < 3; i++ {
		fillRound(img, wx+22+float64(i)*22, wy+16, 12, 12, 6, grayB, 1)
	}
	for i := 0; i < 3; i++ {
		x := wx + 28 + float64(i)*140
		fillRound(img, x, wy+70, 128, 74, 14, rgb{246, 247, 251}, 1)
		fillRound(img, x+14, wy+84, 50, 8, 4, grayB, 1)
		fillRound(img, x+14, wy+104, 78, 18, 8, ink, 1)
	}
	for i, hh := range []float64{.35, .5, .42, .7, .9, 1, .78, .6} {
		bh := 130 * hh
		col := grayB
		if i%2 == 1 {
			col = rgb{251, 146, 60}
		}
		fillRound(img, wx+34+float64(i)*51, wy+360-bh, 34, bh, 8, col, 1)
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imgCache["og"] = buf.Bytes()
	return imgCache["og"]
}

func (s *Server) brandRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /og.png", func(w http.ResponseWriter, r *http.Request) { servePNG(w, socialCard()) })
	mux.HandleFunc("GET /logo.png", func(w http.ResponseWriter, r *http.Request) { servePNG(w, square(512, false, 1)) })
	mux.HandleFunc("GET /apple-touch-icon.png", func(w http.ResponseWriter, r *http.Request) { servePNG(w, square(180, true, .86)) })
	for _, n := range []int{48, 96, 144, 192, 512} {
		n := n
		mux.HandleFunc(fmt.Sprintf("GET /icon-%d.png", n), func(w http.ResponseWriter, r *http.Request) { servePNG(w, square(n, false, 1)) })
	}
	mux.HandleFunc("GET /icon-maskable-512.png", func(w http.ResponseWriter, r *http.Request) { servePNG(w, square(512, true, .66)) })
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(faviconICO())
	})
	mux.HandleFunc("GET /favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write([]byte(faviconSVG()))
	})
	mux.HandleFunc("GET /site.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		name := s.setting("company")
		m := map[string]any{"name": name, "short_name": name, "description": "Restaurant billing & management software", "start_url": "/", "display": "browser", "background_color": "#ffffff", "theme_color": "#f97316",
			"icons": []map[string]string{
				{"src": "/icon-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any"},
				{"src": "/icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"},
				{"src": "/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
			}}
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		json.NewEncoder(w).Encode(m)
	})
}
