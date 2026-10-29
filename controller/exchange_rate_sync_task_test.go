package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestExchangeRateSyncTaskFollowsConfiguredSourceAndTime(t *testing.T) {
	settings := operation_setting.GetGeneralSetting()
	previousSource, previousTime := settings.ExchangeRateSource, settings.ExchangeRateSyncTime
	t.Cleanup(func() {
		settings.ExchangeRateSource = previousSource
		settings.ExchangeRateSyncTime = previousTime
	})

	handler := exchangeRateSyncTaskHandler{}
	settings.ExchangeRateSource = model.ExchangeRateSourceManual
	require.False(t, handler.Enabled(), "manual rates do not have an automatic provider to sync")

	settings.ExchangeRateSource = model.ExchangeRateSourceVietcombank
	settings.ExchangeRateSyncTime = "not-a-time"
	require.False(t, handler.Enabled(), "invalid schedule must not enqueue tasks")

	settings.ExchangeRateSyncTime = "10:00"
	require.True(t, handler.Enabled())

	location := time.FixedZone("server", 7*60*60)
	day := time.Date(2026, time.October, 29, 0, 0, 0, 0, location)
	require.False(t, handler.IsDue(day.Add(9*time.Hour+59*time.Minute), nil))
	require.True(t, handler.IsDue(day.Add(10*time.Hour), nil))

	todayTask := &model.SystemTask{CreatedAt: day.Add(10 * time.Hour).Unix()}
	require.False(t, handler.IsDue(day.Add(11*time.Hour), todayTask), "run at most once per local calendar day")
	require.True(t, handler.IsDue(day.Add(24*time.Hour+10*time.Hour), todayTask))
}
