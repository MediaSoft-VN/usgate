package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrTenantFundingConflict     = errors.New("tenant funding reference conflict")
	ErrTenantFundingInsufficient = errors.New("insufficient tenant wallet balance")
	ErrTenantFundingForbidden    = errors.New("tenant funding member is not active")
)

// TenantFunding is the immutable audit/history record for a transfer from a
// tenant's organization wallet to a member's quota and financial wallet.
type TenantFunding struct {
	ID           int64  `json:"id" gorm:"primaryKey"`
	TenantID     int    `json:"tenant_id" gorm:"not null;index;uniqueIndex:idx_tenant_funding_reference,priority:1"`
	UserID       int    `json:"user_id" gorm:"not null;index"`
	Reference    string `json:"reference" gorm:"type:varchar(255);not null;uniqueIndex:idx_tenant_funding_reference,priority:2"`
	AmountMinor  int64  `json:"amount_minor" gorm:"not null"`
	Currency     string `json:"currency" gorm:"type:varchar(16);not null;index"`
	Quota        int    `json:"quota" gorm:"not null"`
	ExchangeRate string `json:"exchange_rate" gorm:"type:varchar(64);not null"`
	CreatedAt    int64  `json:"created_at" gorm:"not null;index"`
	CreatedBy    string `json:"created_by" gorm:"type:varchar(255);index"`
}

type TenantFundingHistory struct {
	TenantFunding
	Username    string `json:"username" gorm:"-"`
	DisplayName string `json:"display_name" gorm:"-"`
}

