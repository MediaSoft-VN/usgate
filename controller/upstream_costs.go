package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	maxUpstreamCostChannels = 50
	maxUpstreamCostModels   = 20000
	maxUpstreamCostBytes    = 10 << 20
	maxUpstreamApplyBytes   = 16 << 20
)

type upstreamCostChannel struct {
	ID             int                      `json:"id"`
	Name           string                   `json:"name"`
	Type           int                      `json:"type"`
	TypeName       string                   `json:"type_name"`
	Supported      bool                     `json:"supported"`
	UnsupportedWhy string                   `json:"unsupported_reason,omitempty"`
	Version        int64                    `json:"version"`
	Costs          []model.ChannelModelCost `json:"costs"`
}

type upstreamCostResult struct {
	ChannelID int                      `json:"channel_id"`
	Source    string                   `json:"source"`
	Supported bool                     `json:"supported"`
	Error     string                   `json:"error,omitempty"`
	Version   int64                    `json:"version"`
	Costs     []model.ChannelModelCost `json:"costs,omitempty"`
}

type upstreamCostRequest struct {
	ChannelIDs []int `json:"channel_ids"`
	Timeout    int   `json:"timeout"`
}

type upstreamCostApplyRequest struct {
	Items []struct {
		ChannelID int                      `json:"channel_id"`
		Version   int64                    `json:"version"`
		Costs     []model.ChannelModelCost `json:"costs"`
	} `json:"items"`
}

func ListUpstreamCostChannels(c *gin.Context) {
	channels, err := model.GetAllChannels(0, 0, true, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(channels) > maxUpstreamCostChannels {
		channels = channels[:maxUpstreamCostChannels]
	}
	result := make([]upstreamCostChannel, 0, len(channels))
	for _, channel := range channels {
		costs, costErr := model.ListChannelModelCosts(channel.Id)
		if costErr != nil {
			common.ApiError(c, costErr)
			return
		}
		version, versionErr := model.ChannelModelCostVersion(channel.Id)
		if versionErr != nil {
			common.ApiError(c, versionErr)
			return
		}
		item := upstreamCostChannel{ID: channel.Id, Name: channel.Name, Type: channel.Type, TypeName: constant.GetChannelTypeName(channel.Type), Supported: supportsAutomaticUpstreamCosts(channel), Version: version, Costs: costs}
		if !item.Supported {
			item.Supported = false
			item.UnsupportedWhy = "This channel has no configured upstream cost source"
		}
		result = append(result, item)
	}
	common.ApiSuccess(c, result)
}

func PreviewUpstreamCosts(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	var request upstreamCostRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid request parameter format"))
		return
	}
	if err := validateUpstreamCostRequest(&request); err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage(err.Error()))
		return
	}
	results := fetchUpstreamCosts(c.Request.Context(), request.ChannelIDs, request.Timeout)
	common.ApiSuccess(c, gin.H{"results": results})
}

func ApplyUpstreamCosts(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUpstreamApplyBytes)
	var request upstreamCostApplyRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || len(request.Items) == 0 || len(request.Items) > maxUpstreamCostChannels {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid request parameter format"))
		return
	}
	updated := make([]int, 0, len(request.Items))
	for _, item := range request.Items {
		if item.ChannelID <= 0 || len(item.Costs) > maxUpstreamCostModels {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid request parameter format"))
			return
		}
		if err := model.ApplyChannelModelCosts(item.ChannelID, item.Version, item.Costs); err != nil {
			if err == model.ErrChannelModelCostConflict {
				common.ApiErrorStatus(c, http.StatusConflict, common.NewMessage(err.Error()))
				return
			}
			common.ApiError(c, err)
			return
		}
		updated = append(updated, item.ChannelID)
	}
	recordManageAudit(c, "upstream_costs.apply", map[string]any{"channels": updated})
	common.ApiSuccess(c, gin.H{"updated_channels": updated})
}

func validateUpstreamCostRequest(request *upstreamCostRequest) error {
	if len(request.ChannelIDs) == 0 || len(request.ChannelIDs) > maxUpstreamCostChannels {
		return fmt.Errorf("channel_ids must contain between 1 and %d items", maxUpstreamCostChannels)
	}
	if request.Timeout <= 0 {
		request.Timeout = defaultTimeoutSeconds
	}
	if request.Timeout > 60 {
		return fmt.Errorf("timeout must not exceed 60 seconds")
	}
	return nil
}

