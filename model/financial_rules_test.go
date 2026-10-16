package model

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinancialAmountToMinorUnitsUsesHalfAwayFromZeroRounding(t *testing.T) {
	amount, err := FinancialAmountToMinorUnits(decimal.RequireFromString("10.005"), 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1001), amount)

	amount, err = FinancialAmountToMinorUnits(decimal.RequireFromString("10.004"), 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), amount)
}

func TestFinancialAmountToMinorUnitsRejectsInvalidBusinessValues(t *testing.T) {
	tests := []struct {
		name        string
		amount      string
		minorDigits int32
		wantErr     error
	}{
		{name: "zero", amount: "0", minorDigits: 2, wantErr: ErrFinancialInvalidAmount},
		{name: "negative", amount: "-1", minorDigits: 2, wantErr: ErrFinancialInvalidAmount},
		{name: "invalid scale", amount: "1", minorDigits: 9, wantErr: ErrFinancialInvalidScale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FinancialAmountToMinorUnits(decimal.RequireFromString(tt.amount), tt.minorDigits)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateFinancialMinorAmountRejectsOverflow(t *testing.T) {
	assert.NoError(t, ValidateFinancialMinorAmount(FinancialMinMinorUnits))
	assert.ErrorIs(t, ValidateFinancialMinorAmount(0), ErrFinancialInvalidAmount)
	assert.ErrorIs(t, ValidateFinancialMinorAmount(FinancialMaxMinorUnits+1), ErrFinancialAmountOverflow)
}
