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
	ErrInvalidProviderFunding      = errors.New("invalid provider funding")
	ErrProviderFundingConflict     = errors.New("provider funding reference conflict")
	ErrProviderFundingInsufficient = errors.New("insufficient provider funding")
)

const (
	ProviderCashFlowDirectionIncoming = "incoming"
	ProviderCashFlowDirectionOutgoing = "outgoing"
	ProviderCashFlowTypeFunding       = "funding"
	ProviderCashFlowTypeAllocation    = "allocation"
)

// ProviderCashFlow is the canonical immutable cash-flow log for provider
// funding. AmountMinor is always positive; Direction determines its effect on
// the aggregate available balance.
type ProviderCashFlow struct {
	ID                int64  `json:"id" gorm:"primaryKey"`
	ChannelID         int    `json:"channel_id" gorm:"not null;index"`
	UserID            int    `json:"user_id" gorm:"not null;index"`
	Direction         string `json:"direction" gorm:"type:varchar(16);not null;index;uniqueIndex:idx_provider_cash_flow_reference_direction,priority:2"`
	Type              string `json:"type" gorm:"type:varchar(16);not null;index"`
	Reference         string `json:"reference" gorm:"type:varchar(255);not null;uniqueIndex:idx_provider_cash_flow_reference_direction,priority:1"`
	AmountMinor       int64  `json:"amount_minor" gorm:"not null"`
	Currency          string `json:"currency" gorm:"type:varchar(16);not null;index"`
	Source            string `json:"source" gorm:"type:varchar(16);not null;index"`
	ProviderReference string `json:"provider_reference" gorm:"type:varchar(255)"`
	Note              string `json:"note" gorm:"type:text"`
	Quota             int    `json:"quota" gorm:"not null"`
	ExchangeRate      string `json:"exchange_rate" gorm:"type:varchar(64)"`
	CreatedAt         int64  `json:"created_at" gorm:"not null;index"`
	CreatedBy         string `json:"created_by" gorm:"type:varchar(255);index"`
}

type ProviderCashFlowHistory struct {
	ProviderCashFlow
	UserName         string `json:"user_name"`
	UserDisplayName  string `json:"user_display_name"`
	CounterpartyName string `json:"counterparty_name"`
}

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
	ChannelID      int    `json:"channel_id"`
	Currency       string `json:"currency"`
	AmountMinor    int64  `json:"amount_minor"`
	AllocatedMinor int64  `json:"allocated_minor"`
	AvailableMinor int64  `json:"available_minor"`
}

type ProviderFundingChannelSummary struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
}

// ProviderFundingAllocation is an immutable debit from the aggregate provider
// pool. It intentionally has no channel ID: provider money is fungible across
// all channels.
type ProviderFundingAllocation struct {
	ID           int64  `json:"id" gorm:"primaryKey"`
	UserID       int    `json:"user_id" gorm:"not null;index"`
	Reference    string `json:"reference" gorm:"type:varchar(255);not null;uniqueIndex"`
	AmountMinor  int64  `json:"amount_minor" gorm:"not null"`
	Currency     string `json:"currency" gorm:"type:varchar(16);not null;index"`
	Quota        int    `json:"quota" gorm:"not null"`
	ExchangeRate string `json:"exchange_rate" gorm:"type:varchar(64);not null"`
	CreatedAt    int64  `json:"created_at" gorm:"not null;index"`
	CreatedBy    string `json:"created_by" gorm:"type:varchar(255);index"`
}

// ProviderFundingAllocationHistory is the read model used by the provider
// wallet UI. Allocation rows are only created after the quota credit succeeds,
// so each row represents a completed debit from the provider balance.
type ProviderFundingAllocationHistory struct {
	ID               int64  `json:"id"`
	UserID           int    `json:"user_id"`
	UserName         string `json:"user_name"`
	UserDisplayName  string `json:"user_display_name"`
	CounterpartyName string `json:"counterparty_name"`
	Reference        string `json:"reference"`
	AmountMinor      int64  `json:"amount_minor"`
	Currency         string `json:"currency"`
	Quota            int    `json:"quota"`
	ExchangeRate     string `json:"exchange_rate"`
	CreatedAt        int64  `json:"created_at"`
	CreatedBy        string `json:"created_by"`
}

func normalizeProviderFundingCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func migrateProviderCashFlows() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable(&ProviderFunding{}) {
			var fundings []ProviderFunding
			if err := tx.Find(&fundings).Error; err != nil {
				return err
			}
			for i := range fundings {
				flow := ProviderCashFlow{ChannelID: fundings[i].ChannelID, Direction: ProviderCashFlowDirectionIncoming, Type: ProviderCashFlowTypeFunding, Reference: fundings[i].Reference, AmountMinor: fundings[i].AmountMinor, Currency: fundings[i].Currency, Source: fundings[i].Source, ProviderReference: fundings[i].ProviderReference, Note: fundings[i].Note, CreatedAt: fundings[i].CreatedAt, CreatedBy: fundings[i].CreatedBy}
				if err := tx.Where("reference = ? AND direction = ?", flow.Reference, flow.Direction).FirstOrCreate(&flow).Error; err != nil {
					return err
				}
			}
		}
		if tx.Migrator().HasTable(&ProviderFundingAllocation{}) {
			var allocations []ProviderFundingAllocation
			if err := tx.Find(&allocations).Error; err != nil {
				return err
			}
			for i := range allocations {
				flow := ProviderCashFlow{UserID: allocations[i].UserID, Direction: ProviderCashFlowDirectionOutgoing, Type: ProviderCashFlowTypeAllocation, Reference: allocations[i].Reference, AmountMinor: allocations[i].AmountMinor, Currency: allocations[i].Currency, Source: "allocation", Quota: allocations[i].Quota, ExchangeRate: allocations[i].ExchangeRate, CreatedAt: allocations[i].CreatedAt, CreatedBy: allocations[i].CreatedBy}
				if err := tx.Where("reference = ? AND direction = ?", flow.Reference, flow.Direction).FirstOrCreate(&flow).Error; err != nil {
					return err
				}
			}
		}
		// Legacy rows were not normalized before the cash-flow table became the
		// source of truth. Normalize both backfilled and already migrated rows so
		// currencies cannot be split into separate wallets by letter case.
		if tx.Migrator().HasTable(&ProviderCashFlow{}) {
			var flows []ProviderCashFlow
			if err := tx.Find(&flows).Error; err != nil {
				return err
			}
			for i := range flows {
				currency := normalizeProviderFundingCurrency(flows[i].Currency)
				if currency == flows[i].Currency {
					continue
				}
				if err := tx.Model(&ProviderCashFlow{}).Where("id = ?", flows[i].ID).Update("currency", currency).Error; err != nil {
					return err
				}
			}
		}
		if tx.Migrator().HasTable(&ProviderFunding{}) {
			if err := tx.Migrator().DropTable(&ProviderFunding{}); err != nil {
				return err
			}
		}
		if tx.Migrator().HasTable(&ProviderFundingAllocation{}) {
			if err := tx.Migrator().DropTable(&ProviderFundingAllocation{}); err != nil {
				return err
			}
		}
		return nil
	})
}

