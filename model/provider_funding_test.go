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
	flows, total, err := ListProviderCashFlows(1, 20)
	require.NoError(t, err)
	assert.True(t, DB.Migrator().HasTable(&ProviderCashFlow{}))
	assert.False(t, DB.Migrator().HasTable(&ProviderFunding{}))
	assert.False(t, DB.Migrator().HasTable(&ProviderFundingAllocation{}))
	assert.Equal(t, int64(1), total)
	require.Len(t, flows, 1)
	assert.Equal(t, ProviderCashFlowDirectionIncoming, flows[0].Direction)
	assert.Equal(t, ProviderCashFlowTypeFunding, flows[0].Type)
}

func TestListProviderFundingAllocationHistoryIncludesCounterpartyName(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&ProviderFundingAllocation{}, &User{}))
	user := &User{Username: "cashflow-counterparty", DisplayName: "Cash Flow Recipient", Password: "test-password"}
	require.NoError(t, DB.Create(user).Error)
	allocation := &ProviderFundingAllocation{
		UserID: user.Id, Reference: "cashflow-counterparty-reference", AmountMinor: 1250,
		Currency: "USD", Quota: 100, ExchangeRate: "1",
		CreatedAt: 1, CreatedBy: "admin",
	}
	require.NoError(t, DB.Create(allocation).Error)
	t.Cleanup(func() {
		DB.Delete(&ProviderFundingAllocation{}, allocation.ID)
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
