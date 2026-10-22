package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newChannelModelCostTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestChannelModelCostValidate(t *testing.T) {
	cost := &ChannelModelCost{ChannelID: 1, Model: "  gpt-test  ", InputUSDPerMillionMicros: 10}
	require.NoError(t, cost.Validate())
	assert.Equal(t, "gpt-test", cost.Model)

	assert.ErrorIs(t, (&ChannelModelCost{Model: "gpt-test"}).Validate(), ErrChannelModelCostInvalidID)
	assert.ErrorIs(t, (&ChannelModelCost{ChannelID: 1}).Validate(), ErrChannelModelCostInvalidModel)
	assert.ErrorIs(t, (&ChannelModelCost{ChannelID: 1, Model: "gpt-test", OutputUSDPerMillionMicros: -1}).Validate(), ErrChannelModelCostNegative)
}

func TestChannelModelCostMigrationIsIdempotentAndUpsertIsUnique(t *testing.T) {
	db := newChannelModelCostTestDB(t)
	previous := DB
	DB = db
	t.Cleanup(func() { DB = previous })

	require.NoError(t, DB.AutoMigrate(&ChannelModelCost{}))
	require.NoError(t, DB.AutoMigrate(&ChannelModelCost{}))

	first := &ChannelModelCost{ChannelID: 7, Model: "model-a", InputUSDPerMillionMicros: 100, Source: "models.dev", Valid: true}
	require.NoError(t, UpsertChannelModelCost(first))
	second := &ChannelModelCost{ChannelID: 7, Model: "model-a", InputUSDPerMillionMicros: 200, OutputUSDPerMillionMicros: 300, Source: "manual", Manual: true, Valid: true}
	require.NoError(t, UpsertChannelModelCost(second))

	var count int64
	require.NoError(t, DB.Model(&ChannelModelCost{}).Where("channel_id = ? AND model = ?", 7, "model-a").Count(&count).Error)
	assert.Equal(t, int64(1), count)
	got, err := GetChannelModelCost(7, "model-a")
	require.NoError(t, err)
	assert.Equal(t, int64(200), got.InputUSDPerMillionMicros)
	assert.Equal(t, int64(300), got.OutputUSDPerMillionMicros)
	assert.Equal(t, "manual", got.Source)
}
