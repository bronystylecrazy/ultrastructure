// Adapted from go.n16f.net/thumbhash (ISC license), with local optimizations.
package thumbhash

import (
	"errors"
	"math"
)

var (
	ErrInvalidHash = errors.New("invalid hash")
)

type Hash struct {
	LDC      float64
	PDC      float64
	QDC      float64
	LScale   float64
	HasAlpha bool

	Lx          int
	Ly          int
	LCount      int
	PScale      float64
	QScale      float64
	IsLandscape bool

	ADC    float64
	AScale float64

	LAC []float64
	PAC []float64
	QAC []float64
	AAC []float64
}

func (h *Hash) Encode() []byte {
	nbAC := len(h.LAC) + len(h.PAC) + len(h.QAC)
	if h.HasAlpha {
		nbAC += len(h.AAC)
	}
	hashSize := 3 + 2 + (nbAC+1)/2
	if h.HasAlpha {
		hashSize += 1
	}

	hash := make([]byte, hashSize)

	header24 := iround(63.0 * h.LDC)
	header24 |= iround(31.5+31.5*h.PDC) << 6
	header24 |= iround(31.5+31.5*h.QDC) << 12
	header24 |= iround(31.0*h.LScale) << 18
	if h.HasAlpha {
		header24 |= 1 << 23
	}

	hash[0] = byte(header24)
	hash[1] = byte(header24 >> 8)
	hash[2] = byte(header24 >> 16)

	h.LCount = h.Lx
	if h.IsLandscape {
		h.LCount = h.Ly
	}

	header16 := h.LCount
	header16 |= iround(63.0*h.PScale) << 3
	header16 |= iround(63.0*h.QScale) << 9
	if h.IsLandscape {
		header16 |= 1 << 15
	}

	hash[3] = byte(header16)
	hash[4] = byte(header16 >> 8)

	if h.HasAlpha {
		hash[5] = byte(iround(15.0*h.ADC) | iround(15.0*h.AScale)<<4)
	}

	acs := [][]float64{h.LAC, h.PAC, h.QAC}
	if h.HasAlpha {
		acs = append(acs, h.AAC)
	}

	start := 5
	if h.HasAlpha {
		start = 6
	}

	idx := 0
	for i := 0; i < len(acs); i++ {
		ac := acs[i]
		for j := 0; j < len(ac); j++ {
			f := ac[j]
			hash[start+(idx/2)] |= byte(iround(15.0*f) << ((idx & 1) * 4))
			idx++
		}
	}

	return hash
}

func (h *Hash) Decode(data []byte, cfg *DecodingCfg) error {
	if len(data) < 5 {
		return ErrInvalidHash
	}

	header24 := int(data[0]) | int(data[1])<<8 | int(data[2])<<16

	h.LDC = float64(header24&63) / 63.0
	h.PDC = float64((header24>>6)&63)/31.5 - 1.0
	h.QDC = float64((header24>>12)&63)/31.5 - 1.0
	h.LScale = float64((header24>>18)&31) / 31.0
	h.HasAlpha = (header24 >> 23) != 0

	header16 := int(data[3]) | int(data[4])<<8
	h.PScale = float64((header16>>3)&63) / 63.0
	h.QScale = float64((header16>>9)&63) / 63.0
	h.IsLandscape = (header16 >> 15) != 0

	h.LCount = int(header16 & 7)
	if h.IsLandscape {
		if h.HasAlpha {
			h.Lx = 5
		} else {
			h.Lx = 7
		}
		h.Ly = imax(3, h.LCount)
	} else {
		h.Lx = imax(3, h.LCount)
		if h.HasAlpha {
			h.Ly = 5
		} else {
			h.Ly = 7
		}
	}

	h.ADC = 1.0
	h.AScale = 0.0
	if h.HasAlpha {
		if len(data) < 6 {
			return ErrInvalidHash
		}
		h.ADC = float64(data[5]&15) / 15.0
		h.AScale = float64(data[5]>>4) / 15.0
	}

	start := 5
	if h.HasAlpha {
		start = 6
	}
	idx := 0

	var err error
	decodeChannel := func(nx, ny int, scale float64) []float64 {
		count := channelACCount(nx, ny)
		ac := make([]float64, count)
		out := 0

		for cy := 0; cy < ny; cy++ {
			cx := 0
			if cy == 0 {
				cx = 1
			}
			for ; cx*ny < nx*(ny-cy); cx++ {
				hidx := start + (idx / 2)
				if hidx >= len(data) {
					err = ErrInvalidHash
					return nil
				}
				f := (float64((data[hidx]>>((idx&1)*4))&15)/7.5 - 1.0) * scale
				ac[out] = f
				out++
				idx++
			}
		}
		return ac
	}

	h.LAC = decodeChannel(h.Lx, h.Ly, h.LScale)
	h.PAC = decodeChannel(3, 3, h.PScale*cfg.SaturationBoost)
	h.QAC = decodeChannel(3, 3, h.QScale*cfg.SaturationBoost)
	if h.HasAlpha {
		h.AAC = decodeChannel(5, 5, h.AScale)
	}

	return err
}

func (hash *Hash) Size(baseSize int) (w int, h int) {
	ratio := float64(hash.Lx) / float64(hash.Ly)
	if ratio > 1.0 {
		w = baseSize
		h = iround(float64(baseSize) / ratio)
	} else {
		w = iround(float64(baseSize) * ratio)
		h = baseSize
	}
	return
}

func channelACCount(nx, ny int) int {
	count := 0
	for cy := 0; cy < ny; cy++ {
		cx := 0
		if cy == 0 {
			cx = 1
		}
		for ; cx*ny < nx*(ny-cy); cx++ {
			count++
		}
	}
	return count
}

func iround(x float64) int {
	return int(math.Round(x))
}

func imax(x, y int) int {
	if x >= y {
		return x
	}
	return y
}
