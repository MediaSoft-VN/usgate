package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFinancialTransactionsFiltersAndPaginates(t *testing.T) {
	models := []any{&WalletAccount{}, &FinancialTransaction{}}
	require.NoError(t, DB.AutoMigrate(models...))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM financial_transactions")
		DB.Exec("DELETE FROM wallet_accounts")
	})

	wallet, err := GetOrCreateWalletAccount(FinancialOwnerProvider, 7001, "USD")
	require.NoError(t, err)
	for index, transactionType := range []string{FinancialTransactionTopUp, FinancialTransactionDebit} {
		require.NoError(t, DB.Create(&FinancialTransaction{
			TransactionNo:   "list-txn-" + string(rune('a'+index)),
			WalletAccountID: wallet.ID,
			Type:            transactionType,
			Status:          FinancialTransactionCompleted,
			AmountMinor:     int64(100 + index),
			Currency:        "USD",
			CreatedAt:       int64(index + 1),
		}).Error)
	}

	transactions, total, err := ListFinancialTransactions(FinancialTransactionFilter{
		OwnerType: FinancialOwnerProvider, Type: FinancialTransactionDebit, Page: 1, PageSize: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, transactions, 1)
	assert.Equal(t, FinancialTransactionDebit, transactions[0].Type)
}
