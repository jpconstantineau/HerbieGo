package ports

// RandomSource provides deterministic randomness to the application and engine.
type RandomSource interface {
	IntN(n int) int
	Float64() float64
}
