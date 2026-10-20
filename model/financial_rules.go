package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
)

// Financial rules keep money in integer minor units. This avoids float64
// precision loss at the wallet boundary and keeps the value safe for clients
// that represent numbers as JavaScript numbers.
const (
	FinancialDefaultCurrency = "USD"
	FinancialMinMinorUnits   = int64(1)
	FinancialMaxMinorUnits   = int64(common.MaxWalletQuota)
)

var (
	ErrFinancialInvalidScale   = errors.New("invalid currency minor-unit scale")
	ErrFinancialAmountOverflow = errors.New("financial amount exceeds supported range")
)

// ValidateFinancialMinorAmount validates a positive wallet amount already
// expressed in minor units.
func ValidateFinancialMinorAmount(amount int64) error {
	if amount < FinancialMinMinorUnits {
		return ErrFinancialInvalidAmount
	}
	if amount > FinancialMaxMinorUnits {
		return ErrFinancialAmountOverflow
	}
	return nil
}

// FinancialAmountToMinorUnits converts a major-unit decimal amount to minor
// units using half-away-from-zero rounding. The rounded result must be
// positive and within the wallet's JavaScript-safe integer boundary.
func FinancialAmountToMinorUnits(amount decimal.Decimal, minorDigits int32) (int64, error) {
	if minorDigits < 0 || minorDigits > 8 {
		return 0, ErrFinancialInvalidScale
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return 0, ErrFinancialInvalidAmount
	}

	minorUnits := amount.Shift(minorDigits).Round(0)
	value, err := common.WalletQuotaFromDecimalStrict(minorUnits)
	if err != nil {
		return 0, ErrFinancialAmountOverflow
	}
	result := int64(value)
	if err := ValidateFinancialMinorAmount(result); err != nil {
		return 0, err
	}
	return result, nil
}
