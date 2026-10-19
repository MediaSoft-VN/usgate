package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ProviderFundingSourceManual = "manual"
	ProviderFundingSourceSync   = "sync"
)

var (
	ErrInvalidProviderFunding  = errors.New("invalid provider funding")
	ErrProviderFundingConflict = errors.New("provider funding reference conflict")
	ErrProviderFundingInsufficient = errors.New("insufficient provider funding")
)

// ProviderFunding is an immutable record of money deposited into a provider
// channel's wallet. AmountMinor is stored in the currency's smallest unit.
type ProviderFunding struct {
	ID                int64  `json:"id" gorm:"primaryKey"`
	ChannelID         int    `json:"channel_id" gorm:"not null;index"`
	Reference         string `json:"reference" gorm:"type:varchar(255);not null;uniqueIndex"`
	AmountMinor       int64  `json:"amount_minor" gorm:"not null"`
	Currency          string `json:"currency" gorm:"type:varchar(16);not null;index"`
	Source            string `json:"source" gorm:"type:varchar(16);not null;index"`
	ProviderReference string `json:"provider_reference" gorm:"type:varchar(255)"`
	Note              string `json:"note" gorm:"type:text"`
	CreatedAt         int64  `json:"created_at" gorm:"not null;index"`
	CreatedBy         string `json:"created_by" gorm:"type:varchar(255);index"`
}

type ProviderFundingFilter struct {
	ChannelID int
	Currency  string
	Source    string
	Page      int
	PageSize  int
}

type ProviderFundingSummary struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
}

type ProviderFundingWalletSummary struct {
	ChannelID   int    `json:"channel_id"`
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
	AllocatedMinor int64 `json:"allocated_minor"`
	AvailableMinor int64 `json:"available_minor"`
}

// ProviderFundingAllocation is an immutable debit from the aggregate provider
// pool. It intentionally has no channel ID: provider money is fungible across
// all channels.
type ProviderFundingAllocation struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	UserID      int    `json:"user_id" gorm:"not null;index"`
	Reference   string `json:"reference" gorm:"type:varchar(255);not null;uniqueIndex"`
	AmountMinor int64  `json:"amount_minor" gorm:"not null"`
	Currency    string `json:"currency" gorm:"type:varchar(16);not null;index"`
	Quota       int    `json:"quota" gorm:"not null"`
	ExchangeRate string `json:"exchange_rate" gorm:"type:varchar(64);not null"`
	CreatedAt   int64  `json:"created_at" gorm:"not null;index"`
	CreatedBy   string `json:"created_by" gorm:"type:varchar(255);index"`
}

func normalizeProviderFundingCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func validateProviderFunding(funding *ProviderFunding) error {
	if funding == nil || funding.ChannelID <= 0 || funding.AmountMinor <= 0 || strings.TrimSpace(funding.Reference) == "" {
		return ErrInvalidProviderFunding
	}
	if err := ValidateFinancialMinorAmount(funding.AmountMinor); err != nil {
		return err
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
	var channel Channel
	if err = DB.Where("id = ?", funding.ChannelID).First(&channel).Error; err != nil {
		return nil, false, err
	}

	existing := &ProviderFunding{}
	if lookupErr := DB.Where("reference = ?", funding.Reference).First(existing).Error; lookupErr == nil {
		if existing.ChannelID != funding.ChannelID || existing.AmountMinor != funding.AmountMinor || existing.Currency != funding.Currency || existing.Source != funding.Source {
			return nil, false, ErrProviderFundingConflict
		}
		return existing, true, nil
	}

	funding.CreatedAt = common.GetTimestamp()
	if err = DB.Create(funding).Error; err != nil {
		if DB.Where("reference = ?", funding.Reference).First(existing).Error == nil {
			if existing.ChannelID == funding.ChannelID && existing.AmountMinor == funding.AmountMinor && existing.Currency == funding.Currency && existing.Source == funding.Source {
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
	if filter.Currency != "" {
		query = query.Where("currency = ?", normalizeProviderFundingCurrency(filter.Currency))
	}
	var summary []ProviderFundingSummary
	err := query.
		Select("currency, SUM(amount_minor) AS amount_minor").
		Group("currency").Order("currency ASC").Scan(&summary).Error
	return summary, err
}

func SummarizeProviderFundingWallets() ([]ProviderFundingWalletSummary, error) {
	var wallets []ProviderFundingWalletSummary
	err := DB.Model(&ProviderFunding{}).
		Select("0 AS channel_id, currency, SUM(amount_minor) AS amount_minor").
		Group("currency").Order("currency ASC").Scan(&wallets).Error
	if err != nil {
		return nil, err
	}
	for i := range wallets {
		var allocated int64
		if err := DB.Model(&ProviderFundingAllocation{}).Where("currency = ?", wallets[i].Currency).Select("COALESCE(SUM(amount_minor), 0)").Scan(&allocated).Error; err != nil {
			return nil, err
		}
		wallets[i].AllocatedMinor = allocated
		wallets[i].AvailableMinor = wallets[i].AmountMinor - allocated
	}
	return wallets, nil
}

// AllocateProviderFunding atomically debits the aggregate provider pool and
// credits the selected user's legacy quota. Reusing reference is idempotent.
func AllocateProviderFunding(allocation *ProviderFundingAllocation) (*ProviderFundingAllocation, bool, error) {
	if allocation == nil || allocation.UserID <= 0 || allocation.AmountMinor <= 0 || allocation.Quota <= 0 || strings.TrimSpace(allocation.Reference) == "" {
		return nil, false, ErrInvalidProviderFunding
	}
	if err := ValidateFinancialMinorAmount(allocation.AmountMinor); err != nil {
		return nil, false, err
	}
	allocation.Currency = normalizeProviderFundingCurrency(allocation.Currency)
	if len(allocation.Currency) < 2 || len(allocation.Currency) > 16 {
		return nil, false, ErrInvalidProviderFunding
	}
	var existing ProviderFundingAllocation
	if err := DB.Where("reference = ?", allocation.Reference).First(&existing).Error; err == nil {
		if existing.UserID != allocation.UserID || existing.AmountMinor != allocation.AmountMinor || existing.Currency != allocation.Currency || existing.Quota != allocation.Quota {
			return nil, false, ErrProviderFundingConflict
		}
		return &existing, true, nil
	}
	var committed ProviderFundingAllocation
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).First(&user, allocation.UserID).Error; err != nil {
			return err
		}
		var deposits []ProviderFunding
		if err := lockForUpdate(tx).Where("currency = ?", allocation.Currency).Find(&deposits).Error; err != nil {
			return err
		}
		var funded int64
		for i := range deposits {
			if deposits[i].AmountMinor > FinancialMaxMinorUnits-funded {
				return ErrFinancialAmountOverflow
			}
			funded += deposits[i].AmountMinor
		}
		var allocated int64
		if err := tx.Model(&ProviderFundingAllocation{}).Where("currency = ?", allocation.Currency).Select("COALESCE(SUM(amount_minor), 0)").Scan(&allocated).Error; err != nil {
			return err
		}
		if allocation.AmountMinor > funded-allocated {
			return ErrProviderFundingInsufficient
		}
		if user.Quota > common.MaxWalletQuota || int64(user.Quota)+int64(allocation.Quota) > int64(common.MaxWalletQuota) {
			return ErrWalletQuotaLimitExceeded
		}
		allocation.CreatedAt = common.GetTimestamp()
		if err := tx.Create(allocation).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", gorm.Expr("quota + ?", allocation.Quota)).Error; err != nil {
			return err
		}
		committed = *allocation
		return nil
	})
	if err != nil {
		if lookupErr := DB.Where("reference = ?", allocation.Reference).First(&existing).Error; lookupErr == nil {
			if existing.UserID == allocation.UserID && existing.AmountMinor == allocation.AmountMinor && existing.Currency == allocation.Currency && existing.Quota == allocation.Quota {
				return &existing, true, nil
			}
			return nil, false, ErrProviderFundingConflict
		}
		return nil, false, err
	}
	if deltaErr := cacheIncrUserQuota(committed.UserID, int64(committed.Quota)); deltaErr != nil {
		common.SysError(common.LogText("failed to sync provider allocation quota for user %d: %s", committed.UserID, deltaErr))
	}
	return &committed, false, nil
}

func ListProviderFundingAllocations() ([]ProviderFundingAllocation, error) {
	var allocations []ProviderFundingAllocation
	err := DB.Order("created_at DESC, id DESC").Find(&allocations).Error
	return allocations, err
}
