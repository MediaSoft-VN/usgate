package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordProviderFundingIsIdempotentAndSummarized(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Channel{}, &ProviderPartner{}, &ProviderFunding{}))
	partner := &ProviderPartner{Code: "funding-test", Name: "Funding Test", APIKey: "test-api-key", APIEndpoint: "https://provider.example.test/usage"}
	require.NoError(t, CreateProviderPartner(partner))
	duplicate := &ProviderPartner{Code: "funding-test-duplicate", Name: "Funding Test", APIKey: "test-api-key", APIEndpoint: "https://provider.example.test/usage"}
	require.ErrorIs(t, CreateProviderPartner(duplicate), ErrProviderPartnerExists)
	unassigned := &ProviderPartner{Code: "unassigned-test", Name: "Unassigned Test", APIKey: "test-api-key", APIEndpoint: "https://provider.example.test/usage"}
	require.NoError(t, CreateProviderPartner(unassigned))
	channel := &Channel{Name: "provider funding test", Type: 1, Key: "provider-funding-test-key", ProviderPartnerID: &partner.ID}
	require.NoError(t, DB.Create(channel).Error)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM provider_fundings")
		DB.Exec("DELETE FROM channels WHERE id = ?", channel.Id)
		DB.Exec("DELETE FROM provider_partners WHERE id = ?", partner.ID)
		DB.Exec("DELETE FROM provider_partners WHERE id = ?", unassigned.ID)
	})
	partners, err := ListProviderPartners()
	require.NoError(t, err)
	require.Len(t, partners, 1)
	assert.Equal(t, partner.ID, partners[0].ID)

	funding := &ProviderFunding{ProviderPartnerID: partner.ID, ChannelID: channel.Id, Reference: "provider-deposit-1", AmountMinor: 12500, Currency: "usd", Source: ProviderFundingSourceManual}
	record, replay, err := RecordProviderFunding(funding)
	require.NoError(t, err)
	assert.False(t, replay)
	assert.Equal(t, "USD", record.Currency)

	replayed, replay, err := RecordProviderFunding(&ProviderFunding{ProviderPartnerID: partner.ID, ChannelID: channel.Id, Reference: "provider-deposit-1", AmountMinor: 12500, Currency: "USD", Source: ProviderFundingSourceManual})
	require.NoError(t, err)
	assert.True(t, replay)
	assert.Equal(t, record.ID, replayed.ID)

	summary, err := SummarizeProviderFundings(ProviderFundingFilter{ProviderPartnerID: partner.ID})
	require.NoError(t, err)
	require.Len(t, summary, 1)
	assert.Equal(t, int64(12500), summary[0].AmountMinor)
}

func TestProviderPartnerHidesAPIKeyFromJSON(t *testing.T) {
	partner := &ProviderPartner{Code: "partner-secret-test", Name: "Secret Test", APIKey: "do-not-expose", APIEndpoint: "https://provider.example.test/usage"}
	require.NoError(t, normalizeProviderPartner(partner))
	encoded, err := common.Marshal(partner)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), partner.APIKey)
}
