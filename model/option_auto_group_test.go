package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOptionValueRejectsInvalidMaxTokenAutoGroups(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "1.5", "invalid"} {
		t.Run(value, func(t *testing.T) {
			assert.Error(t, validateOptionValue("MaxTokenAutoGroups", value))
		})
	}
	require.NoError(t, validateOptionValue("MaxTokenAutoGroups", "999999"))
}

func TestValidateOptionValueBoundsVietcombankConversionFee(t *testing.T) {
	key := "general_setting.exchange_rate_fee_ratio"
	for _, value := range []string{"-0.01", "1.01", "NaN", "+Inf", "invalid"} {
		t.Run(value, func(t *testing.T) {
			assert.Error(t, validateOptionValue(key, value))
		})
	}
	for _, value := range []string{"0", "0.1", "1"} {
		t.Run("valid_"+value, func(t *testing.T) {
			require.NoError(t, validateOptionValue(key, value))
		})
	}
}