func fetchUpstreamCosts(parent context.Context, channelIDs []int, timeout int) []upstreamCostResult {
	results := make([]upstreamCostResult, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		result := upstreamCostResult{ChannelID: channelID}
		channel, err := model.GetChannelById(channelID, true)
		if err != nil {
			result.Error = "failed to load channel"
			results = append(results, result)
			continue
		}
		result.Version, _ = model.ChannelModelCostVersion(channelID)
		if !supportsAutomaticUpstreamCosts(channel) {
			result.Supported = false
			result.Source = "manual"
			result.Error = "This channel has no configured upstream cost source and must be entered manually"
			results = append(results, result)
			continue
		}
		baseURL := strings.TrimRight(channel.GetBaseURL(), "/")
		isOpenRouter := channel.Type == constant.ChannelTypeOpenRouter
		endpoint := baseURL + modelsDevPath
		result.Source = "models.dev"
		if isOpenRouter {
			endpoint = baseURL + "/v1/models"
			result.Source = "openrouter"
		}
		ctx, cancel := context.WithTimeout(parent, time.Duration(timeout)*time.Second)
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if requestErr == nil && channel.Type == constant.ChannelTypeOpenRouter {
			key, _, keyErr := channel.GetNextEnabledKey()
			if keyErr != nil || strings.TrimSpace(key) == "" {
				requestErr = fmt.Errorf("no API key configured")
			} else {
				req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(key))
			}
		}
		if requestErr != nil {
			cancel()
			result.Error = requestErr.Error()
			results = append(results, result)
			continue
		}
		response, requestErr := http.DefaultClient.Do(req)
		if requestErr != nil {
			cancel()
			result.Error = requestErr.Error()
			results = append(results, result)
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxUpstreamCostBytes+1))

		_ = response.Body.Close()
		cancel()
		if response.StatusCode != http.StatusOK {
			result.Error = response.Status
		} else if readErr != nil || len(body) > maxUpstreamCostBytes {
			result.Error = "upstream response is too large or unreadable"
		} else if channel.Type == constant.ChannelTypeOpenRouter {
			var parseErr error
			result.Costs, parseErr = parseOpenRouterCosts(bytes.NewReader(body), channelID)
			if parseErr != nil {
				result.Error = parseErr.Error()
			}
		} else {
			var parseErr error
			result.Costs, parseErr = parseModelsDevCosts(bytes.NewReader(body), channelID)
			if parseErr != nil {
				result.Error = parseErr.Error()
			}
		}
		result.Supported = result.Error == ""
		results = append(results, result)
	}
	return results
}

func supportsAutomaticUpstreamCosts(channel *model.Channel) bool {
	if channel == nil {
		return false
	}
	if channel.Type == constant.ChannelTypeOpenRouter {
		return true
	}
	return strings.TrimRight(channel.GetBaseURL(), "/") == modelsDevPresetBaseURL
}

func microsPerMillion(value float64) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > float64(math.MaxInt64)/1e6 {
		return 0, false
	}
	return int64(math.Round(value * 1e6)), true
}

func parseOpenRouterCosts(reader io.Reader, channelID int) ([]model.ChannelModelCost, error) {
	var response struct {
		Data []struct {
			ID      string `json:"id"`
			Pricing struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
				CacheRead  string `json:"input_cache_read"`
			} `json:"pricing"`
		} `json:"data"`
	}
	if err := common.DecodeJson(reader, &response); err != nil {
		return nil, err
	}
	result := make([]model.ChannelModelCost, 0, len(response.Data))
	for _, item := range response.Data {
		input, inputErr := strconv.ParseFloat(item.Pricing.Prompt, 64)
		output, outputErr := strconv.ParseFloat(item.Pricing.Completion, 64)
		if inputErr != nil || outputErr != nil || input < 0 || output < 0 || (input == 0 && output == 0) {
			continue
		}
		inputMicros, inputOK := microsPerMillion(input * 1e6)
		outputMicros, outputOK := microsPerMillion(output * 1e6)
		if !inputOK || !outputOK || strings.TrimSpace(item.ID) == "" {
			continue
		}
		cost := model.ChannelModelCost{ChannelID: channelID, Model: item.ID, InputUSDPerMillionMicros: inputMicros, OutputUSDPerMillionMicros: outputMicros, Source: "openrouter", Valid: true}
		if cache, err := strconv.ParseFloat(item.Pricing.CacheRead, 64); err == nil && cache >= 0 {
			cost.CacheReadUSDPerMillionMicros, _ = microsPerMillion(cache * 1e6)
		}
		result = append(result, cost)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid OpenRouter pricing entries found")
	}
	return result, nil
}

func parseModelsDevCosts(reader io.Reader, channelID int) ([]model.ChannelModelCost, error) {
	data, err := convertModelsDevCosts(reader, channelID)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func convertModelsDevCosts(reader io.Reader, channelID int) ([]model.ChannelModelCost, error) {
	var providers map[string]modelsDevProvider
	if err := common.DecodeJson(reader, &providers); err != nil {
		return nil, err
	}
	result := make([]model.ChannelModelCost, 0)
	for _, provider := range providers {
		for name, item := range provider.Models {
			if item.Cost.Input == nil || !isValidNonNegativeCost(*item.Cost.Input) {
				continue
			}
			input, ok := microsPerMillion(*item.Cost.Input)
			if !ok {
				continue
			}
			cost := model.ChannelModelCost{ChannelID: channelID, Model: name, InputUSDPerMillionMicros: input, Source: "models.dev", Valid: true}
			if item.Cost.Output != nil && isValidNonNegativeCost(*item.Cost.Output) {
				cost.OutputUSDPerMillionMicros, _ = microsPerMillion(*item.Cost.Output)
			}
			if item.Cost.CacheRead != nil && isValidNonNegativeCost(*item.Cost.CacheRead) {
				cost.CacheReadUSDPerMillionMicros, _ = microsPerMillion(*item.Cost.CacheRead)
			}
			result = append(result, cost)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid models.dev pricing entries found")
	}
	return result, nil
}
