package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SaveExchangeRate(rate *ExchangeRate) error {
	if rate == nil {
		return ErrFinancialInvalidAmount
	}
	if err := ValidateExchangeRateValue(rate.Rate); err != nil {
		return err
	}
	return DB.Where("base_currency = ? AND quote_currency = ? AND source = ? AND rate_type = ? AND effective_date = ?", rate.BaseCurrency, rate.QuoteCurrency, rate.Source, rate.RateType, rate.EffectiveDate).
		Assign(rate).FirstOrCreate(rate).Error
}

func GetLatestExchangeRate(baseCurrency string, quoteCurrency string, source string, rateType string) *ExchangeRate {
	var rate ExchangeRate
	if err := DB.Where("base_currency = ? AND quote_currency = ? AND source = ? AND rate_type = ?", baseCurrency, quoteCurrency, source, rateType).
		Order("effective_date DESC").First(&rate).Error; err != nil {
		return nil
	}
	return &rate
}

// ListLatestExchangeRates returns the newest snapshot per quote currency for
// one base/source/type triple. The table only holds one row per day, so the
// result stays small enough to dedupe in Go for all three dialects.
func ListLatestExchangeRates(baseCurrency string, source string, rateType string) []ExchangeRate {
	var rows []ExchangeRate
	if err := DB.Where("base_currency = ? AND source = ? AND rate_type = ?", baseCurrency, source, rateType).
		Order("quote_currency ASC, effective_date DESC").Find(&rows).Error; err != nil {
		return nil
	}
	seen := make(map[string]bool, len(rows))
	out := make([]ExchangeRate, 0, len(rows))
	for _, row := range rows {
		if seen[row.QuoteCurrency] {
			continue
		}
		seen[row.QuoteCurrency] = true
		out = append(out, row)
	}
	return out
}

const (
	FinancialOwnerUser         = "user"
	FinancialOwnerOrganization = "organization"
	FinancialOwnerProvider     = "provider"
)

const (
	WalletAccountActive = "active"
	WalletAccountLocked = "locked"
	WalletAccountClosed = "closed"
)

const (
	FinancialTransactionPending   = "pending"
	FinancialTransactionCompleted = "completed"
	FinancialTransactionFailed    = "failed"
	FinancialTransactionReversed  = "reversed"
)

const (
	FinancialTransactionTopUp      = "topup"
	FinancialTransactionDebit      = "debit"
	FinancialTransactionCredit     = "credit"
	FinancialTransactionReserve    = "reserve"
	FinancialTransactionRelease    = "release"
	FinancialTransactionRefund     = "refund"
	FinancialTransactionAdjustment = "adjustment"
	FinancialTransactionTransfer   = "transfer"
)

const (
	LedgerEntryDebit  = "debit"
	LedgerEntryCredit = "credit"
)

const (
	IdempotencyProcessing = "processing"
	IdempotencyCompleted  = "completed"
	IdempotencyFailed     = "failed"
)

// WalletAccount stores a financial balance separately from the legacy quota
// balance. Amounts are integer minor units in Currency and must never be
// represented as floating-point values.
type WalletAccount struct {
	ID             int64  `json:"id" gorm:"primaryKey"`
	OwnerType      string `json:"owner_type" gorm:"type:varchar(32);not null;uniqueIndex:idx_wallet_owner_currency,priority:1"`
	OwnerID        int64  `json:"owner_id" gorm:"not null;uniqueIndex:idx_wallet_owner_currency,priority:2"`
	Currency       string `json:"currency" gorm:"type:varchar(16);not null;uniqueIndex:idx_wallet_owner_currency,priority:3"`
	AvailableMinor int64  `json:"available_minor" gorm:"not null;default:0"`
	ReservedMinor  int64  `json:"reserved_minor" gorm:"not null;default:0"`
	Status         string `json:"status" gorm:"type:varchar(16);not null;index"`
	Version        int64  `json:"version" gorm:"not null;default:0"`
	CreatedAt      int64  `json:"created_at" gorm:"not null;index"`
	UpdatedAt      int64  `json:"updated_at" gorm:"not null;index"`
}

