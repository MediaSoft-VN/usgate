package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordProviderFundingIsIdempotentAndSummarized(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Channel{}, &ProviderFunding{}, &ProviderFundingAllocation{}, &ProviderCashFlow{}, &User{}))
	channel := &Channel{Name: "provider funding test", Type: 1, Key: "provider-funding-test-key"}
	require.NoError(t, DB.Create(channel).Error)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM provider_fundings")
		DB.Exec("DELETE FROM provider_funding_allocations")
		DB.Exec("DELETE FROM provider_cash_flows")
		DB.Exec("DELETE FROM channels WHERE id = ?", channel.Id)
	})

	funding := &ProviderFunding{ChannelID: channel.Id, Reference: "provider-deposit-1", AmountMinor: 12500, Currency: "usd", Source: ProviderFundingSourceManual, CreatedBy: "admin"}
	record, replay, err := RecordProviderFunding(funding)
	require.NoError(t, err)
	assert.False(t, replay)
	assert.Equal(t, "USD", record.Currency)
	assert.Equal(t, "admin", record.CreatedBy)

	replayed, replay, err := RecordProviderFunding(&ProviderFunding{ChannelID: channel.Id, Reference: "provider-deposit-1", AmountMinor: 12500, Currency: "USD", Source: ProviderFundingSourceManual})
	require.NoError(t, err)
	assert.True(t, replay)
	assert.Equal(t, record.ID, replayed.ID)

	summary, err := SummarizeProviderFundings(ProviderFundingFilter{})
	require.NoError(t, err)
	require.Len(t, summary, 1)
	assert.Equal(t, int64(12500), summary[0].AmountMinor)
	wallets, err := SummarizeProviderFundingWallets()
	require.NoError(t, err)
	require.Len(t, wallets, 1)
	assert.Zero(t, wallets[0].ChannelID)
	assert.Equal(t, int64(12500), wallets[0].AmountMinor)
	channelWallets, err := SummarizeProviderFundingByChannel()
	require.NoError(t, err)
	require.Len(t, channelWallets, 1)
	assert.Equal(t, channel.Id, channelWallets[0].ChannelID)
	assert.Equal(t, int64(12500), channelWallets[0].AmountMinor)
	flows, total, err := ListProviderCashFlows(ProviderFundingFilter{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.True(t, DB.Migrator().HasTable(&ProviderCashFlow{}))
	assert.False(t, DB.Migrator().HasTable(&ProviderFunding{}))
	assert.False(t, DB.Migrator().HasTable(&ProviderFundingAllocation{}))
	assert.Equal(t, int64(1), total)
	require.Len(t, flows, 1)
	assert.Equal(t, ProviderCashFlowDirectionIncoming, flows[0].Direction)
	assert.Equal(t, ProviderCashFlowTypeFunding, flows[0].Type)
	filtered, filteredTotal, err := ListProviderCashFlows(ProviderFundingFilter{
		Direction: ProviderCashFlowDirectionIncoming,
		Type:      ProviderCashFlowTypeFunding,
		Partner:   "provider funding test",
		From:      record.CreatedAt - 1,
		To:        record.CreatedAt + 1,
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), filteredTotal)
	assert.Len(t, filtered, 1)
}

func TestProviderFundingWalletUsesNetLedgerBalancePerCurrency(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Channel{}, &ProviderCashFlow{}))
	channel := &Channel{Name: "provider wallet ledger test", Type: 1, Key: "provider-wallet-ledger-test-key"}
	require.NoError(t, DB.Create(channel).Error)
	flows := []ProviderCashFlow{
		{ChannelID: channel.Id, Direction: ProviderCashFlowDirectionIncoming, Type: ProviderCashFlowTypeFunding, Reference: "wallet-ledger-in-1", AmountMinor: 10000, Currency: "usd", Source: ProviderFundingSourceManual},
		{ChannelID: channel.Id, Direction: ProviderCashFlowDirectionIncoming, Type: ProviderCashFlowTypeFunding, Reference: "wallet-ledger-in-2", AmountMinor: 5000, Currency: "USD", Source: ProviderFundingSourceManual},
		{UserID: 1, Direction: ProviderCashFlowDirectionOutgoing, Type: ProviderCashFlowTypeAllocation, Reference: "wallet-ledger-out-1", AmountMinor: 2500, Currency: "USD", Source: "allocation"},
		{ChannelID: channel.Id, Direction: ProviderCashFlowDirectionIncoming, Type: ProviderCashFlowTypeFunding, Reference: "wallet-ledger-eur", AmountMinor: 700, Currency: "eur", Source: ProviderFundingSourceManual},
	}
	for i := range flows {
		require.NoError(t, DB.Create(&flows[i]).Error)
	}
	t.Cleanup(func() {
		DB.Where("reference LIKE ?", "wallet-ledger-%").Delete(&ProviderCashFlow{})
		DB.Delete(&Channel{}, channel.Id)
	})

	wallets, err := SummarizeProviderFundingWallets()
	require.NoError(t, err)
	require.Len(t, wallets, 2)
	assert.Equal(t, "EUR", wallets[0].Currency)
	assert.Equal(t, int64(700), wallets[0].AmountMinor)
	assert.Equal(t, int64(0), wallets[0].AllocatedMinor)
	assert.Equal(t, int64(700), wallets[0].AvailableMinor)
	assert.Equal(t, "USD", wallets[1].Currency)
	assert.Equal(t, int64(12500), wallets[1].AmountMinor)
	assert.Equal(t, int64(2500), wallets[1].AllocatedMinor)
	assert.Equal(t, int64(12500), wallets[1].AvailableMinor)

	channelWallets, err := SummarizeProviderFundingByChannel()
	require.NoError(t, err)
	var usdChannelWallet *ProviderFundingChannelSummary
	for i := range channelWallets {
		if channelWallets[i].ChannelID == channel.Id && channelWallets[i].Currency == "USD" {
			usdChannelWallet = &channelWallets[i]
			break
		}
	}
	require.NotNil(t, usdChannelWallet)
	assert.Equal(t, int64(15000), usdChannelWallet.AmountMinor)

	// Deleting a channel must hide its per-channel wallet summary without
	// deleting the financial ledger entries or changing the aggregate pool.
	require.NoError(t, DB.Delete(channel).Error)
	channelWallets, err = SummarizeProviderFundingByChannel()
	require.NoError(t, err)
	assert.Empty(t, channelWallets)

	wallets, err = SummarizeProviderFundingWallets()
	require.NoError(t, err)
	require.Len(t, wallets, 2)
	var retainedPoolAmount int64
	for i := range wallets {
		if wallets[i].Currency == "USD" {
			retainedPoolAmount = wallets[i].AmountMinor
			break
		}
	}
	assert.Equal(t, int64(12500), retainedPoolAmount)
	var retainedFlows int64
	require.NoError(t, DB.Model(&ProviderCashFlow{}).
		Where("reference LIKE ?", "wallet-ledger-%").Count(&retainedFlows).Error)
	assert.Equal(t, int64(4), retainedFlows)
}

