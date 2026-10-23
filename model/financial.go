package model

import (
	"errors"
	"fmt"
	"sort"
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

// BeforeUpdate and BeforeDelete keep completed financial transactions
// append-only. Reversals must be recorded as new transactions.
func (t *FinancialTransaction) BeforeUpdate(_ *gorm.DB) error { return ErrFinancialLedgerImmutable }
func (t *FinancialTransaction) BeforeDelete(_ *gorm.DB) error { return ErrFinancialLedgerImmutable }

type LedgerAccount struct {
	ID              int64  `json:"id" gorm:"primaryKey"`
	Code            string `json:"code" gorm:"type:varchar(128);not null;uniqueIndex"`
	OwnerType       string `json:"owner_type" gorm:"type:varchar(32);not null;index"`
	OwnerID         int64  `json:"owner_id" gorm:"not null;index"`
	WalletAccountID int64  `json:"wallet_account_id" gorm:"index"`
	Currency        string `json:"currency" gorm:"type:varchar(16);not null;index"`
	Status          string `json:"status" gorm:"type:varchar(16);not null;index"`
	CreatedAt       int64  `json:"created_at" gorm:"not null;index"`
}

func (a *LedgerAccount) BeforeCreate(_ *gorm.DB) error {
	if a.CreatedAt == 0 {
		a.CreatedAt = time.Now().Unix()
	}
	if a.Status == "" {
		a.Status = WalletAccountActive
	}
	return nil
}

// LedgerEntry is an immutable debit or credit line. A transaction must balance
// before it can be marked completed.
type LedgerEntry struct {
	ID                     int64  `json:"id" gorm:"primaryKey"`
	FinancialTransactionID int64  `json:"financial_transaction_id" gorm:"not null;index"`
	LedgerAccountID        int64  `json:"ledger_account_id" gorm:"index"`
	WalletAccountID        int64  `json:"wallet_account_id" gorm:"not null;index"`
	EntryType              string `json:"entry_type" gorm:"type:varchar(8);not null"`
	AmountMinor            int64  `json:"amount_minor" gorm:"not null"`
	Currency               string `json:"currency" gorm:"type:varchar(16);not null"`
	CreatedAt              int64  `json:"created_at" gorm:"not null;index"`
}

func (e *LedgerEntry) BeforeUpdate(_ *gorm.DB) error { return ErrFinancialLedgerImmutable }
func (e *LedgerEntry) BeforeDelete(_ *gorm.DB) error { return ErrFinancialLedgerImmutable }

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
	ErrFinancialLedgerImmutable    = errors.New("financial ledger is immutable")
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

func GetOrCreateLedgerAccount(code string, ownerType string, ownerID int64, walletAccountID int64, currency string) (*LedgerAccount, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrFinancialInvalidOwner
	}
	currency, err := normalizeFinancialCurrency(currency)
	if err != nil {
		return nil, err
	}
	account := &LedgerAccount{Code: strings.TrimSpace(code), OwnerType: ownerType, OwnerID: ownerID, WalletAccountID: walletAccountID, Currency: currency, Status: WalletAccountActive}
	if err := DB.Where("code = ?", account.Code).First(account).Error; err == nil {
		return account, nil
	}
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(account).Error; err != nil {
		return nil, err
	}
	if err := DB.Where("code = ?", account.Code).First(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

type AppendLedgerTransactionRequest struct {
	Transaction FinancialTransaction
	Entries     []LedgerEntry
}

// AppendLedgerTransaction records one balanced transaction and applies the
// net effect of wallet entries atomically. Ledger entries have no update or
// delete path; corrections must be represented by a new transaction.
func AppendLedgerTransaction(tx *gorm.DB, request AppendLedgerTransactionRequest) (*FinancialTransaction, error) {
	transaction := request.Transaction
	if len(request.Entries) < 2 || transaction.AmountMinor <= 0 || strings.TrimSpace(transaction.TransactionNo) == "" || strings.TrimSpace(transaction.Type) == "" {
		return nil, ErrFinancialUnbalanced
	}
	if err := ValidateFinancialMinorAmount(transaction.AmountMinor); err != nil {
		return nil, err
	}
	currency, err := normalizeFinancialCurrency(transaction.Currency)
	if err != nil {
		return nil, err
	}
	transaction.Currency = currency
	var debit, credit int64
	for i := range request.Entries {
		entry := &request.Entries[i]
		if err := ValidateFinancialMinorAmount(entry.AmountMinor); err != nil {
			return nil, err
		}
		entryCurrency, err := normalizeFinancialCurrency(entry.Currency)
		if err != nil || entryCurrency != currency || entry.LedgerAccountID <= 0 {
			return nil, ErrFinancialUnbalanced
		}
		switch entry.EntryType {
		case LedgerEntryDebit:
			if debit > FinancialMaxMinorUnits-entry.AmountMinor {
				return nil, ErrFinancialAmountOverflow
			}
			debit += entry.AmountMinor
		case LedgerEntryCredit:
			if credit > FinancialMaxMinorUnits-entry.AmountMinor {
				return nil, ErrFinancialAmountOverflow
			}
			credit += entry.AmountMinor
		default:
			return nil, ErrFinancialUnbalanced
		}
	}
	if debit != credit || debit != transaction.AmountMinor {
		return nil, ErrFinancialUnbalanced
	}
	if transaction.CreatedAt == 0 {
		transaction.CreatedAt = time.Now().Unix()
	}
	transaction.Status = FinancialTransactionCompleted

	db := DB
	if tx != nil {
		db = tx
	}
	var result *FinancialTransaction
	err = db.Transaction(func(inner *gorm.DB) error {
		walletDeltas := make(map[int64]int64)
		for i := range request.Entries {
			entry := &request.Entries[i]
			account := &LedgerAccount{}
			if err := inner.Where("id = ?", entry.LedgerAccountID).First(account).Error; err != nil {
				return err
			}
			if account.Status != WalletAccountActive || account.Currency != currency {
				return ErrFinancialInvalidCurrency
			}
			if entry.WalletAccountID > 0 {
				wallet := &WalletAccount{}
				if err := inner.Where("id = ?", entry.WalletAccountID).First(wallet).Error; err != nil {
					return err
				}
				if wallet.Currency != currency || account.WalletAccountID != wallet.ID {
					return ErrFinancialUnbalanced
				}
				delta := entry.AmountMinor
				if entry.EntryType == LedgerEntryDebit {
					delta = -delta
				}
				walletDeltas[wallet.ID] += delta
			}
		}
		walletIDs := make([]int64, 0, len(walletDeltas))
		for walletID := range walletDeltas {
			walletIDs = append(walletIDs, walletID)
		}
		sort.Slice(walletIDs, func(i, j int) bool { return walletIDs[i] < walletIDs[j] })
		for _, walletID := range walletIDs {
			wallet := &WalletAccount{}
			if err := lockForUpdate(inner).First(wallet, walletID).Error; err != nil {
				return err
			}
			delta := walletDeltas[walletID]
			if wallet.Status != WalletAccountActive {
				return ErrWalletAccountClosed
			}
			if delta < 0 && wallet.AvailableMinor < -delta {
				return ErrWalletInsufficientAvailable
			}
			if delta > 0 && wallet.AvailableMinor > FinancialMaxMinorUnits-delta {
				return ErrFinancialAmountOverflow
			}
			if err := inner.Model(wallet).Updates(map[string]any{
				"available_minor": gorm.Expr("available_minor + ?", delta),
				"version":         gorm.Expr("version + 1"),
				"updated_at":      time.Now().Unix(),
			}).Error; err != nil {
				return err
			}
		}
		if err := inner.Create(&transaction).Error; err != nil {
			return err
		}
		for i := range request.Entries {
			entry := &request.Entries[i]
			entry.FinancialTransactionID = transaction.ID
			entry.Currency = currency
			if entry.CreatedAt == 0 {
				entry.CreatedAt = transaction.CreatedAt
			}
			if err := inner.Create(entry).Error; err != nil {
				return err
			}
		}
		result = &transaction
		return nil
	})
	return result, err
}

// CreateFinancialTransaction is retained for callers migrating to the append
// API. Entries must now identify distinct ledger accounts.
func CreateFinancialTransaction(tx *gorm.DB, transaction *FinancialTransaction, entries []LedgerEntry) error {
	if transaction == nil {
		return ErrFinancialUnbalanced
	}
	result, err := AppendLedgerTransaction(tx, AppendLedgerTransactionRequest{Transaction: *transaction, Entries: entries})
	if err == nil {
		*transaction = *result
	}
	return err
}

func migrateFinancialLedgerAccounts() error {
	var wallets []WalletAccount
	if err := DB.Find(&wallets).Error; err != nil {
		return err
	}
	for i := range wallets {
		wallet := &wallets[i]
		code := fmt.Sprintf("wallet:%s:%d:%s", wallet.OwnerType, wallet.OwnerID, wallet.Currency)
		account, err := GetOrCreateLedgerAccount(code, wallet.OwnerType, wallet.OwnerID, wallet.ID, wallet.Currency)
		if err != nil {
			return err
		}
		if err := DB.Model(&LedgerEntry{}).Where("wallet_account_id = ? AND ledger_account_id = 0", wallet.ID).UpdateColumn("ledger_account_id", account.ID).Error; err != nil {
			return err
		}
	}
	return nil
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