func (w *WalletAccount) BeforeCreate(_ *gorm.DB) error {
	now := time.Now().Unix()
	if w.CreatedAt == 0 {
		w.CreatedAt = now
	}
	if w.UpdatedAt == 0 {
		w.UpdatedAt = now
	}
	if w.Status == "" {
		w.Status = WalletAccountActive
	}
	return nil
}

// FinancialTransaction represents one business operation. Its ledger entries
// are immutable accounting facts; status changes belong to the transaction.
type FinancialTransaction struct {
	ID                  int64  `json:"id" gorm:"primaryKey"`
	TransactionNo       string `json:"transaction_no" gorm:"type:varchar(128);not null;uniqueIndex"`
	WalletAccountID     int64  `json:"wallet_account_id" gorm:"not null;index"`
	IdempotencyKey      string `json:"idempotency_key" gorm:"type:varchar(255);index"`
	Type                string `json:"type" gorm:"type:varchar(32);not null;index"`
	Status              string `json:"status" gorm:"type:varchar(16);not null;index"`
	AmountMinor         int64  `json:"amount_minor" gorm:"not null"`
	AvailableDeltaMinor int64  `json:"available_delta_minor" gorm:"not null;default:0"`
	ReservedDeltaMinor  int64  `json:"reserved_delta_minor" gorm:"not null;default:0"`
	Currency            string `json:"currency" gorm:"type:varchar(16);not null"`
	Provider            string `json:"provider" gorm:"type:varchar(64);index"`
	ExternalReference   string `json:"external_reference" gorm:"type:varchar(255);index"`
	Metadata            string `json:"metadata" gorm:"type:text"`
	CreatedAt           int64  `json:"created_at" gorm:"not null;index"`
	CompletedAt         int64  `json:"completed_at"`
}

// LedgerEntry is an immutable debit or credit line. A transaction must balance
// before it can be marked completed.
type LedgerEntry struct {
	ID                     int64  `json:"id" gorm:"primaryKey"`
	FinancialTransactionID int64  `json:"financial_transaction_id" gorm:"not null;index"`
	WalletAccountID        int64  `json:"wallet_account_id" gorm:"not null;index"`
	EntryType              string `json:"entry_type" gorm:"type:varchar(8);not null"`
	AmountMinor            int64  `json:"amount_minor" gorm:"not null"`
	Currency               string `json:"currency" gorm:"type:varchar(16);not null"`
	CreatedAt              int64  `json:"created_at" gorm:"not null;index"`
}

// IdempotencyKey records a request boundary independently from the business
// transaction so webhook retries can safely return the original result.
type IdempotencyKey struct {
	ID           int64  `json:"id" gorm:"primaryKey"`
	Scope        string `json:"scope" gorm:"type:varchar(32);not null;uniqueIndex:idx_idempotency_scope_key,priority:1"`
	Key          string `json:"key" gorm:"type:varchar(128);not null;uniqueIndex:idx_idempotency_scope_key,priority:2"`
	RequestHash  string `json:"request_hash" gorm:"type:varchar(128);not null"`
	Status       string `json:"status" gorm:"type:varchar(16);not null;index"`
	ResourceType string `json:"resource_type" gorm:"type:varchar(64)"`
	ResourceID   string `json:"resource_id" gorm:"type:varchar(128)"`
	ResponseBody string `json:"response_body" gorm:"type:text"`
	ExpiresAt    int64  `json:"expires_at" gorm:"index"`
	CreatedAt    int64  `json:"created_at" gorm:"not null;index"`
	CompletedAt  int64  `json:"completed_at"`
}

const (
	ExchangeRateSourceVietcombank = "vietcombank"
	ExchangeRateSourceManual      = "manual"
	ExchangeRateTypeCash          = "cash"
	ExchangeRateTypeTransfer      = "transfer"
	ExchangeRateTypeSell          = "sell"
)