func ListProviderCashFlows(page, pageSize int) ([]ProviderCashFlowHistory, int64, error) {
	if err := migrateProviderCashFlows(); err != nil {
		return nil, 0, err
	}
	query := DB.Model(&ProviderCashFlow{}).Order("provider_cash_flows.created_at DESC, provider_cash_flows.id DESC")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page = max(page, 1)
	pageSize = min(max(pageSize, 1), 100)
	var rows []ProviderCashFlowHistory
	err := query.Select("provider_cash_flows.*, users.username AS user_name, users.display_name AS user_display_name").Joins("LEFT JOIN users ON users.id = provider_cash_flows.user_id").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	for i := range rows {
		displayName := strings.TrimSpace(rows[i].UserDisplayName)
		username := strings.TrimSpace(rows[i].UserName)
		if displayName != "" && username != "" && displayName != username {
			rows[i].CounterpartyName = displayName + " (" + username + ")"
		} else if displayName != "" {
			rows[i].CounterpartyName = displayName
		} else {
			rows[i].CounterpartyName = username
		}
	}
	return rows, total, err
}

func providerCashFlowTotals(currency string) (int64, int64, error) {
	if err := migrateProviderCashFlows(); err != nil {
		return 0, 0, err
	}
	var incoming, outgoing int64
	query := DB.Model(&ProviderCashFlow{}).Where("currency = ?", normalizeProviderFundingCurrency(currency))
	if err := query.Where("direction = ?", ProviderCashFlowDirectionIncoming).Select("COALESCE(SUM(amount_minor), 0)").Scan(&incoming).Error; err != nil {
		return 0, 0, err
	}
	if err := query.Where("direction = ?", ProviderCashFlowDirectionOutgoing).Select("COALESCE(SUM(amount_minor), 0)").Scan(&outgoing).Error; err != nil {
		return 0, 0, err
	}
	return incoming, outgoing, nil
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

	if err = migrateProviderCashFlows(); err != nil {
		return nil, false, err
	}
	flow := &ProviderCashFlow{}
	if lookupErr := DB.Where("reference = ? AND direction = ?", funding.Reference, ProviderCashFlowDirectionIncoming).First(flow).Error; lookupErr == nil {
		if flow.ChannelID != funding.ChannelID || flow.AmountMinor != funding.AmountMinor || flow.Currency != funding.Currency || flow.Source != funding.Source {
			return nil, false, ErrProviderFundingConflict
		}
		return &ProviderFunding{ID: flow.ID, ChannelID: flow.ChannelID, Reference: flow.Reference, AmountMinor: flow.AmountMinor, Currency: flow.Currency, Source: flow.Source, ProviderReference: flow.ProviderReference, Note: flow.Note, CreatedAt: flow.CreatedAt, CreatedBy: flow.CreatedBy}, true, nil
	}

	funding.CreatedAt = common.GetTimestamp()
	flow = &ProviderCashFlow{ChannelID: funding.ChannelID, Direction: ProviderCashFlowDirectionIncoming, Type: ProviderCashFlowTypeFunding, Reference: funding.Reference, AmountMinor: funding.AmountMinor, Currency: funding.Currency, Source: funding.Source, ProviderReference: funding.ProviderReference, Note: funding.Note, CreatedAt: funding.CreatedAt, CreatedBy: funding.CreatedBy}
	if err = DB.Create(flow).Error; err != nil {
		if DB.Where("reference = ? AND direction = ?", funding.Reference, ProviderCashFlowDirectionIncoming).First(flow).Error == nil {
			if flow.ChannelID == funding.ChannelID && flow.AmountMinor == funding.AmountMinor && flow.Currency == funding.Currency && flow.Source == funding.Source {
				return &ProviderFunding{ID: flow.ID, ChannelID: flow.ChannelID, Reference: flow.Reference, AmountMinor: flow.AmountMinor, Currency: flow.Currency, Source: flow.Source, ProviderReference: flow.ProviderReference, Note: flow.Note, CreatedAt: flow.CreatedAt, CreatedBy: flow.CreatedBy}, true, nil
			}
			return nil, false, ErrProviderFundingConflict
		}
		return nil, false, err
	}
	funding.ID = flow.ID
	return funding, false, nil
}

