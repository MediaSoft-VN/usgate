package controller

import (
	"context"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

const upstreamCostSyncTaskDefaultIntervalMinutes = 360

type upstreamCostSyncTaskHandler struct{}

func (upstreamCostSyncTaskHandler) Type() string { return model.SystemTaskTypeUpstreamCostSync }

func (upstreamCostSyncTaskHandler) Enabled() bool {
	return common.GetEnvOrDefaultBool("UPSTREAM_COST_SYNC_TASK_ENABLED", true)
}

func (upstreamCostSyncTaskHandler) Interval() time.Duration {
	minutes := common.GetEnvOrDefault(
		"UPSTREAM_COST_SYNC_TASK_INTERVAL_MINUTES",
		upstreamCostSyncTaskDefaultIntervalMinutes,
	)
	if minutes < 1 {
		minutes = upstreamCostSyncTaskDefaultIntervalMinutes
	}
	return time.Duration(minutes) * time.Minute
}

func (upstreamCostSyncTaskHandler) NewPayload() any { return nil }

func (upstreamCostSyncTaskHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	summary := runUpstreamCostSyncTask(ctx)
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

func runUpstreamCostSyncTask(ctx context.Context) map[string]int {
	channels, err := model.GetAllChannels(0, 0, true, true)
	if err != nil {
		return map[string]int{"failed": 1}
	}

	channelIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		if channel.Status == common.ChannelStatusEnabled && supportsAutomaticUpstreamCosts(channel) {
			channelIDs = append(channelIDs, channel.Id)
		}
	}
	results := fetchUpstreamCosts(ctx, channelIDs, defaultTimeoutSeconds)
	summary := map[string]int{"channels": len(channelIDs)}
	for _, result := range results {
		if !result.Supported || len(result.Costs) == 0 {
			summary["failed"]++
			continue
		}
		version, parseErr := strconv.ParseInt(result.Version, 10, 64)
		if parseErr != nil {
			summary["failed"]++
			continue
		}
		if err := model.ApplyChannelModelCosts(result.ChannelID, version, result.Costs); err != nil {
			summary["failed"]++
			continue
		}
		summary["updated"]++
	}
	return summary
}

var _ service.ScheduledSystemTaskHandler = upstreamCostSyncTaskHandler{}
