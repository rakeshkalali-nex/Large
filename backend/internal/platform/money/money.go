// Package money implements exact, nonnegative posting arithmetic. Signed ledger
// reversals are a workflow concern, not negative quantity or price inputs.
package money

import (
	"errors"
	"math"
	"math/big"
	"strings"

	"buildestate/backend/internal/contracts"
)

var (
	ErrInvalid        = errors.New("invalid nonnegative decimal")
	ErrOverflow       = errors.New("amount exceeds signed 64-bit storage")
	ErrSourceMismatch = errors.New("cost source requires reviewed correction")
)

func ParseQuantity(value string) (int64, error) { return parseScaled(value, 3) }
func ParsePrice(value string) (int64, error)    { return parseScaled(value, 4) }
func ParseMinor(value string) (int64, error)    { return parseScaled(value, 0) }

// parseScaled rejects signs, whitespace, exponent notation and excess precision,
// including extra zero places. Leading zeroes are accepted without loss of value.
func parseScaled(value string, places int) (int64, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts[0]) == 0 {
		return 0, ErrInvalid
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if places == 0 || len(fraction) == 0 || len(fraction) > places {
			return 0, ErrInvalid
		}
	}
	var result int64
	for _, part := range []string{parts[0], fraction} {
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				return 0, ErrInvalid
			}
			digit := int64(part[i] - '0')
			if result > (math.MaxInt64-digit)/10 {
				return 0, ErrOverflow
			}
			result = result*10 + digit
		}
	}
	for i := len(fraction); i < places; i++ {
		if result > math.MaxInt64/10 {
			return 0, ErrOverflow
		}
		result *= 10
	}
	return result, nil
}

// LineMinor rounds q*p*10^minorDigits/10^7 half-up once per line. Big
// intermediates permit valid results even when the unscaled product overflows.
func LineMinor(quantity, price int64, minorDigits int) (int64, error) {
	if quantity < 0 || price < 0 || minorDigits < 0 || minorDigits > 4 {
		return 0, ErrInvalid
	}
	denominator := int64(1)
	for i := 0; i < 7-minorDigits; i++ {
		denominator *= 10
	}
	numerator := new(big.Int).Mul(big.NewInt(quantity), big.NewInt(price))
	numerator.Add(numerator, big.NewInt(denominator/2))
	numerator.Quo(numerator, big.NewInt(denominator))
	if !numerator.IsInt64() {
		return 0, ErrOverflow
	}
	return numerator.Int64(), nil
}

type Line struct{ Quantity, Price int64 }

// TotalMinor sums separately rounded lines and explicit nonnegative minor tax.
func TotalMinor(lines []Line, tax int64, minorDigits int) (int64, error) {
	if tax < 0 || minorDigits < 0 || minorDigits > 4 {
		return 0, ErrInvalid
	}
	total := tax
	for _, line := range lines {
		amount, err := LineMinor(line.Quantity, line.Price, minorDigits)
		if err != nil {
			return 0, err
		}
		if amount > math.MaxInt64-total {
			return 0, ErrOverflow
		}
		total += amount
	}
	return total, nil
}

// MatchCostSource checks shared economic identity and exact amount only. Origin
// records may differ (expense versus bill); eligibility, approved limits, source
// persistence and recognition remain the calling workflow's responsibility.
func MatchCostSource(existing, linked contracts.CostSource) error {
	a, err := ParseMinor(string(existing.Amount))
	if err != nil {
		return err
	}
	b, err := ParseMinor(string(linked.Amount))
	if err != nil {
		return err
	}
	if existing.ID == "" || existing.ProjectID == "" || existing.CategoryID == "" ||
		existing.ID != linked.ID || existing.ProjectID != linked.ProjectID ||
		existing.CategoryID != linked.CategoryID || a != b {
		return ErrSourceMismatch
	}
	if (existing.CommitmentLineID == nil) != (linked.CommitmentLineID == nil) {
		return ErrSourceMismatch
	}
	if existing.CommitmentLineID != nil && (*existing.CommitmentLineID == "" || *existing.CommitmentLineID != *linked.CommitmentLineID) {
		return ErrSourceMismatch
	}
	return nil
}
