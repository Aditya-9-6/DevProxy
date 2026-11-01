package tdigest

import (
	"math"
	"testing"
)

func TestDigestBasic(t *testing.T) {
	d := AcquireDigest()
	defer d.Release()

	for i := 1; i <= 100; i++ {
		d.Add(float64(i))
	}

	if d.Count() != 100 {
		t.Errorf("expected count 100, got %d", d.Count())
	}

	q50 := d.Quantile(0.50)
	if math.IsNaN(q50) || q50 < 45 || q50 > 55 {
		t.Errorf("expected p50 around 50, got %f", q50)
	}
}

func TestDigestBoundariesAndErrors(t *testing.T) {
	d := New(50)
	
	// Empty digest quantile should be NaN
	if !math.IsNaN(d.Quantile(0.5)) {
		t.Errorf("expected NaN for empty digest quantile")
	}

	d.Add(10.0)
	
	// Invalid quantile bounds
	if !math.IsNaN(d.Quantile(-0.1)) {
		t.Errorf("expected NaN for q < 0")
	}
	if !math.IsNaN(d.Quantile(1.1)) {
		t.Errorf("expected NaN for q > 1")
	}

	// Invalid additions
	d.Add(math.NaN())
	d.AddWeighted(20.0, -1.0)
	if d.Count() != 1 {
		t.Errorf("expected count to remain 1 after invalid additions, got %d", d.Count())
	}
}

func TestCentroidAtomicity(t *testing.T) {
	var c Centroid
	c.SetMean(10.0)
	c.SetWeight(1.0)

	c.Add(20.0, 1.0)

	if c.GetWeight() != 2.0 {
		t.Errorf("expected weight 2.0, got %f", c.GetWeight())
	}
	if c.GetMean() != 15.0 {
		t.Errorf("expected mean 15.0, got %f", c.GetMean())
	}
}