// ExchangeRate keeps the rate snapshot used by a payment. Rate is stored as a
// decimal string so precision is preserved consistently across all dialects.
type ExchangeRate struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	BaseCurrency  string `json:"base_currency" gorm:"type:varchar(16);not null;uniqueIndex:idx_exchange_rate_snapshot,priority:1"`
	QuoteCurrency string `json:"quote_currency" gorm:"type:varchar(16);not null;uniqueIndex:idx_exchange_rate_snapshot,priority:2"`
	Rate          string `json:"rate" gorm:"type:varchar(64);not null"`
	Source        string `json:"source" gorm:"type:varchar(32);not null;uniqueIndex:idx_exchange_rate_snapshot,priority:3"`
	RateType      string `json:"rate_type" gorm:"type:varchar(16);not null;uniqueIndex:idx_exchange_rate_snapshot,priority:4"`
	EffectiveDate string `json:"effective_date" gorm:"type:varchar(10);not null;uniqueIndex:idx_exchange_rate_snapshot,priority:5"`
	FetchedAt     int64  `json:"fetched_at" gorm:"not null;index"`
	Metadata      string `json:"metadata" gorm:"type:text"`
}

var (
	ErrFinancialInvalidOwner       = errors.New("invalid wallet owner")
	ErrFinancialInvalidCurrency    = errors.New("invalid currency")
	ErrFinancialInvalidAmount      = errors.New("amount must be positive")
	ErrWalletAccountLocked         = errors.New("wallet account is locked")
	ErrWalletAccountClosed         = errors.New("wallet account is closed")
	ErrWalletInsufficientAvailable = errors.New("wallet has insufficient available balance")
	ErrWalletInsufficientReserved  = errors.New("wallet has insufficient reserved balance")
	ErrIdempotencyInProgress       = errors.New("idempotency request is in progress")
	ErrFinancialUnbalanced         = errors.New("ledger entries do not balance")
	ErrIdempotencyConflict         = errors.New("idempotency key payload mismatch")
)

var financialTransactionSequence atomic.Uint64

func normalizeFinancialCurrency(currency string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if len(code) < 2 || len(code) > 16 {
		return "", ErrFinancialInvalidCurrency
	}
	for i := range len(code) {
		if c := code[i]; c < 'A' || c > 'Z' {
			return "", ErrFinancialInvalidCurrency
		}
	}
	return code, nil
}

func normalizeFinancialOwner(ownerType string, ownerID int64) (string, error) {
	ownerType = strings.ToLower(strings.TrimSpace(ownerType))
	if ownerID <= 0 {
		return "", ErrFinancialInvalidOwner
	}
	switch ownerType {
	case FinancialOwnerUser, FinancialOwnerOrganization, FinancialOwnerProvider:
		return ownerType, nil
	default:
		return "", ErrFinancialInvalidOwner
	}
}