func ListProviderFundings(filter ProviderFundingFilter) ([]ProviderFunding, int64, error) {
	if err := migrateProviderCashFlows(); err != nil {
		return nil, 0, err
	}
	query := DB.Model(&ProviderCashFlow{}).Where("direction = ?", ProviderCashFlowDirectionIncoming)
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
	var flows []ProviderCashFlow
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&flows).Error
	items := make([]ProviderFunding, len(flows))
	for i := range flows {
		items[i] = ProviderFunding{ID: flows[i].ID, ChannelID: flows[i].ChannelID, Reference: flows[i].Reference, AmountMinor: flows[i].AmountMinor, Currency: flows[i].Currency, Source: flows[i].Source, ProviderReference: flows[i].ProviderReference, Note: flows[i].Note, CreatedAt: flows[i].CreatedAt, CreatedBy: flows[i].CreatedBy}
	}
	return items, total, err
}

func SummarizeProviderFundings(filter ProviderFundingFilter) ([]ProviderFundingSummary, error) {
	if err := migrateProviderCashFlows(); err != nil {
		return nil, err
	}
	query := DB.Model(&ProviderCashFlow{}).Where("direction = ?", ProviderCashFlowDirectionIncoming)
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
	if err := migrateProviderCashFlows(); err != nil {
		return nil, err
	}
	err := DB.Model(&ProviderCashFlow{}).
		Select("0 AS channel_id, currency, SUM(CASE WHEN direction = ? THEN amount_minor ELSE -amount_minor END) AS amount_minor, SUM(CASE WHEN direction = ? THEN amount_minor ELSE 0 END) AS allocated_minor, SUM(CASE WHEN direction = ? THEN amount_minor ELSE -amount_minor END) AS available_minor", ProviderCashFlowDirectionIncoming, ProviderCashFlowDirectionOutgoing, ProviderCashFlowDirectionIncoming).
		Group("currency").Order("currency ASC").Scan(&wallets).Error
	if err != nil {
		return nil, err
	}
	return wallets, nil
}

