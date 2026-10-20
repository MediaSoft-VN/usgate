package model

import (
	"errors"
	"math"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	providerCreditThresholdKey = "provider_credit_threshold"
	providerCreditAutoDisabled = "provider_credit_auto_disabled"
	providerCreditDisableReason = "provider credit below threshold"
)

var ErrInvalidProviderCreditThreshold = errors.New("invalid provider credit threshold")

type ProviderCreditSettings struct {
	Threshold   float64 `json:"threshold"`
	AutoDisabled bool   `json:"auto_disabled"`
}

func providerCreditThreshold(channel *Channel) float64 {
	value, ok := channel.GetOtherInfo()[providerCreditThresholdKey].(float64)
	if !ok || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	return value
}

func GetProviderCreditSettings(channel *Channel) ProviderCreditSettings {
	if channel == nil {
		return ProviderCreditSettings{}
	}
	info := channel.GetOtherInfo()
	_, autoDisabled := info[providerCreditAutoDisabled].(bool)
	return ProviderCreditSettings{
		Threshold:    providerCreditThreshold(channel),
		AutoDisabled: autoDisabled,
	}
}

func SetProviderCreditThreshold(channelID int, threshold float64) (*Channel, error) {
	if channelID <= 0 || math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < 0 {
		return nil, ErrInvalidProviderCreditThreshold
	}
	channel, err := GetChannelById(channelID, true)
	if err != nil {
		return nil, err
	}
	info := channel.GetOtherInfo()
	info[providerCreditThresholdKey] = threshold
	channel.SetOtherInfo(info)
	if err = DB.Model(&Channel{}).Where("id = ?", channelID).Update("other_info", channel.OtherInfo).Error; err != nil {
		return nil, err
	}
	if err = EvaluateProviderCredit(channelID); err != nil {
		return nil, err
	}
	return GetChannelById(channelID, false)
}

// EvaluateProviderCredit applies the configured threshold without overriding a
// manual disable. The marker is stored in OtherInfo to avoid a schema change.
func EvaluateProviderCredit(channelID int) error {
	channel, err := GetChannelById(channelID, true)
	if err != nil {
		return err
	}
	settings := GetProviderCreditSettings(channel)
	if settings.Threshold <= 0 {
		if settings.AutoDisabled {
			info := channel.GetOtherInfo()
			delete(info, providerCreditAutoDisabled)
			channel.SetOtherInfo(info)
			if err := DB.Model(&Channel{}).Where("id = ?", channelID).Update("other_info", channel.OtherInfo).Error; err != nil {
				return err
			}
			UpdateChannelStatus(channelID, "", common.ChannelStatusEnabled, "provider credit policy disabled")
		}
		return nil
	}
	if channel.Balance < settings.Threshold && channel.Status == common.ChannelStatusEnabled {
		info := channel.GetOtherInfo()
		info[providerCreditAutoDisabled] = true
		channel.SetOtherInfo(info)
		if err := DB.Model(&Channel{}).Where("id = ?", channelID).Update("other_info", channel.OtherInfo).Error; err != nil {
			return err
		}
		UpdateChannelStatus(channelID, "", common.ChannelStatusAutoDisabled, providerCreditDisableReason)
		return nil
	}
	if channel.Balance >= settings.Threshold && settings.AutoDisabled {
		info := channel.GetOtherInfo()
		delete(info, providerCreditAutoDisabled)
		channel.SetOtherInfo(info)
		if err := DB.Model(&Channel{}).Where("id = ?", channelID).Update("other_info", channel.OtherInfo).Error; err != nil {
			return err
		}
		UpdateChannelStatus(channelID, "", common.ChannelStatusEnabled, "provider credit restored")
	}
	return nil
}

func ProviderCreditSnapshot(channel *Channel) map[string]any {
	settings := GetProviderCreditSettings(channel)
	return map[string]any{
		"channel_id":     channel.Id,
		"balance":        channel.Balance,
		"threshold":      settings.Threshold,
		"below_threshold": settings.Threshold > 0 && channel.Balance < settings.Threshold,
		"auto_disabled":  settings.AutoDisabled,
		"status":         channel.Status,
		"checked_at":     time.Now().Unix(),
	}
}
