package seeded

import randv2 "math/rand/v2"

// Source is a deterministic random source backed by PCG.
type Source struct {
	random *randv2.Rand
}

// New constructs a deterministic random source from a single persisted seed.
func New(seed uint64) *Source {
	return &Source{
		random: randv2.New(randv2.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
	}
}

// IntN returns, as an int, a non-negative pseudo-random number in [0,n).
func (s *Source) IntN(n int) int {
	return s.random.IntN(n)
}

// Float64 returns, as a float64, a pseudo-random number in [0.0,1.0).
func (s *Source) Float64() float64 {
	return s.random.Float64()
}
