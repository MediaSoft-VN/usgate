package controller

import (
	"context"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

type exchangeRateSyncTaskHandler struct{}

func (exchangeRateSyncTaskHandler) Type() string { return model.SystemTaskTypeExchangeRateSync }

func (exchangeRateSyncTaskHandler) Enabled() bool {
	settings := operation_setting.GetGeneralSetting()
	if settings.ExchangeRateSource != model.ExchangeRateSourceVietcombank {
		return false
	}
	_, err := time.Parse("15:04", settings.ExchangeRateSyncTime)
	return err == nil
}

func (exchangeRateSyncTaskHandler) Interval() time.Duration { return 24 * time.Hour }

func (exchangeRateSyncTaskHandler) IsDue(now time.Time, latest *model.SystemTask) bool {
	scheduledTime, err := time.ParseInLocation("15:04", operation_setting.GetGeneralSetting().ExchangeRateSyncTime, now.Location())
	if err != nil {
		return false
	}
	todayRunAt := time.Date(now.Year(), now.Month(), now.Day(), scheduledTime.Hour(), scheduledTime.Minute(), 0, 0, now.Location())
	if now.Before(todayRunAt) {
		return false
	}
	if latest == nil {
		return true
	}
	lastRun := time.Unix(latest.CreatedAt, 0).In(now.Location())
	lastYear, lastMonth, lastDay := lastRun.Date()
	currentYear, currentMonth, currentDay := now.Date()
	return lastYear != currentYear || lastMonth != currentMonth || lastDay != currentDay
}

func (exchangeRateSyncTaskHandler) NewPayload() any { return nil }

func (exchangeRateSyncTaskHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	if operation_setting.GetGeneralSetting().ExchangeRateSource != model.ExchangeRateSourceVietcombank {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, map[string]string{
			"skipped": "exchange rate source is not Vietcombank",
		}, nil)
		return
	}

	now := time.Now()
	if err := service.VietcombankExchangeRates(ctx, now); err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, errors.New("exchange rate synchronization failed"))
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, map[string]string{
		"source": model.ExchangeRateSourceVietcombank,
		"date":   now.Format("2006-01-02"),
	}, nil)
	common.SysLog(common.LogText("daily exchange rate synchronization completed: date=%s", now.Format("2006-01-02")))
}

var _ service.ScheduledSystemTaskHandler = exchangeRateSyncTaskHandler{}
var _ service.ScheduledSystemTaskDueChecker = exchangeRateSyncTaskHandler{}
