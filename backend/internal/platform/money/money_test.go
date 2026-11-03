package money

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"buildestate/backend/internal/contracts"
)

func TestScaledParsing(t *testing.T) {
	for _, tc := range []struct {
		value  string
		places int
		want   int64
	}{
		{"1.250", 3, 1250}, {"12.3456", 4, 123456}, {"0", 3, 0}, {"0001.2", 3, 1200},
		{"9223372036854775.807", 3, math.MaxInt64}, {"922337203685477.5807", 4, math.MaxInt64}, {"9223372036854775807", 0, math.MaxInt64},
	} {
		got, err := parseScaled(tc.value, tc.places)
		if err != nil || got != tc.want {
			t.Fatalf("%s: %d, %v", tc.value, got, err)
		}
	}
	for _, value := range []string{"", "-1", "+1", " 1", "1 ", "1e3", ".1", "1.", "1.0000", "1..0", "NaN", "١", "9223372036854775.808"} {
		if _, err := ParseQuantity(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	if _, err := ParsePrice("0.00001"); err == nil {
		t.Fatal("accepted excess price precision")
	}
	if _, err := ParseMinor("1.0"); err == nil {
		t.Fatal("accepted fractional minor units")
	}
}

func TestLineAndTotal(t *testing.T) {
	q, _ := ParseQuantity("1.250")
	p, _ := ParsePrice("12.3456")
	if got, err := LineMinor(q, p, 2); err != nil || got != 1543 {
		t.Fatalf("example: %d %v", got, err)
	}
	for _, tc := range []struct {
		q, p int64
		d    int
		want int64
	}{
		{1, 49999, 2, 0}, {1, 50000, 2, 1}, {1, 50001, 2, 1}, {math.MaxInt64, 1000, 4, math.MaxInt64},
	} {
		if got, err := LineMinor(tc.q, tc.p, tc.d); err != nil || got != tc.want {
			t.Fatalf("%+v: %d %v", tc, got, err)
		}
	}
	if got, err := TotalMinor([]Line{{1, 50000}, {1, 50000}}, 3, 2); err != nil || got != 5 {
		t.Fatalf("per-line total: %d %v", got, err)
	}
	for _, tc := range []struct {
		q, p int64
		d    int
	}{{-1, 1, 2}, {1, -1, 2}, {1, 1, -1}, {1, 1, 5}, {math.MaxInt64, math.MaxInt64, 4}} {
		if _, err := LineMinor(tc.q, tc.p, tc.d); err == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
	if _, err := TotalMinor([]Line{{1000, 10000}}, math.MaxInt64, 2); err == nil {
		t.Fatal("total overflow accepted")
	}
	if _, err := TotalMinor(nil, -1, 2); err == nil {
		t.Fatal("negative tax accepted")
	}
	if _, err := TotalMinor(nil, 0, 5); err == nil {
		t.Fatal("invalid precision accepted")
	}
}

func TestIntegerOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 2000; i++ {
		q, p := rng.Int63(), rng.Int63n(100000)
		d := rng.Intn(5)
		n := new(big.Int).Mul(big.NewInt(q), big.NewInt(p))
		n.Mul(n, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d)), nil))
		whole, remainder := new(big.Int), new(big.Int)
		whole.QuoRem(n, big.NewInt(10000000), remainder)
		if remainder.Int64() >= 5000000 {
			whole.Add(whole, big.NewInt(1))
		}
		got, err := LineMinor(q, p, d)
		if !whole.IsInt64() {
			if err == nil {
				t.Fatal("oracle overflow accepted")
			}
			continue
		}
		if err != nil || got != whole.Int64() {
			t.Fatalf("oracle mismatch: %d %v want %s", got, err, whole)
		}
	}
}

func TestMatchCostSource(t *testing.T) {
	line := "line"
	other := "other"
	source := contracts.CostSource{ID: "source", ProjectID: "project", CategoryID: "category", CommitmentLineID: &line, Amount: "10", OriginType: "expense", OriginID: "expense"}
	linked := source
	linked.Amount = "010"
	linked.OriginType = "invoice"
	linked.OriginID = "invoice"
	if err := MatchCostSource(source, linked); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*contracts.CostSource){
		func(s *contracts.CostSource) { s.ID = "other" }, func(s *contracts.CostSource) { s.ProjectID = "other" },
		func(s *contracts.CostSource) { s.CategoryID = "other" }, func(s *contracts.CostSource) { s.Amount = "11" },
		func(s *contracts.CostSource) { s.Amount = "-10" }, func(s *contracts.CostSource) { s.CommitmentLineID = nil },
		func(s *contracts.CostSource) { s.CommitmentLineID = &other },
	} {
		linked = source
		change(&linked)
		if err := MatchCostSource(source, linked); err == nil {
			t.Fatal("mismatch accepted")
		}
	}
}
