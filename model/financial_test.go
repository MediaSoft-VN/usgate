package model

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func migrateFinancialTestTables(t *testing.T) {
	t.Helper()
	models := []any{&WalletAccount{}, &LedgerAccount{}, &FinancialTransaction{}, &LedgerEntry{}, &IdempotencyKey{}, &ExchangeRate{}}
	require.NoError(t, DB.AutoMigrate(models...))
	// Startup migration must be idempotent when the process is restarted.
	require.NoError(t, DB.AutoMigrate(models...))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM ledger_entries")
		DB.Exec("DELETE FROM financial_transactions")
		DB.Exec("DELETE FROM ledger_accounts")
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
	walletAccount, err := GetOrCreateLedgerAccount("wallet:user:9002:VND", FinancialOwnerUser, 9002, wallet.ID, "VND")
	require.NoError(t, err)
	cashAccount, err := GetOrCreateLedgerAccount("system:cash:VND", "system", 0, 0, "VND")
	require.NoError(t, err)

	balanced := &FinancialTransaction{
		TransactionNo: "fin-txn-balanced", WalletAccountID: wallet.ID,
		Type: FinancialTransactionTopUp, Status: FinancialTransactionCompleted,
		AmountMinor: 5000, Currency: "VND",
	}
	entries := []LedgerEntry{
		{EntryType: LedgerEntryDebit, AmountMinor: 5000, Currency: "VND", LedgerAccountID: cashAccount.ID},
		{EntryType: LedgerEntryCredit, AmountMinor: 5000, Currency: "VND", LedgerAccountID: walletAccount.ID, WalletAccountID: wallet.ID},
	}
	require.NoError(t, CreateFinancialTransaction(nil, balanced, entries))

	unbalanced := &FinancialTransaction{
		TransactionNo: "fin-txn-unbalanced", WalletAccountID: wallet.ID,
		Type: FinancialTransactionTopUp, AmountMinor: 5000, Currency: "VND",
	}
	badEntries := []LedgerEntry{
		{EntryType: LedgerEntryDebit, AmountMinor: 4000, Currency: "VND", LedgerAccountID: cashAccount.ID},
		{EntryType: LedgerEntryCredit, AmountMinor: 5000, Currency: "VND", LedgerAccountID: walletAccount.ID, WalletAccountID: wallet.ID},
	}
	assert.ErrorIs(t, CreateFinancialTransaction(nil, unbalanced, badEntries), ErrFinancialUnbalanced)

	duplicate := &FinancialTransaction{
		TransactionNo: "fin-txn-balanced", WalletAccountID: wallet.ID,
		Type: FinancialTransactionTopUp, AmountMinor: 5000, Currency: "VND",
	}
	assert.Error(t, CreateFinancialTransaction(nil, duplicate, entries))

	entry := &LedgerEntry{}
	require.NoError(t, DB.First(entry, "financial_transaction_id = ?", balanced.ID).Error)
	assert.ErrorIs(t, DB.Model(entry).Update("amount_minor", 1).Error, ErrFinancialLedgerImmutable)
	assert.ErrorIs(t, DB.Delete(entry).Error, ErrFinancialLedgerImmutable)
}

func TestMigrateFinancialLedgerAccountsUpdatesLegacyEntries(t *testing.T) {
	migrateFinancialTestTables(t)

	wallet, err := GetOrCreateWalletAccount(FinancialOwnerUser, 9004, "VND")
	require.NoError(t, err)
	legacyEntry := &LedgerEntry{
		WalletAccountID: wallet.ID,
		EntryType:       LedgerEntryCredit,
		AmountMinor:     100,
		Currency:        "VND",
		CreatedAt:       1,
	}
	require.NoError(t, DB.Create(legacyEntry).Error)

	require.NoError(t, migrateFinancialLedgerAccounts())

	var migrated LedgerEntry
	require.NoError(t, DB.First(&migrated, legacyEntry.ID).Error)
	assert.NotZero(t, migrated.LedgerAccountID)
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

func TestWalletAccountMutationsMaintainAvailableAndReservedBalances(t *testing.T) {
	migrateFinancialTestTables(t)

	transaction, replay, err := CreditWalletAccount(FinancialOwnerUser, 9101, "USD", 1000, "wallet-test", "credit-1", "hash-credit-1", "")
	require.NoError(t, err)
	require.False(t, replay)
	assert.Equal(t, int64(1000), transaction.AvailableDeltaMinor)

	transaction, replay, err = ReserveWalletAccount(FinancialOwnerUser, 9101, "USD", 400, "wallet-test", "reserve-1", "hash-reserve-1", "")
	require.NoError(t, err)
	require.False(t, replay)
	assert.Equal(t, int64(-400), transaction.AvailableDeltaMinor)
	assert.Equal(t, int64(400), transaction.ReservedDeltaMinor)

	_, _, err = DebitWalletAccount(FinancialOwnerUser, 9101, "USD", 601, "wallet-test", "debit-1", "hash-debit-1", "")
	assert.ErrorIs(t, err, ErrWalletInsufficientAvailable)

	_, _, err = ReleaseWalletAccount(FinancialOwnerUser, 9101, "USD", 400, "wallet-test", "release-1", "hash-release-1", "")
	require.NoError(t, err)

	wallet, err := GetOrCreateWalletAccount(FinancialOwnerUser, 9101, "USD")
	require.NoError(t, err)
	assert.Equal(t, int64(1000), wallet.AvailableMinor)
	assert.Equal(t, int64(0), wallet.ReservedMinor)
}

func TestWalletAccountMutationsAreIdempotentAndRejectClosedWallet(t *testing.T) {
	migrateFinancialTestTables(t)

	first, replay, err := CreditWalletAccount(FinancialOwnerProvider, 9102, "USD", 500, "provider-wallet", "credit-1", "same", "")
	require.NoError(t, err)
	require.False(t, replay)
	second, replay, err := CreditWalletAccount(FinancialOwnerProvider, 9102, "USD", 500, "provider-wallet", "credit-1", "same", "")
	require.NoError(t, err)
	require.True(t, replay)
	assert.Equal(t, first.ID, second.ID)

	_, _, err = CreditWalletAccount(FinancialOwnerProvider, 9102, "USD", 600, "provider-wallet", "credit-1", "different", "")
	assert.ErrorIs(t, err, ErrIdempotencyConflict)

	wallet, err := GetOrCreateWalletAccount(FinancialOwnerProvider, 9102, "USD")
	require.NoError(t, err)
	require.NoError(t, DB.Model(wallet).Update("status", WalletAccountClosed).Error)
	_, _, err = DebitWalletAccount(FinancialOwnerProvider, 9102, "USD", 1, "provider-wallet", "debit-1", "debit", "")
	assert.ErrorIs(t, err, ErrWalletAccountClosed)
}

func TestWalletAccountConcurrentDebitsCannotOverspend(t *testing.T) {
	migrateFinancialTestTables(t)
	_, _, err := CreditWalletAccount(FinancialOwnerUser, 9103, "USD", 100, "concurrency", "credit-1", "credit", "")
	require.NoError(t, err)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	for i := range 10 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			key := "debit-" + strconv.Itoa(index)
			_, _, operationErr := DebitWalletAccount(FinancialOwnerUser, 9103, "USD", 20, "concurrency", key, "hash-"+key, "")
			if operationErr == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	wallet, err := GetOrCreateWalletAccount(FinancialOwnerUser, 9103, "USD")
	require.NoError(t, err)
	assert.Equal(t, 5, successes)
	assert.Equal(t, int64(0), wallet.AvailableMinor)
}
