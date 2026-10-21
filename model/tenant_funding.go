package model

import (
	"errors"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrTenantFundingConflict     = errors.New("tenant funding reference conflict")
	ErrTenantFundingInsufficient = errors.New("insufficient partner owner platform wallet quota")
	ErrTenantFundingForbidden    = errors.New("tenant funding member is not active")
	ErrTenantFundingSelf         = errors.New("tenant owner cannot fund itself")
)

// TenantFunding is the immutable audit/history record for a transfer from the
// partner owner's platform quota to a tenant member's quota.
type TenantFunding struct {
	ID           int64  `json:"id" gorm:"primaryKey"`
	TenantID     int    `json:"tenant_id" gorm:"not null;index;uniqueIndex:idx_tenant_funding_reference,priority:1"`
	OwnerUserID  int    `json:"owner_user_id" gorm:"not null;default:0;index"`
	UserID       int    `json:"user_id" gorm:"not null;index"`
	Reference    string `json:"reference" gorm:"type:varchar(255);not null;uniqueIndex:idx_tenant_funding_reference,priority:2"`
	AmountMinor  int64  `json:"amount_minor" gorm:"not null"`
	Currency     string `json:"currency" gorm:"type:varchar(16);not null;index"`
	Quota        int    `json:"quota" gorm:"not null"`
	ExchangeRate string `json:"exchange_rate" gorm:"type:varchar(64);not null"`
	CreatedAt    int64  `json:"created_at" gorm:"not null;index"`
	CreatedBy    string `json:"created_by" gorm:"type:varchar(255);index"`
}

// TenantFundingWallet is the partner owner's platform wallet projected in the
// currency used by the funding form. The organization wallet remains a
// supported legacy record, but is deliberately not read or mutated here.
type TenantFundingWallet struct {
	OwnerUserID    int    `json:"owner_user_id"`
	AvailableQuota int    `json:"available_quota"`
	AvailableMinor int64  `json:"available_minor"`
	ReservedMinor  int64  `json:"reserved_minor"`
	Currency       string `json:"currency"`
}

type TenantFundingHistory struct {
	TenantFunding
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// FundTenantMember atomically debits the partner owner's quota, credits the
// member's quota, and records the transfer. Both users are locked in a stable
// order so concurrent funding cannot overspend the owner's platform wallet.
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
	if err := common.ValidateWalletQuota(quota); err != nil {
		return nil, false, err
	}
	committed := &TenantFunding{}
	replay := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing TenantFunding
		var tenant Tenant
		if err := tx.Where("id = ? AND status = ? AND deleted_at IS NULL", tenantID, TenantActive).First(&tenant).Error; err != nil {
			return ErrTenantFundingForbidden
		}
		var ownerMembership TenantMembership
		if err := tx.Where("tenant_id = ? AND role_key = ? AND status = ?", tenantID, TenantRoleOwner, MembershipActive).First(&ownerMembership).Error; err != nil {
			return ErrTenantFundingForbidden
		}
		var memberMembership TenantMembership
		if err := tx.Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, MembershipActive).First(&memberMembership).Error; err != nil {
			return ErrTenantFundingForbidden
		}
		if ownerMembership.UserId == userID {
			return ErrTenantFundingSelf
		}
		if err := tx.Where("tenant_id = ? AND reference = ?", tenantID, reference).First(&existing).Error; err == nil {
			if existing.UserID != userID || existing.AmountMinor != amountMinor || existing.Currency != currency || existing.Quota != quota {
				return ErrTenantFundingConflict
			}
			*committed = existing
			replay = true
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		userIDs := []int{ownerMembership.UserId, userID}
		sort.Ints(userIDs)
		users := make(map[int]User, len(userIDs))
		for _, id := range userIDs {
			var user User
			if err := lockForUpdate(tx).Where("id = ?", id).First(&user).Error; err != nil {
				return err
			}
			if user.Status != common.UserStatusEnabled {
				return ErrTenantFundingForbidden
			}
			users[id] = user
		}
		owner := users[ownerMembership.UserId]
		member := users[userID]
		if owner.Quota < quota {
			return ErrTenantFundingInsufficient
		}
		if int64(member.Quota)+int64(quota) > int64(common.MaxWalletQuota) {
			return ErrWalletQuotaLimitExceeded
		}
		now := common.GetTimestamp()
		funding := &TenantFunding{TenantID: tenantID, OwnerUserID: owner.Id, UserID: userID, Reference: reference, AmountMinor: amountMinor, Currency: currency, Quota: quota, ExchangeRate: exchangeRate, CreatedAt: now, CreatedBy: createdBy}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(funding).Error; err != nil {
			return err
		}
		if funding.ID == 0 {
			if err := tx.Where("tenant_id = ? AND reference = ?", tenantID, reference).First(&existing).Error; err != nil {
				return err
			}
			if existing.UserID == userID && existing.AmountMinor == amountMinor && existing.Currency == currency && existing.Quota == quota {
				*committed = existing
				replay = true
				return nil
			}
			return ErrTenantFundingConflict
		}
		if result := tx.Model(&User{}).Where("id = ? AND quota >= ?", owner.Id, quota).Update("quota", gorm.Expr("quota - ?", quota)); result.Error != nil || result.RowsAffected != 1 {
			if result.Error != nil {
				return result.Error
			}
			return ErrTenantFundingInsufficient
		}
		if result := tx.Model(&User{}).Where("id = ? AND quota <= ?", userID, common.MaxWalletQuota-quota).Update("quota", gorm.Expr("quota + ?", quota)); result.Error != nil || result.RowsAffected != 1 {
			if result.Error != nil {
				return result.Error
			}
			return ErrWalletQuotaLimitExceeded
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
	if err := cacheIncrUserQuota(committed.OwnerUserID, -int64(quota)); err != nil {
		common.SysError(common.LogText("failed to sync tenant owner quota cache for user %d: %s", committed.OwnerUserID, err))
	}
	if err := cacheIncrUserQuota(userID, int64(quota)); err != nil {
		common.SysError(common.LogText("failed to sync tenant funding quota for user %d: %s", userID, err))
	}
	return committed, false, nil
}

func GetTenantFundingWallet(tenantID int, currency string) (*TenantFundingWallet, error) {
	currency = normalizeFinancialCurrencyForFunding(currency)
	if tenantID <= 0 || currency == "" {
		return nil, ErrInvalidProviderFunding
	}
	var tenant Tenant
	if err := DB.Where("id = ? AND status = ? AND deleted_at IS NULL", tenantID, TenantActive).First(&tenant).Error; err != nil {
		return nil, ErrTenantFundingForbidden
	}
	var membership TenantMembership
	if err := DB.Where("tenant_id = ? AND role_key = ? AND status = ?", tenantID, TenantRoleOwner, MembershipActive).First(&membership).Error; err != nil {
		return nil, ErrTenantFundingForbidden
	}
	var user User
	if err := DB.Where("id = ? AND status = ?", membership.UserId, common.UserStatusEnabled).First(&user).Error; err != nil {
		return nil, ErrTenantFundingForbidden
	}
	availableQuota := max(user.Quota, 0)
	minor := decimal.NewFromInt(int64(availableQuota)).Mul(decimal.NewFromInt(100)).Div(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0).IntPart()
	return &TenantFundingWallet{OwnerUserID: user.Id, AvailableQuota: availableQuota, AvailableMinor: minor, Currency: currency}, nil
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