// GetOrCreateWalletAccount returns the wallet for an owner/currency pair,
// creating an active empty wallet when none exists.
func GetOrCreateWalletAccount(ownerType string, ownerID int64, currency string) (*WalletAccount, error) {
	ownerType, err := normalizeFinancialOwner(ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	currency, err = normalizeFinancialCurrency(currency)
	if err != nil {
		return nil, err
	}
	wallet := &WalletAccount{OwnerType: ownerType, OwnerID: ownerID, Currency: currency}
	if err := DB.Where("owner_type = ? AND owner_id = ? AND currency = ?", ownerType, ownerID, currency).First(wallet).Error; err == nil {
		return wallet, nil
	}
	wallet.Status = WalletAccountActive
	if err := DB.Create(wallet).Error; err != nil {
		existing := &WalletAccount{}
		if lookupErr := DB.Where("owner_type = ? AND owner_id = ? AND currency = ?", ownerType, ownerID, currency).First(existing).Error; lookupErr == nil {
			return existing, nil
		}
		return nil, err
	}
	return wallet, nil
}

// ListWalletAccounts returns financial wallets for an owner type. It is used by
// administrative views and never falls back to legacy quota balances.
func ListWalletAccounts(ownerType string, ownerID int64) ([]WalletAccount, error) {
	if ownerID < 0 {
		return nil, ErrFinancialInvalidOwner
	}
	validationID := ownerID
	if validationID == 0 {
		validationID = 1
	}
	ownerType, err := normalizeFinancialOwner(ownerType, validationID)
	if err != nil {
		return nil, err
	}
	query := DB.Where("owner_type = ?", ownerType)
	if ownerID > 0 {
		query = query.Where("owner_id = ?", ownerID)
	}
	var wallets []WalletAccount
	if err := query.Order("owner_id ASC, currency ASC").Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

// AvailableDeltaMinor and ReservedDeltaMinor record the balance movement made
// by a wallet operation. They are kept on the transaction so the balance
// change can be audited without reconstructing it from mutable account state.
// The ledger entry model remains reserved for the double-entry phase.

// mutateWalletAccount applies one atomic wallet operation. The idempotency row,
// balance update, and transaction record commit or roll back together.
func mutateWalletAccount(ownerType string, ownerID int64, currency string, amountMinor int64, operation string, availableDelta int64, reservedDelta int64, scope string, idempotencyKey string, requestHash string, metadata string) (*FinancialTransaction, bool, error) {
	ownerType, err := normalizeFinancialOwner(ownerType, ownerID)
	if err != nil {
		return nil, false, err
	}
	currency, err = normalizeFinancialCurrency(currency)
	if err != nil {
		return nil, false, err
	}
	if err := ValidateFinancialMinorAmount(amountMinor); err != nil {
		return nil, false, err
	}
	scope = strings.TrimSpace(scope)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	requestHash = strings.TrimSpace(requestHash)
	if scope == "" || idempotencyKey == "" || requestHash == "" {
		return nil, false, ErrIdempotencyConflict
	}

	var result *FinancialTransaction
	replay := false
	err = DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix()
		claim := &IdempotencyKey{
			Scope: scope, Key: idempotencyKey, RequestHash: requestHash,
			Status: IdempotencyProcessing, CreatedAt: now,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(claim).Error; err != nil {
			return err
		}
		existing := &IdempotencyKey{}
		if err := tx.Where("scope = ? AND key = ?", scope, idempotencyKey).First(existing).Error; err != nil {
			return err
		}
		if existing.RequestHash != requestHash {
			return ErrIdempotencyConflict
		}
		if existing.Status == IdempotencyCompleted {
			transactionID, parseErr := strconv.ParseInt(existing.ResourceID, 10, 64)
			if parseErr != nil || transactionID <= 0 {
				return ErrIdempotencyInProgress
			}
			result = &FinancialTransaction{}
			if err := tx.First(result, transactionID).Error; err != nil {
				return err
			}
			replay = true
			return nil
		}
		if existing.Status == IdempotencyProcessing {
			if existing.ID != claim.ID {
				return ErrIdempotencyInProgress
			}
		} else {
			return ErrIdempotencyInProgress
		}

		wallet := &WalletAccount{}
		if err := lockForUpdate(tx).Where("owner_type = ? AND owner_id = ? AND currency = ?", ownerType, ownerID, currency).First(wallet).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				wallet = &WalletAccount{OwnerType: ownerType, OwnerID: ownerID, Currency: currency, Status: WalletAccountActive}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(wallet).Error; err != nil {
					return err
				}
				if err := lockForUpdate(tx).Where("owner_type = ? AND owner_id = ? AND currency = ?", ownerType, ownerID, currency).First(wallet).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		switch wallet.Status {
		case WalletAccountLocked:
			return ErrWalletAccountLocked
		case WalletAccountClosed:
			return ErrWalletAccountClosed
		case WalletAccountActive:
		default:
			return ErrWalletAccountClosed
		}
		if wallet.AvailableMinor > FinancialMaxMinorUnits-availableDelta && availableDelta > 0 {
			return ErrFinancialAmountOverflow
		}
		if wallet.ReservedMinor > FinancialMaxMinorUnits-reservedDelta && reservedDelta > 0 {
			return ErrFinancialAmountOverflow
		}
		if availableDelta < 0 && wallet.AvailableMinor < -availableDelta {
			return ErrWalletInsufficientAvailable
		}
		if reservedDelta < 0 && wallet.ReservedMinor < -reservedDelta {
			return ErrWalletInsufficientReserved
		}

		updated := tx.Model(&WalletAccount{}).
			Where("id = ? AND version = ? AND status = ?", wallet.ID, wallet.Version, WalletAccountActive).
			Updates(map[string]any{
				"available_minor": gorm.Expr("available_minor + ?", availableDelta),
				"reserved_minor":  gorm.Expr("reserved_minor + ?", reservedDelta),
				"version":         gorm.Expr("version + 1"),
				"updated_at":      now,
			})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrWalletInsufficientAvailable
		}
		result = &FinancialTransaction{
			TransactionNo:   fmt.Sprintf("wallet-%d-%d", now, financialTransactionSequence.Add(1)),
			WalletAccountID: wallet.ID, IdempotencyKey: idempotencyKey,
			Type: operation, Status: FinancialTransactionCompleted,
			AmountMinor: amountMinor, Currency: currency, Metadata: metadata,
			AvailableDeltaMinor: availableDelta, ReservedDeltaMinor: reservedDelta,
			CreatedAt: now, CompletedAt: now,
		}
		if err := tx.Create(result).Error; err != nil {
			return err
		}
		if err := tx.Model(existing).Updates(map[string]any{
			"status": IdempotencyCompleted, "resource_type": "financial_transaction",
			"resource_id": strconv.FormatInt(result.ID, 10), "completed_at": now,
		}).Error; err != nil {
			return err
		}
		return nil
	})
	return result, replay, err
}

