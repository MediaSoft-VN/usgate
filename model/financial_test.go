package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func migrateFinancialTestTables(t *testing.T) {
	t.Helper()
	models := []any{&WalletAccount{}, &FinancialTransaction{}, &LedgerEntry{}, &IdempotencyKey{}, &ExchangeRate{}}
	require.NoError(t, DB.AutoMigrate(models...))
	// Startup migration must be idempotent when the process is restarted.
	require.NoError(t, DB.AutoMigrate(models...))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM ledger_entries")
		DB.Exec("DELETE FROM financial_transactions")
		DB.Exec("DELETE FROM wallet_accounts")
		DB.Exec("DELETE FROM idempotency_keys")
		DB.Exec("DELETE FROM exchange_rates")
	})
}

func TestGetOrCreateWalletAccountDedupesOwnerCurrency(t *testing.T) {
	migrateFinancialTestTables(t)

	first, err := GetOrCreateWalletAccount("user", 9001, "usd")
	require.NoError(t, err)
	assert.Equal(t, "USD", first.Currency)
	assert.Equal(t, WalletAccountActive, first.Status)

	second, err := GetOrCreateWalletAccount("USER", 9001, "USD")
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	_, err = GetOrCreateWalletAccount("user", 9001, "US!")
	assert.ErrorIs(t, err, ErrFinancialInvalidCurrency)

	_, err = GetOrCreateWalletAccount("unknown", 9001, "USD")
	assert.ErrorIs(t, err, ErrFinancialInvalidOwner)
}

func TestCreateFinancialTransactionRequiresBalancedEntries(t *testing.T) {
	migrateFinancialTestTables(t)

	wallet, err := GetOrCreateWalletAccount(FinancialOwnerUser, 9002, "VND")
	require.NoError(t, err)

	balanced := &FinancialTransaction{
		TransactionNo: "fin-txn-balanced", WalletAccountID: wallet.ID,
		Type: FinancialTransactionTopUp, Status: FinancialTransactionCompleted,
		AmountMinor: 5000, Currency: "VND",
	}
	entries := []LedgerEntry{
		{EntryType: LedgerEntryDebit, AmountMinor: 5000, Currency: "VND", WalletAccountID: wallet.ID},
		{EntryType: LedgerEntryCredit, AmountMinor: 5000, Currency: "VND", WalletAccountID: wallet.ID},
	}
	require.NoError(t, CreateFinancialTransaction(nil, balanced, entries))

	unbalanced := &FinancialTransaction{
		TransactionNo: "fin-txn-unbalanced", WalletAccountID: wallet.ID,
		Type: FinancialTransactionTopUp, AmountMinor: 5000, Currency: "VND",
	}
	badEntries := []LedgerEntry{
		{EntryType: LedgerEntryDebit, AmountMinor: 4000, Currency: "VND", WalletAccountID: wallet.ID},
		{EntryType: LedgerEntryCredit, AmountMinor: 5000, Currency: "VND", WalletAccountID: wallet.ID},
	}
	assert.ErrorIs(t, CreateFinancialTransaction(nil, unbalanced, badEntries), ErrFinancialUnbalanced)

	duplicate := &FinancialTransaction{
		TransactionNo: "fin-txn-balanced", WalletAccountID: wallet.ID,
		Type: FinancialTransactionTopUp, AmountMinor: 5000, Currency: "VND",
	}
	assert.Error(t, CreateFinancialTransaction(nil, duplicate, entries))
}

func TestClaimIdempotencyKeyReplaysSamePayload(t *testing.T) {
	migrateFinancialTestTables(t)

	first, replay, err := ClaimIdempotencyKey("sepay_webhook", "webhook-1", "hash-a", 3600)
	require.NoError(t, err)
	assert.False(t, replay)
	require.NotNil(t, first)

	second, replay, err := ClaimIdempotencyKey("sepay_webhook", "webhook-1", "hash-a", 3600)
	require.NoError(t, err)
	assert.True(t, replay)
	assert.Equal(t, first.ID, second.ID)

	_, _, err = ClaimIdempotencyKey("sepay_webhook", "webhook-1", "hash-b", 3600)
	assert.ErrorIs(t, err, ErrIdempotencyConflict)
}

func TestSaveExchangeRateRejectsNonPositiveRate(t *testing.T) {
	migrateFinancialTestTables(t)

	require.NoError(t, SaveExchangeRate(&ExchangeRate{
		BaseCurrency: "VND", QuoteCurrency: "USD", Rate: "26100",
		Source: ExchangeRateSourceVietcombank, RateType: ExchangeRateTypeSell,
		EffectiveDate: "2026-10-13", FetchedAt: 1,
	}))
	assert.ErrorIs(t, SaveExchangeRate(&ExchangeRate{
		BaseCurrency: "VND", QuoteCurrency: "USD", Rate: "0",
		Source: ExchangeRateSourceManual, RateType: ExchangeRateTypeSell,
		EffectiveDate: "2026-10-14", FetchedAt: 2,
	}), ErrFinancialInvalidAmount)

	latest := GetLatestExchangeRate("VND", "USD", ExchangeRateSourceVietcombank, ExchangeRateTypeSell)
	require.NotNil(t, latest)
	assert.Equal(t, "26100", latest.Rate)
}
