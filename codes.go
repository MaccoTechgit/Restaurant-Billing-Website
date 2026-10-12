package main

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/skip2/go-qrcode"
)

// Real, scannable Code 128 barcodes and QR codes for the website's sample bills (drawn as inline SVG).

var c128 = strings.Split("212222,222122,222221,121223,121322,131222,122213,122312,132212,221213,221312,231212,112232,122132,122231,113222,123122,123221,223211,221132,221231,213212,223112,312131,311222,321122,321221,312212,322112,322211,212123,212321,232121,111323,131123,131321,112313,132113,132311,211313,231113,231311,112133,112331,132131,113123,113321,133121,313121,211331,231131,213113,213311,213131,311123,311321,331121,312113,312311,332111,314111,221411,431111,111224,111422,121124,121421,141122,141221,112214,112412,122114,122411,142112,142211,241211,221114,413111,241112,134111,111242,121142,121241,114212,124112,124211,411212,421112,421211,212141,214121,412121,111143,111341,131141,114113,114311,411113,411311,113141,114131,311141,411131,211412,211214,211232,2331112", ",")

func barcodeSVG(text string, h int) template.HTML {
	codes := []int{104}
	sum := 104
	for i, r := range text {
		if r < 32 || r > 126 {
			continue
		}
		v := int(r) - 32
		codes = append(codes, v)
		sum += v * (i + 1)
	}
	codes = append(codes, sum%103, 106)
	var b strings.Builder
	x := 10
	for _, k := range codes {
		bar := true
		for _, ch := range c128[k] {
			w := int(ch - '0')
			if bar {
				fmt.Fprintf(&b, `<rect x="%d" width="%d" height="%d"/>`, x, w, h)
			}
			x += w
			bar = !bar
		}
	}
	return template.HTML(fmt.Sprintf(`<svg class="bcode" viewBox="0 0 %d %d" preserveAspectRatio="none" shape-rendering="crispEdges" fill="#0a0f1e" role="img" aria-label="Barcode %s">%s</svg>`, x+10, h, template.HTMLEscapeString(text), b.String()))
}

func qrSVG(text string) template.HTML {
	q, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return ""
	}
	bm := q.Bitmap() // includes the quiet zone
	n := len(bm)
	var d strings.Builder
	for y, row := range bm {
		for x := 0; x < n; {
			if !row[x] {
				x++
				continue
			}
			s := x
			for x < n && row[x] {
				x++
			}
			fmt.Fprintf(&d, "M%d %dh%dv1h-%dz", s, y, x-s, x-s)
		}
	}
	return template.HTML(fmt.Sprintf(`<svg class="qrc" viewBox="0 0 %d %d" shape-rendering="crispEdges" role="img" aria-label="QR code"><rect width="%d" height="%d" fill="#fff"/><path d="%s" fill="#0a0f1e"/></svg>`, n, n, n, n, d.String()))
}