func CreditWalletAccount(ownerType string, ownerID int64, currency string, amountMinor int64, scope string, idempotencyKey string, requestHash string, metadata string) (*FinancialTransaction, bool, error) {
	return mutateWalletAccount(ownerType, ownerID, currency, amountMinor, FinancialTransactionCredit, amountMinor, 0, scope, idempotencyKey, requestHash, metadata)
}

func DebitWalletAccount(ownerType string, ownerID int64, currency string, amountMinor int64, scope string, idempotencyKey string, requestHash string, metadata string) (*FinancialTransaction, bool, error) {
	return mutateWalletAccount(ownerType, ownerID, currency, amountMinor, FinancialTransactionDebit, -amountMinor, 0, scope, idempotencyKey, requestHash, metadata)
}

func ReserveWalletAccount(ownerType string, ownerID int64, currency string, amountMinor int64, scope string, idempotencyKey string, requestHash string, metadata string) (*FinancialTransaction, bool, error) {
	return mutateWalletAccount(ownerType, ownerID, currency, amountMinor, FinancialTransactionReserve, -amountMinor, amountMinor, scope, idempotencyKey, requestHash, metadata)
}

func ReleaseWalletAccount(ownerType string, ownerID int64, currency string, amountMinor int64, scope string, idempotencyKey string, requestHash string, metadata string) (*FinancialTransaction, bool, error) {
	return mutateWalletAccount(ownerType, ownerID, currency, amountMinor, FinancialTransactionRelease, amountMinor, -amountMinor, scope, idempotencyKey, requestHash, metadata)
}

