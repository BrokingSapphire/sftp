package ai

import (
	"math"
	"testing"
)

func TestCosine(t *testing.T) {
	if got := Cosine([]float32{1, 0, 0}, []float32{1, 0, 0}); math.Abs(float64(got)-1) > 1e-6 {
		t.Fatalf("identical vectors = %f, want 1", got)
	}
	if got := Cosine([]float32{1, 0}, []float32{0, 1}); math.Abs(float64(got)) > 1e-6 {
		t.Fatalf("orthogonal = %f, want 0", got)
	}
	if got := Cosine([]float32{1, 2, 3}, []float32{2, 4, 6}); math.Abs(float64(got)-1) > 1e-6 {
		t.Fatalf("parallel = %f, want 1", got)
	}
	if got := Cosine([]float32{1, 0}, []float32{1, 0, 0}); got != 0 {
		t.Fatalf("mismatched length should be 0, got %f", got)
	}
	if got := Cosine(nil, nil); got != 0 {
		t.Fatalf("empty = %f, want 0", got)
	}
}