func SummarizeProviderFundingByChannel() ([]ProviderFundingChannelSummary, error) {
	var summaries []ProviderFundingChannelSummary
	if err := migrateProviderCashFlows(); err != nil {
		return nil, err
	}
	err := DB.Model(&ProviderCashFlow{}).
		Select("provider_cash_flows.channel_id, channels.name AS channel_name, provider_cash_flows.currency, SUM(provider_cash_flows.amount_minor) AS amount_minor").
		Where("provider_cash_flows.direction = ?", ProviderCashFlowDirectionIncoming).
		Joins("LEFT JOIN channels ON channels.id = provider_cash_flows.channel_id").
		Group("provider_cash_flows.channel_id, channels.name, provider_cash_flows.currency").
		Order("channel_id ASC, currency ASC").
		Scan(&summaries).Error
	return summaries, err
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
	if err := migrateProviderCashFlows(); err != nil {
		return nil, false, err
	}
	var existingFlow ProviderCashFlow
	if err := DB.Where("reference = ? AND direction = ?", allocation.Reference, ProviderCashFlowDirectionOutgoing).First(&existingFlow).Error; err == nil {
		if existingFlow.UserID != allocation.UserID || existingFlow.AmountMinor != allocation.AmountMinor || existingFlow.Currency != allocation.Currency || existingFlow.Quota != allocation.Quota {
			return nil, false, ErrProviderFundingConflict
		}
		return &ProviderFundingAllocation{ID: existingFlow.ID, UserID: existingFlow.UserID, Reference: existingFlow.Reference, AmountMinor: existingFlow.AmountMinor, Currency: existingFlow.Currency, Quota: existingFlow.Quota, ExchangeRate: existingFlow.ExchangeRate, CreatedAt: existingFlow.CreatedAt, CreatedBy: existingFlow.CreatedBy}, true, nil
	}
	var committed ProviderFundingAllocation
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).First(&user, allocation.UserID).Error; err != nil {
			return err
		}
		var deposits []ProviderCashFlow
		if err := lockForUpdate(tx).Where("currency = ? AND direction = ?", allocation.Currency, ProviderCashFlowDirectionIncoming).Find(&deposits).Error; err != nil {
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
		if err := tx.Model(&ProviderCashFlow{}).Where("currency = ? AND direction = ?", allocation.Currency, ProviderCashFlowDirectionOutgoing).Select("COALESCE(SUM(amount_minor), 0)").Scan(&allocated).Error; err != nil {
			return err
		}
		if allocation.AmountMinor > funded-allocated {
			return ErrProviderFundingInsufficient
		}
		if user.Quota > common.MaxWalletQuota || int64(user.Quota)+int64(allocation.Quota) > int64(common.MaxWalletQuota) {
			return ErrWalletQuotaLimitExceeded
		}
		allocation.CreatedAt = common.GetTimestamp()
		flow := &ProviderCashFlow{UserID: allocation.UserID, Direction: ProviderCashFlowDirectionOutgoing, Type: ProviderCashFlowTypeAllocation, Reference: allocation.Reference, AmountMinor: allocation.AmountMinor, Currency: allocation.Currency, Source: "allocation", Quota: allocation.Quota, ExchangeRate: allocation.ExchangeRate, CreatedAt: allocation.CreatedAt, CreatedBy: allocation.CreatedBy}
		if err := tx.Create(flow).Error; err != nil {
			return err
		}
		allocation.ID = flow.ID
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", gorm.Expr("quota + ?", allocation.Quota)).Error; err != nil {
			return err
		}
		committed = *allocation
		return nil
	})
	if err != nil {
		if lookupErr := DB.Where("reference = ? AND direction = ?", allocation.Reference, ProviderCashFlowDirectionOutgoing).First(&existingFlow).Error; lookupErr == nil {
			if existingFlow.UserID == allocation.UserID && existingFlow.AmountMinor == allocation.AmountMinor && existingFlow.Currency == allocation.Currency && existingFlow.Quota == allocation.Quota {
				return &ProviderFundingAllocation{ID: existingFlow.ID, UserID: existingFlow.UserID, Reference: existingFlow.Reference, AmountMinor: existingFlow.AmountMinor, Currency: existingFlow.Currency, Quota: existingFlow.Quota, ExchangeRate: existingFlow.ExchangeRate, CreatedAt: existingFlow.CreatedAt, CreatedBy: existingFlow.CreatedBy}, true, nil
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

func ListProviderFundingAllocationHistory() ([]ProviderFundingAllocationHistory, error) {
	var allocations []ProviderFundingAllocationHistory
	if err := migrateProviderCashFlows(); err != nil {
		return nil, err
	}
	err := DB.Model(&ProviderCashFlow{}).
		Where("provider_cash_flows.direction = ?", ProviderCashFlowDirectionOutgoing).
		Select("provider_cash_flows.id, provider_cash_flows.user_id, users.username AS user_name, users.display_name AS user_display_name, provider_cash_flows.reference, provider_cash_flows.amount_minor, provider_cash_flows.currency, provider_cash_flows.quota, provider_cash_flows.exchange_rate, provider_cash_flows.created_at, provider_cash_flows.created_by").
		Joins("LEFT JOIN users ON users.id = provider_cash_flows.user_id").
		Order("provider_cash_flows.created_at DESC, provider_cash_flows.id DESC").Find(&allocations).Error
	for i := range allocations {
		displayName := strings.TrimSpace(allocations[i].UserDisplayName)
		username := strings.TrimSpace(allocations[i].UserName)
		switch {
		case displayName != "" && username != "" && displayName != username:
			allocations[i].CounterpartyName = displayName + " (" + username + ")"
		case displayName != "":
			allocations[i].CounterpartyName = displayName
		case username != "":
			allocations[i].CounterpartyName = username
		default:
			allocations[i].CounterpartyName = ""
		}
	}
	return allocations, err
}
