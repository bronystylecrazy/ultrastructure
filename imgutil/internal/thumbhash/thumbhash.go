// Adapted from go.n16f.net/thumbhash (ISC license), with local optimizations.
package thumbhash

import (
	"image"
	"image/draw"
	"math"
	"sync"

	xdraw "golang.org/x/image/draw"
)

const (
	maxEncodeDim    = 128
	maxEncodePixels = maxEncodeDim * maxEncodeDim
)

var (
	rgbaPool = sync.Pool{
		New: func() any {
			return image.NewRGBA(image.Rect(0, 0, maxEncodeDim, maxEncodeDim))
		},
	}

	lpqaPool = sync.Pool{
		New: func() any {
			return &lpqaBuf{
				L: make([]float64, maxEncodePixels),
				P: make([]float64, maxEncodePixels),
				Q: make([]float64, maxEncodePixels),
				A: make([]float64, maxEncodePixels),
			}
		},
	}

	fxPool = sync.Pool{
		New: func() any {
			s := make([]float64, maxEncodeDim)
			return &s
		},
	}

	decodeCosPool = sync.Pool{
		New: func() any {
			return &decodeCosBuf{
				cosX: make([]float64, 32*5),
				cosY: make([]float64, 32*5),
			}
		},
	}
)

type lpqaBuf struct {
	L []float64
	P []float64
	Q []float64
	A []float64
}

type decodeCosBuf struct {
	cosX []float64
	cosY []float64
}

type DecodingCfg struct {
	BaseSize        int
	SaturationBoost float64
}

type EncodeMeta struct {
	Width  int
	Height int
	AvgR   float64
	AvgG   float64
	AvgB   float64
	AvgA   float64
}

func EncodeImage(img image.Image) []byte {
	hash, _ := EncodeImageWithMeta(img)
	return hash
}

func EncodeImageWithMeta(img image.Image) ([]byte, EncodeMeta) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	origW, origH := w, h

	rgba := rgbaPool.Get().(*image.RGBA)
	defer rgbaPool.Put(rgba)

	if maxDim := imax(w, h); maxDim > maxEncodeDim {
		var scaleFactor float64
		if w > h {
			scaleFactor = maxEncodeDim / float64(w)
		} else {
			scaleFactor = maxEncodeDim / float64(h)
		}
		w = int(float64(w) * scaleFactor)
		h = int(float64(h) * scaleFactor)
		xdraw.NearestNeighbor.Scale(rgba, image.Rect(0, 0, w, h), img, bounds, draw.Src, nil)
	} else {
		draw.Draw(rgba, image.Rect(0, 0, w, h), img, bounds.Min, draw.Src)
	}

	var avgR, avgG, avgB, avgA float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := rgba.PixOffset(x, y)
			a := float64(rgba.Pix[i+3]) / 255.0
			avgR += a / 255.0 * float64(rgba.Pix[i])
			avgG += a / 255.0 * float64(rgba.Pix[i+1])
			avgB += a / 255.0 * float64(rgba.Pix[i+2])
			avgA += a
		}
	}
	if avgA > 0.0 {
		avgR /= avgA
		avgG /= avgA
		avgB /= avgA
	}

	nbPixels := w * h
	lpqa := lpqaPool.Get().(*lpqaBuf)
	defer lpqaPool.Put(lpqa)

	hasAlpha := avgA < float64(nbPixels)
	lLimit := 7.0
	if hasAlpha {
		lLimit = 5.0
	}

	wf := float64(w)
	hf := float64(h)
	maxWH := math.Max(wf, hf)
	lx := imax(1, iround((lLimit*wf)/maxWH))
	ly := imax(1, iround((lLimit*hf)/maxWH))

	pixNum := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := rgba.PixOffset(x, y)
			a := float64(rgba.Pix[i+3]) / 255.0

			r := avgR*(1.0-a) + a/255.0*float64(rgba.Pix[i])
			g := avgG*(1.0-a) + a/255.0*float64(rgba.Pix[i+1])
			b := avgB*(1.0-a) + a/255.0*float64(rgba.Pix[i+2])

			lpqa.L[pixNum] = (r + g + b) / 3.0
			lpqa.P[pixNum] = (r+g)/2.0 - b
			lpqa.Q[pixNum] = r - g
			lpqa.A[pixNum] = a
			pixNum++
		}
	}

	fx := fxPool.Get().(*[]float64)
	defer fxPool.Put(fx)

	encodeChannel := func(channel []float64, nx, ny int) (dc float64, ac []float64, scale float64) {
		ac = make([]float64, channelACCount(nx, ny))
		acIdx := 0

		for cy := 0; cy < ny; cy++ {
			cyf := float64(cy)
			for cx := 0; cx*ny < nx*(ny-cy); cx++ {
				cxf := float64(cx)
				f := 0.0

				for x := 0; x < w; x++ {
					(*fx)[x] = math.Cos(math.Pi / wf * cxf * (float64(x) + 0.5))
				}

				for y := 0; y < h; y++ {
					fy := math.Cos(math.Pi / hf * cyf * (float64(y) + 0.5))
					for x := 0; x < w; x++ {
						f += channel[x+y*w] * (*fx)[x] * fy
					}
				}
				f /= float64(nbPixels)

				if cx > 0 || cy > 0 {
					ac[acIdx] = f
					acIdx++
					scale = math.Max(scale, math.Abs(f))
				} else {
					dc = f
				}
			}
		}

		if scale > 0.0 {
			for i := 0; i < len(ac); i++ {
				ac[i] = 0.5 + 0.5/scale*ac[i]
			}
		}
		return
	}

	lDC, lAC, lScale := encodeChannel(lpqa.L, imax(lx, 3), imax(ly, 3))
	pDC, pAC, pScale := encodeChannel(lpqa.P, 3, 3)
	qDC, qAC, qScale := encodeChannel(lpqa.Q, 3, 3)

	var aDC, aScale float64
	var aAC []float64
	if hasAlpha {
		aDC, aAC, aScale = encodeChannel(lpqa.A, 5, 5)
	}

	hash := Hash{
		LDC:      lDC,
		PDC:      pDC,
		QDC:      qDC,
		LScale:   lScale,
		HasAlpha: hasAlpha,

		Lx:          lx,
		Ly:          ly,
		PScale:      pScale,
		QScale:      qScale,
		IsLandscape: w > h,

		ADC:    aDC,
		AScale: aScale,

		LAC: lAC,
		PAC: pAC,
		QAC: qAC,
		AAC: aAC,
	}

	avgA01 := avgA / float64(nbPixels)
	meta := EncodeMeta{
		Width:  origW,
		Height: origH,
		AvgR:   avgR * 255.0,
		AvgG:   avgG * 255.0,
		AvgB:   avgB * 255.0,
		AvgA:   avgA01 * 255.0,
	}

	return hash.Encode(), meta
}

