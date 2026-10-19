package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordProviderFundingIsIdempotentAndSummarized(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Channel{}, &ProviderFunding{}, &ProviderFundingAllocation{}, &User{}))
	channel := &Channel{Name: "provider funding test", Type: 1, Key: "provider-funding-test-key"}
	require.NoError(t, DB.Create(channel).Error)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM provider_fundings")
		DB.Exec("DELETE FROM provider_funding_allocations")
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
}