// FundTenantMember atomically debits the tenant organization wallet, credits
// the member's financial wallet and legacy quota, and records the transfer.
// The tenant membership is checked inside the transaction to prevent a stale
// UI selection from funding a removed or cross-tenant user.
func FundTenantMember(tenantID, userID int, reference, currency string, amountMinor int64, quota int, exchangeRate, createdBy string) (*TenantFunding, bool, error) {
	reference = strings.TrimSpace(reference)
	currency = normalizeFinancialCurrencyForFunding(currency)
	if tenantID <= 0 || userID <= 0 || reference == "" || amountMinor <= 0 || quota <= 0 || currency == "" {
		return nil, false, ErrInvalidProviderFunding
	}
	if err := ValidateFinancialMinorAmount(amountMinor); err != nil {
		return nil, false, err
	}
	var existing TenantFunding
	if err := DB.Where("tenant_id = ? AND reference = ?", tenantID, reference).First(&existing).Error; err == nil {
		if existing.UserID != userID || existing.AmountMinor != amountMinor || existing.Currency != currency || existing.Quota != quota {
			return nil, false, ErrTenantFundingConflict
		}
		return &existing, true, nil
	}
	var membership TenantMembership
	if err := DB.Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, MembershipActive).First(&membership).Error; err != nil {
		return nil, false, ErrTenantFundingForbidden
	}
	userWallet, err := GetOrCreateWalletAccount(FinancialOwnerUser, int64(userID), currency)
	if err != nil {
		return nil, false, err
	}
	tenantWallet, err := GetOrCreateWalletAccount(FinancialOwnerOrganization, int64(tenantID), currency)
	if err != nil {
		return nil, false, err
	}
	committed := &TenantFunding{}
	replay := false
	err = DB.Transaction(func(tx *gorm.DB) error {
		var membership TenantMembership
		if err := tx.Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, MembershipActive).First(&membership).Error; err != nil {
			return ErrTenantFundingForbidden
		}
		var user User
		if err := lockForUpdate(tx).Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		if user.Status != common.UserStatusEnabled {
			return ErrTenantFundingForbidden
		}
		var wallet WalletAccount
		if err := lockForUpdate(tx).First(&wallet, tenantWallet.ID).Error; err != nil {
			return err
		}
		if wallet.Status != WalletAccountActive || wallet.Currency != currency || wallet.AvailableMinor < amountMinor {
			return ErrTenantFundingInsufficient
		}
		if user.Quota > common.MaxWalletQuota || int64(user.Quota)+int64(quota) > int64(common.MaxWalletQuota) {
			return ErrWalletQuotaLimitExceeded
		}
		if err := lockForUpdate(tx).First(userWallet, userWallet.ID).Error; err != nil {
			return err
		}
		if userWallet.AvailableMinor > FinancialMaxMinorUnits-amountMinor {
			return ErrFinancialAmountOverflow
		}
		now := common.GetTimestamp()
		funding := &TenantFunding{TenantID: tenantID, UserID: userID, Reference: reference, AmountMinor: amountMinor, Currency: currency, Quota: quota, ExchangeRate: exchangeRate, CreatedAt: now, CreatedBy: createdBy}
		if err := tx.Create(funding).Error; err != nil {
			if tx.Where("tenant_id = ? AND reference = ?", tenantID, reference).First(&existing).Error == nil {
				if existing.UserID == userID && existing.AmountMinor == amountMinor && existing.Currency == currency && existing.Quota == quota {
					*committed = existing
					replay = true
					return nil
				}
				return ErrTenantFundingConflict
			}
			return err
		}
		transaction := &FinancialTransaction{
			TransactionNo: fmt.Sprintf("tenant-funding-%d", funding.ID), WalletAccountID: wallet.ID,
			IdempotencyKey: reference, Type: FinancialTransactionTransfer, Status: FinancialTransactionCompleted,
			AmountMinor: amountMinor, Currency: currency, ExternalReference: reference, CreatedAt: now, CompletedAt: now,
		}
		if err := tx.Create(transaction).Error; err != nil {
			return err
		}
		entries := []LedgerEntry{
			{FinancialTransactionID: transaction.ID, WalletAccountID: wallet.ID, EntryType: LedgerEntryDebit, AmountMinor: amountMinor, Currency: currency, CreatedAt: now},
			{FinancialTransactionID: transaction.ID, WalletAccountID: userWallet.ID, EntryType: LedgerEntryCredit, AmountMinor: amountMinor, Currency: currency, CreatedAt: now},
		}
		if err := tx.Create(&entries).Error; err != nil {
			return err
		}
		if err := tx.Model(&WalletAccount{}).Where("id = ? AND version = ?", wallet.ID, wallet.Version).Updates(map[string]any{"available_minor": wallet.AvailableMinor - amountMinor, "updated_at": now, "version": wallet.Version + 1}).Error; err != nil {
			return err
		}
		if err := tx.Model(&WalletAccount{}).Where("id = ? AND version = ?", userWallet.ID, userWallet.Version).Updates(map[string]any{"available_minor": userWallet.AvailableMinor + amountMinor, "updated_at": now, "version": userWallet.Version + 1}).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", userID).Update("quota", gorm.Expr("quota + ?", quota)).Error; err != nil {
			return err
		}
		*committed = *funding
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if replay {
		return committed, true, nil
	}
	if err := cacheIncrUserQuota(userID, int64(quota)); err != nil {
		common.SysError(common.LogText("failed to sync tenant funding quota for user %d: %s", userID, err))
	}
	return committed, false, nil
}

func normalizeFinancialCurrencyForFunding(currency string) string {
	code, err := normalizeFinancialCurrency(currency)
	if err != nil {
		return ""
	}
	return code
}

func ListTenantFunding(tenantID int) ([]TenantFundingHistory, error) {
	var rows []TenantFundingHistory
	err := DB.Model(&TenantFunding{}).
		Select("tenant_fundings.*, users.username, users.display_name").
		Joins("LEFT JOIN users ON users.id = tenant_fundings.user_id").
		Where("tenant_fundings.tenant_id = ?", tenantID).
		Order("tenant_fundings.created_at DESC, tenant_fundings.id DESC").Find(&rows).Error
	return rows, err
}