func DecodeImage(hashData []byte) (image.Image, error) {
	return DecodeImageWithCfg(hashData, DecodingCfg{})
}

func DecodeImageWithCfg(hashData []byte, cfg DecodingCfg) (image.Image, error) {
	if cfg.BaseSize == 0 {
		cfg.BaseSize = 32
	}
	if cfg.SaturationBoost == 0.0 {
		cfg.SaturationBoost = 1.25
	}

	var hash Hash
	if err := hash.Decode(hashData, &cfg); err != nil {
		return nil, err
	}

	w, h := hash.Size(cfg.BaseSize)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	data := img.Pix

	nx := imax(hash.Lx, 3)
	ny := imax(hash.Ly, 3)
	if hash.HasAlpha {
		nx = imax(nx, 5)
		ny = imax(ny, 5)
	}

	buf := decodeCosPool.Get().(*decodeCosBuf)
	defer decodeCosPool.Put(buf)

	needX := w * nx
	needY := h * ny
	if cap(buf.cosX) < needX {
		buf.cosX = make([]float64, needX)
	}
	if cap(buf.cosY) < needY {
		buf.cosY = make([]float64, needY)
	}
	cosX := buf.cosX[:needX]
	cosY := buf.cosY[:needY]

	for x := 0; x < w; x++ {
		xf := float64(x)
		base := x * nx
		for cx := 0; cx < nx; cx++ {
			cosX[base+cx] = math.Cos(math.Pi / float64(w) * (xf + 0.5) * float64(cx))
		}
	}
	for y := 0; y < h; y++ {
		yf := float64(y)
		base := y * ny
		for cy := 0; cy < ny; cy++ {
			cosY[base+cy] = math.Cos(math.Pi / float64(h) * (yf + 0.5) * float64(cy))
		}
	}

	idx := 0
	for y := 0; y < h; y++ {
		fyBase := y * ny
		for x := 0; x < w; x++ {
			fxBase := x * nx

			l := hash.LDC
			j := 0
			for cy := 0; cy < hash.Ly; cy++ {
				cx := 0
				if cy == 0 {
					cx = 1
				}
				fy2 := cosY[fyBase+cy] * 2.0
				for ; cx*hash.Ly < hash.Lx*(hash.Ly-cy); cx++ {
					l += hash.LAC[j] * cosX[fxBase+cx] * fy2
					j++
				}
			}

			p := hash.PDC
			q := hash.QDC
			j = 0
			for cy := 0; cy < 3; cy++ {
				cx := 0
				if cy == 0 {
					cx = 1
				}
				fy2 := cosY[fyBase+cy] * 2.0
				for ; cx < 3-cy; cx++ {
					f := cosX[fxBase+cx] * fy2
					p += hash.PAC[j] * f
					q += hash.QAC[j] * f
					j++
				}
			}

			a := hash.ADC
			if hash.HasAlpha {
				j = 0
				for cy := 0; cy < 5; cy++ {
					cx := 0
					if cy == 0 {
						cx = 1
					}
					fy2 := cosY[fyBase+cy] * 2.0
					for ; cx < 5-cy; cx++ {
						a += hash.AAC[j] * cosX[fxBase+cx] * fy2
						j++
					}
				}
			}

			b := l - 2.0/3.0*p
			r := (3.0*l - b + q) / 2.0
			g := r - q
			af := math.Max(0.0, math.Min(1.0, a))

			data[idx] = byte(math.Max(0.0, math.Min(1.0, r)*255.0*af))
			data[idx+1] = byte(math.Max(0.0, math.Min(1.0, g)*255.0*af))
			data[idx+2] = byte(math.Max(0.0, math.Min(1.0, b)*255.0*af))
			data[idx+3] = byte(af * 255.0)
			idx += 4
		}
	}

	return img, nil
}
