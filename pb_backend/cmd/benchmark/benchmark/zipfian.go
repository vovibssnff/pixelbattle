package benchmark

import (
	"math/rand"
)

// CoordinateGenerator produces (x, y) pairs for benchmark workloads.
type CoordinateGenerator interface {
	Next() (x, y uint)
}

// UniformGenerator picks coordinates uniformly at random.
type UniformGenerator struct {
	rng           *rand.Rand
	width, height uint
}

func NewUniformGenerator(width, height uint, seed int64) *UniformGenerator {
	return &UniformGenerator{
		// #nosec G404 -- benchmark workload intentionally uses deterministic pseudo-random streams
		rng:    rand.New(rand.NewSource(seed)),
		width:  width,
		height: height,
	}
}

func (u *UniformGenerator) Next() (x, y uint) {
	return uint(u.rng.Intn(int(u.width))), uint(u.rng.Intn(int(u.height)))
}

// ZipfianGenerator produces coordinates with a Zipfian (power-law) distribution,
// modeling hotspot behaviour where ~20% of pixels receive ~80% of writes.
// A fixed permutation scrambles the rank-to-coordinate mapping so that
// hotspots are spread across the canvas rather than clustered at (0,0).
type ZipfianGenerator struct {
	zipf        *rand.Zipf
	permutation []uint32
	width       uint
}

func NewZipfianGenerator(width, height uint, seed int64, skew float64) *ZipfianGenerator {
	// #nosec G404 -- benchmark workload intentionally uses deterministic pseudo-random streams
	rng := rand.New(rand.NewSource(seed))
	area := width * height

	perm := make([]uint32, area)
	for i := range perm {
		perm[i] = uint32(i)
	}
	rng.Shuffle(len(perm), func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })

	// Larger s values increase hotspot skew while v stays at 1.
	if skew < 1.01 {
		skew = 1.01
	}
	zipf := rand.NewZipf(rng, skew, 1.0, uint64(area-1))

	return &ZipfianGenerator{
		zipf:        zipf,
		permutation: perm,
		width:       width,
	}
}

func (z *ZipfianGenerator) Next() (x, y uint) {
	rank := z.zipf.Uint64()
	idx := z.permutation[rank]
	return uint(idx) % z.width, uint(idx) / z.width
}
