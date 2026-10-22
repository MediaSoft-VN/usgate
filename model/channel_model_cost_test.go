package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
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

func TestChannelRoutingPolicyLowestCostUsesCostBucketsAndPriorityFallback(t *testing.T) {
	db := newChannelModelCostTestDB(t)
	previousDB := DB
	previousMemoryCache := common.MemoryCacheEnabled
	DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		DB = previousDB
		common.MemoryCacheEnabled = previousMemoryCache
	})
	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}, &ChannelModelCost{}))
	modelName := "routing-model"
	for _, fixture := range []struct {
		id       int
		priority int64
		weight   uint
	}{
		{id: 1, priority: 100, weight: 100},
		{id: 2, priority: 1, weight: 100},
		{id: 3, priority: 1, weight: 100},
		{id: 4, priority: 1, weight: 100},
	} {
		priority, weight := fixture.priority, fixture.weight
		require.NoError(t, db.Create(&Channel{
			Id: fixture.id, Type: constant.ChannelTypeOpenAI, Key: "key", Status: common.ChannelStatusEnabled,
			Name: "channel", Weight: &weight, Models: modelName, Group: "default", Priority: &priority,
		}).Error)
		require.NoError(t, db.Create(&Ability{
			Group: "default", Model: modelName, ChannelId: fixture.id, Enabled: true,
			Priority: &priority, Weight: weight,
		}).Error)
	}
	require.NoError(t, UpsertChannelModelCost(&ChannelModelCost{ChannelID: 1, Model: modelName, InputUSDPerMillionMicros: 100, Valid: true}))
	require.NoError(t, UpsertChannelModelCost(&ChannelModelCost{ChannelID: 2, Model: modelName, InputUSDPerMillionMicros: 10, Valid: true}))
	require.NoError(t, UpsertChannelModelCost(&ChannelModelCost{ChannelID: 3, Model: modelName, InputUSDPerMillionMicros: 10, Valid: true}))
	require.NoError(t, UpsertChannelModelCost(&ChannelModelCost{ChannelID: 4, Model: modelName, InputUSDPerMillionMicros: 1, Valid: false}))

	first, err := GetChannelWithPolicy("default", modelName, 0, nil, "lowest_cost")
	require.NoError(t, err)
	assert.Contains(t, []int{2, 3}, first.Id)
	second, err := GetChannelWithPolicy("default", modelName, 0, nil, "lowest_cost")
	require.NoError(t, err)
	assert.Contains(t, []int{2, 3}, second.Id, "equal costs remain one weighted bucket")
	nextCost, err := GetChannelWithPolicy("default", modelName, 1, nil, "lowest_cost")
	require.NoError(t, err)
	assert.Equal(t, 1, nextCost.Id, "retry advances to the next known cost bucket")
	unknown, err := GetChannelWithPolicy("default", modelName, 2, nil, "lowest_cost")
	require.NoError(t, err)
	assert.Equal(t, 4, unknown.Id, "retry advances to the unknown-cost bucket")
	require.NoError(t, db.Model(&ChannelModelCost{}).Where("model = ?", modelName).Update("valid", false).Error)
	fallback, err := GetChannelWithPolicy("default", modelName, 0, nil, "lowest_cost")
	require.NoError(t, err)
	assert.Equal(t, 1, fallback.Id, "without valid costs lowest_cost falls back to priority")

	priority, err := GetChannelWithPolicy("default", modelName, 0, nil, "priority")
	require.NoError(t, err)
	assert.Equal(t, 1, priority.Id, "priority remains the default routing policy")
}

func TestBuildRequestPolicyDefaultsToPriorityAndAllowsLowestCost(t *testing.T) {
	defaultPolicy, err := BuildRequestPolicy(nil)
	require.NoError(t, err)
	assert.Equal(t, "priority", defaultPolicy.RoutingPolicy)
	lowestCost, err := BuildRequestPolicy(map[string]string{"ChannelRoutingPolicy": "lowest_cost"})
	require.NoError(t, err)
	assert.Equal(t, "lowest_cost", lowestCost.RoutingPolicy)
	_, err = BuildRequestPolicy(map[string]string{"ChannelRoutingPolicy": "unknown"})
	assert.Error(t, err)
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
