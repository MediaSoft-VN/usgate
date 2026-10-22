package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const channelModelCostModelMaxLength = 255

var (
	ErrChannelModelCostNil          = errors.New("channel model cost is nil")
	ErrChannelModelCostInvalidID    = errors.New("channel model cost channel id must be positive")
	ErrChannelModelCostInvalidModel = errors.New("channel model cost model must not be empty")
	ErrChannelModelCostModelTooLong = errors.New("channel model cost model is too long")
	ErrChannelModelCostNegative     = errors.New("channel model cost must not be negative")
)

// ChannelModelCost stores provider costs in integer USD micro-units per one
// million tokens. It is deliberately separate from the customer sale price
// kept in the ratio and billing settings.
type ChannelModelCost struct {
	ID                            int    `json:"id" gorm:"primaryKey"`
	ChannelID                     int    `json:"channel_id" gorm:"not null;uniqueIndex:idx_channel_model_cost_channel_model,priority:1"`
	Model                         string `json:"model" gorm:"size:255;not null;uniqueIndex:idx_channel_model_cost_channel_model,priority:2"`
	InputUSDPerMillionMicros      int64  `json:"input_usd_per_million_micros" gorm:"not null;default:0"`
	OutputUSDPerMillionMicros     int64  `json:"output_usd_per_million_micros" gorm:"not null;default:0"`
	CacheReadUSDPerMillionMicros  int64  `json:"cache_read_usd_per_million_micros" gorm:"not null;default:0"`
	CacheWriteUSDPerMillionMicros int64  `json:"cache_write_usd_per_million_micros" gorm:"not null;default:0"`
	Source                        string `json:"source" gorm:"size:64;not null"`
	Manual                        bool   `json:"manual" gorm:"not null"`
	Valid                         bool   `json:"valid" gorm:"not null"`
	Error                         string `json:"error" gorm:"type:text"`
	FetchedAt                     int64  `json:"fetched_at" gorm:"bigint"`
	CreatedAt                     int64  `json:"created_at" gorm:"bigint;autoCreateTime"`
	UpdatedAt                     int64  `json:"updated_at" gorm:"bigint;autoUpdateTime"`
}

// RoutingCost is the simple provider-cost estimate used when no request usage
// mix is available during channel selection. Customer pricing is intentionally
// not involved in this value.
func (cost ChannelModelCost) RoutingCost() int64 {
	return cost.InputUSDPerMillionMicros + cost.OutputUSDPerMillionMicros
}

func (ChannelModelCost) TableName() string {
	return "channel_model_costs"
}

func (cost *ChannelModelCost) Validate() error {
	if cost == nil {
		return ErrChannelModelCostNil
	}
	if cost.ChannelID <= 0 {
		return ErrChannelModelCostInvalidID
	}
	cost.Model = strings.TrimSpace(cost.Model)
	if cost.Model == "" {
		return ErrChannelModelCostInvalidModel
	}
	if len(cost.Model) > channelModelCostModelMaxLength {
		return ErrChannelModelCostModelTooLong
	}
	if cost.InputUSDPerMillionMicros < 0 || cost.OutputUSDPerMillionMicros < 0 || cost.CacheReadUSDPerMillionMicros < 0 || cost.CacheWriteUSDPerMillionMicros < 0 {
		return ErrChannelModelCostNegative
	}
	return nil
}

// UpsertChannelModelCost inserts a cost or replaces the latest fetched data
// for the same channel and model. It does not change any customer sale price.
func UpsertChannelModelCost(cost *ChannelModelCost) error {
	if err := cost.Validate(); err != nil {
		return err
	}
	if DB == nil {
		return errors.New("database is not initialized")
	}
	now := time.Now().Unix()
	if cost.FetchedAt == 0 {
		cost.FetchedAt = now
	}
	if cost.UpdatedAt == 0 {
		cost.UpdatedAt = now
	}
	if cost.CreatedAt == 0 {
		cost.CreatedAt = now
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "channel_id"}, {Name: "model"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"input_usd_per_million_micros", "output_usd_per_million_micros",
			"cache_read_usd_per_million_micros", "cache_write_usd_per_million_micros",
			"source", "manual", "valid", "error", "fetched_at", "updated_at",
		}),
	}).Create(cost).Error
}

func GetChannelModelCost(channelID int, model string) (*ChannelModelCost, error) {
	if channelID <= 0 || strings.TrimSpace(model) == "" {
		return nil, errors.New("invalid channel model cost lookup")
	}
	if DB == nil {
		return nil, errors.New("database is not initialized")
	}
	var cost ChannelModelCost
	err := DB.Where("channel_id = ? AND model = ?", channelID, strings.TrimSpace(model)).First(&cost).Error
	if err != nil {
		return nil, err
	}
	return &cost, nil
}

// ChannelModelCostVersion returns the newest update marker for a channel. A
// zero result means that the channel has never been synchronized.
func ChannelModelCostVersion(channelID int) (int64, error) {
	if channelID <= 0 || DB == nil {
		return 0, errors.New("invalid channel model cost version lookup")
	}
	var version *int64
	if err := DB.Model(&ChannelModelCost{}).Where("channel_id = ?", channelID).Select("MAX(updated_at)").Scan(&version).Error; err != nil {
		return 0, err
	}
	if version == nil {
		return 0, nil
	}
	return *version, nil
}

func ListChannelModelCosts(channelID int) ([]ChannelModelCost, error) {
	if channelID <= 0 || DB == nil {
		return nil, errors.New("invalid channel model cost list request")
	}
	var costs []ChannelModelCost
	err := DB.Where("channel_id = ?", channelID).Order("model ASC").Find(&costs).Error
	return costs, err
}

// ApplyChannelModelCosts applies one fetched snapshot. Manual values and a
// higher-precedence source are never overwritten by an automatic fetch.
func ApplyChannelModelCosts(channelID int, expectedVersion int64, costs []ChannelModelCost) error {
	if channelID <= 0 || DB == nil {
		return errors.New("invalid channel model cost apply request")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var current *int64
		if err := tx.Model(&ChannelModelCost{}).Where("channel_id = ?", channelID).Select("MAX(updated_at)").Scan(&current).Error; err != nil {
			return err
		}
		actual := int64(0)
		if current != nil {
			actual = *current
		}
		if actual != expectedVersion {
			return ErrChannelModelCostConflict
		}
		for i := range costs {
			cost := costs[i]
			cost.ChannelID = channelID
			if err := cost.Validate(); err != nil {
				return err
			}
			var existing ChannelModelCost
			err := tx.Where("channel_id = ? AND model = ?", channelID, cost.Model).First(&existing).Error
			if err == nil && (existing.Manual || sourceRank(existing.Source) > sourceRank(cost.Source)) {
				continue
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			cost.UpdatedAt = time.Now().UnixNano()
			if cost.FetchedAt == 0 {
				cost.FetchedAt = time.Now().Unix()
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "channel_id"}, {Name: "model"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"input_usd_per_million_micros", "output_usd_per_million_micros",
					"cache_read_usd_per_million_micros", "cache_write_usd_per_million_micros",
					"source", "manual", "valid", "error", "fetched_at", "updated_at",
				}),
			}).Create(&cost).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

var ErrChannelModelCostConflict = errors.New("channel model costs changed; reload before applying")

func sourceRank(source string) int {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "manual":
		return 100
	case "openrouter":
		return 20
	case "models.dev":
		return 10
	default:
		return 0
	}
}
