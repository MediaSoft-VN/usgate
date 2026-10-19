package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	ProviderFundingSourceManual = "manual"
	ProviderFundingSourceSync   = "sync"
)

var (
	ErrInvalidProviderFunding  = errors.New("invalid provider funding")
	ErrProviderFundingConflict = errors.New("provider funding reference conflict")
)

// ProviderFunding is an immutable record of money deposited into a provider
// partner's wallet. ChannelID is optional provenance for the channel that
// received the deposit. AmountMinor is stored in the currency's smallest unit.
type ProviderFunding struct {
	ID                int64  `json:"id" gorm:"primaryKey"`
	ProviderPartnerID int64  `json:"provider_partner_id" gorm:"index"`
	ChannelID         int    `json:"channel_id" gorm:"index"`
	Reference         string `json:"reference" gorm:"type:varchar(255);not null;uniqueIndex"`
	AmountMinor       int64  `json:"amount_minor" gorm:"not null"`
	Currency          string `json:"currency" gorm:"type:varchar(16);not null;index"`
	Source            string `json:"source" gorm:"type:varchar(16);not null;index"`
	ProviderReference string `json:"provider_reference" gorm:"type:varchar(255)"`
	Note              string `json:"note" gorm:"type:text"`
	CreatedAt         int64  `json:"created_at" gorm:"not null;index"`
}

type ProviderFundingFilter struct {
	ProviderPartnerID int64
	ChannelID         int
	Currency          string
	Source            string
	Page              int
	PageSize          int
}

type ProviderFundingSummary struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
}

func normalizeProviderFundingCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func validateProviderFunding(funding *ProviderFunding) error {
	if funding == nil || funding.ProviderPartnerID <= 0 || funding.AmountMinor <= 0 || strings.TrimSpace(funding.Reference) == "" {
		return ErrInvalidProviderFunding
	}
	funding.Currency = normalizeProviderFundingCurrency(funding.Currency)
	if len(funding.Currency) < 2 || len(funding.Currency) > 16 {
		return ErrInvalidProviderFunding
	}
	switch funding.Source {
	case ProviderFundingSourceManual, ProviderFundingSourceSync:
	default:
		return ErrInvalidProviderFunding
	}
	return nil
}

// RecordProviderFunding records one manual or synchronized provider deposit.
// Replaying the same reference is idempotent when its business fields match.
func RecordProviderFunding(funding *ProviderFunding) (record *ProviderFunding, replay bool, err error) {
	if err = validateProviderFunding(funding); err != nil {
		return nil, false, err
	}
	if _, err = GetProviderPartner(funding.ProviderPartnerID); err != nil {
		return nil, false, err
	}
	if funding.ChannelID > 0 {
		var channel Channel
		if err = DB.Where("id = ?", funding.ChannelID).First(&channel).Error; err != nil {
			return nil, false, err
		}
		if channel.ProviderPartnerID == nil || *channel.ProviderPartnerID != funding.ProviderPartnerID {
			return nil, false, ErrInvalidProviderFunding
		}
	}

	existing := &ProviderFunding{}
	if lookupErr := DB.Where("reference = ?", funding.Reference).First(existing).Error; lookupErr == nil {
		if existing.ProviderPartnerID != funding.ProviderPartnerID || existing.AmountMinor != funding.AmountMinor || existing.Currency != funding.Currency || existing.Source != funding.Source {
			return nil, false, ErrProviderFundingConflict
		}
		return existing, true, nil
	}

	funding.CreatedAt = common.GetTimestamp()
	if err = DB.Create(funding).Error; err != nil {
		if DB.Where("reference = ?", funding.Reference).First(existing).Error == nil {
			if existing.ProviderPartnerID == funding.ProviderPartnerID && existing.AmountMinor == funding.AmountMinor && existing.Currency == funding.Currency && existing.Source == funding.Source {
				return existing, true, nil
			}
			return nil, false, ErrProviderFundingConflict
		}
		return nil, false, err
	}
	return funding, false, nil
}

func ListProviderFundings(filter ProviderFundingFilter) ([]ProviderFunding, int64, error) {
	query := DB.Model(&ProviderFunding{})
	if filter.ProviderPartnerID > 0 {
		query = query.Where("provider_partner_id = ?", filter.ProviderPartnerID)
	}
	if filter.ChannelID > 0 {
		query = query.Where("channel_id = ?", filter.ChannelID)
	}
	if filter.Currency != "" {
		query = query.Where("currency = ?", normalizeProviderFundingCurrency(filter.Currency))
	}
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := max(filter.Page, 1)
	pageSize := min(max(filter.PageSize, 1), 100)
	var items []ProviderFunding
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func SummarizeProviderFundings(filter ProviderFundingFilter) ([]ProviderFundingSummary, error) {
	query := DB.Model(&ProviderFunding{})
	if filter.ProviderPartnerID > 0 {
		query = query.Where("provider_partner_id = ?", filter.ProviderPartnerID)
	}
	if filter.Currency != "" {
		query = query.Where("currency = ?", normalizeProviderFundingCurrency(filter.Currency))
	}
	var summary []ProviderFundingSummary
	err := query.
		Select("currency, SUM(amount_minor) AS amount_minor").
		Group("currency").Order("currency ASC").Scan(&summary).Error
	return summary, err
}