// CreateFinancialTransaction persists a balanced double-entry transaction and
// applies its net effect to the wallet inside one database transaction.
func CreateFinancialTransaction(tx *gorm.DB, transaction *FinancialTransaction, entries []LedgerEntry) error {
	if transaction == nil || len(entries) == 0 {
		return ErrFinancialUnbalanced
	}
	if transaction.AmountMinor <= 0 {
		return ErrFinancialInvalidAmount
	}
	currency, err := normalizeFinancialCurrency(transaction.Currency)
	if err != nil {
		return err
	}
	transaction.Currency = currency
	if strings.TrimSpace(transaction.TransactionNo) == "" || strings.TrimSpace(transaction.Type) == "" {
		return ErrFinancialUnbalanced
	}
	var debit, credit int64
	for i := range entries {
		entry := &entries[i]
		if entry.AmountMinor <= 0 {
			return ErrFinancialInvalidAmount
		}
		entryCurrency, err := normalizeFinancialCurrency(entry.Currency)
		if err != nil {
			return err
		}
		if entryCurrency != currency || entry.WalletAccountID != transaction.WalletAccountID {
			return ErrFinancialUnbalanced
		}
		switch entry.EntryType {
		case LedgerEntryDebit:
			debit += entry.AmountMinor
		case LedgerEntryCredit:
			credit += entry.AmountMinor
		default:
			return ErrFinancialUnbalanced
		}
	}
	if debit != credit || debit != transaction.AmountMinor {
		return ErrFinancialUnbalanced
	}
	if transaction.CreatedAt == 0 {
		transaction.CreatedAt = time.Now().Unix()
	}
	if transaction.Status == "" {
		transaction.Status = FinancialTransactionPending
	}

	db := DB
	if tx != nil {
		db = tx
	}
	return db.Transaction(func(inner *gorm.DB) error {
		wallet := &WalletAccount{}
		if err := lockForUpdate(inner).Where("id = ?", transaction.WalletAccountID).First(wallet).Error; err != nil {
			return err
		}
		if wallet.Status != WalletAccountActive {
			return ErrFinancialInvalidOwner
		}
		if wallet.Currency != currency {
			return ErrFinancialInvalidCurrency
		}
		if err := inner.Create(transaction).Error; err != nil {
			return err
		}
		for i := range entries {
			entries[i].FinancialTransactionID = transaction.ID
			entries[i].WalletAccountID = transaction.WalletAccountID
			entries[i].Currency = currency
			if entries[i].CreatedAt == 0 {
				entries[i].CreatedAt = time.Now().Unix()
			}
			if err := inner.Create(&entries[i]).Error; err != nil {
				return err
			}
		}
		now := time.Now().Unix()
		updates := map[string]any{"available_minor": wallet.AvailableMinor + credit - debit, "updated_at": now, "version": wallet.Version + 1}
		if updates["available_minor"].(int64) < 0 {
			return ErrFinancialInvalidAmount
		}
		return inner.Model(&WalletAccount{}).Where("id = ? AND version = ?", wallet.ID, wallet.Version).Updates(updates).Error
	})
}

// ClaimIdempotencyKey reserves a scope/key pair. When the key already exists
// with the same request hash the original record is returned with replay=true
// so webhook retries reuse the first result; a different hash is a conflict.
func ClaimIdempotencyKey(scope string, key string, requestHash string, ttlSeconds int64) (record *IdempotencyKey, replay bool, err error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	key = strings.TrimSpace(key)
	if scope == "" || key == "" || requestHash == "" {
		return nil, false, ErrIdempotencyConflict
	}
	now := time.Now().Unix()
	record = &IdempotencyKey{Scope: scope, Key: key, RequestHash: requestHash, Status: IdempotencyProcessing, CreatedAt: now}
	if ttlSeconds > 0 {
		record.ExpiresAt = now + ttlSeconds
	}
	if err := DB.Create(record).Error; err == nil {
		return record, false, nil
	}
	existing := &IdempotencyKey{}
	if lookupErr := DB.Where("scope = ? AND key = ?", scope, key).First(existing).Error; lookupErr != nil {
		return nil, false, lookupErr
	}
	if existing.RequestHash != requestHash {
		return nil, false, ErrIdempotencyConflict
	}
	return existing, true, nil
}

// ValidateExchangeRateValue keeps stored rate snapshots comparable across
// SQLite, MySQL, and PostgreSQL by requiring a positive decimal string.
func ValidateExchangeRateValue(rate string) error {
	value, err := decimal.NewFromString(strings.TrimSpace(rate))
	if err != nil || value.LessThanOrEqual(decimal.Zero) {
		return ErrFinancialInvalidAmount
	}
	return nil
}