func TestSummarizeProviderCashFlowsAppliesDirectionAndDateFilters(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&ProviderCashFlow{}, &User{}))
	flows := []ProviderCashFlow{
		{Direction: ProviderCashFlowDirectionIncoming, Type: ProviderCashFlowTypeFunding, Reference: "report-in-1", AmountMinor: 10000, Currency: "USD", Source: ProviderFundingSourceManual, CreatedAt: 100},
		{Direction: ProviderCashFlowDirectionOutgoing, Type: ProviderCashFlowTypeAllocation, Reference: "report-out-1", AmountMinor: 2500, Currency: "USD", Source: "allocation", CreatedAt: 200},
		{Direction: ProviderCashFlowDirectionIncoming, Type: ProviderCashFlowTypeFunding, Reference: "report-in-2", AmountMinor: 500, Currency: "USD", Source: ProviderFundingSourceManual, CreatedAt: 300},
	}
	for i := range flows {
		require.NoError(t, DB.Create(&flows[i]).Error)
	}
	t.Cleanup(func() {
		DB.Where("reference LIKE ?", "report-%").Delete(&ProviderCashFlow{})
	})

	summary, err := SummarizeProviderCashFlows(ProviderFundingFilter{From: 100, To: 300})
	require.NoError(t, err)
	require.Len(t, summary, 1)
	assert.Equal(t, int64(10000), summary[0].IncomingMinor)
	assert.Equal(t, int64(2500), summary[0].OutgoingMinor)
	assert.Equal(t, int64(7500), summary[0].NetMinor)
	assert.Equal(t, int64(2), summary[0].Count)

	incoming, err := SummarizeProviderCashFlows(ProviderFundingFilter{Direction: ProviderCashFlowDirectionIncoming})
	require.NoError(t, err)
	require.Len(t, incoming, 1)
	assert.Equal(t, int64(10500), incoming[0].IncomingMinor)
	assert.Zero(t, incoming[0].OutgoingMinor)
}

func TestListProviderFundingAllocationHistoryIncludesCounterpartyName(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&ProviderCashFlow{}, &User{}))
	user := &User{Username: "cashflow-counterparty", DisplayName: "Cash Flow Recipient", Password: "test-password"}
	require.NoError(t, DB.Create(user).Error)
	allocation := &ProviderCashFlow{
		UserID: user.Id, Direction: ProviderCashFlowDirectionOutgoing, Type: ProviderCashFlowTypeAllocation, Source: "allocation", Reference: "cashflow-counterparty-reference", AmountMinor: 1250,
		Currency: "USD", Quota: 100, ExchangeRate: "1",
		CreatedAt: 1, CreatedBy: "admin",
	}
	require.NoError(t, DB.Create(allocation).Error)
	t.Cleanup(func() {
		DB.Delete(&ProviderCashFlow{}, allocation.ID)
		DB.Delete(&User{}, user.Id)
	})

	history, err := ListProviderFundingAllocationHistory()
	require.NoError(t, err)
	var found *ProviderFundingAllocationHistory
	for i := range history {
		if history[i].ID == allocation.ID {
			found = &history[i]
			break
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "Cash Flow Recipient (cashflow-counterparty)", found.CounterpartyName)
}
