package controller

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestParseOpenRouterCostsRejectsMalformedAndZeroPrices(t *testing.T) {
	body := `{"data":[{"id":"bad","pricing":{"prompt":"nope","completion":"1"}},{"id":"free","pricing":{"prompt":"0","completion":"0"}},{"id":"paid","pricing":{"prompt":"0.000001","completion":"0.000002"}}]}`
	costs, err := parseOpenRouterCosts(strings.NewReader(body), 7)
	require.NoError(t, err)
	require.Len(t, costs, 1)
	assert.Equal(t, "paid", costs[0].Model)
	assert.Equal(t, int64(1_000_000), costs[0].InputUSDPerMillionMicros)
	assert.Equal(t, int64(2_000_000), costs[0].OutputUSDPerMillionMicros)
}

func TestPreviewUpstreamCostsRepresentsOllamaAsUnsupported(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.ChannelModelCost{}))
	previous := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previous })
	baseURL := "http://127.0.0.1:1"
	require.NoError(t, db.Create(&model.Channel{Id: 9, Name: "local", Type: constant.ChannelTypeOllama, BaseURL: &baseURL}).Error)
	results := fetchUpstreamCosts(t.Context(), []int{9}, 1)
	require.Len(t, results, 1)
	assert.False(t, results[0].Supported)
	assert.Equal(t, "manual", results[0].Source)
	assert.Contains(t, results[0].Error, "manual")
}

func TestApplyUpstreamCostsPreservesManualAndRejectsStalePreview(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ChannelModelCost{}))
	previous := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previous })
	manual := &model.ChannelModelCost{ChannelID: 3, Model: "m", InputUSDPerMillionMicros: 99, Source: "manual", Manual: true, Valid: true}
	require.NoError(t, model.UpsertChannelModelCost(manual))
	version, err := model.ChannelModelCostVersion(3)
	require.NoError(t, err)
	require.NoError(t, model.ApplyChannelModelCosts(3, version, []model.ChannelModelCost{
		{Model: "m", InputUSDPerMillionMicros: 1, Source: "openrouter", Valid: true},
		{Model: "other", InputUSDPerMillionMicros: 1, Source: "openrouter", Valid: true},
	}))
	got, err := model.GetChannelModelCost(3, "m")
	require.NoError(t, err)
	assert.Equal(t, int64(99), got.InputUSDPerMillionMicros)

	err = model.ApplyChannelModelCosts(3, version, []model.ChannelModelCost{{Model: "new", InputUSDPerMillionMicros: 1, Source: "openrouter", Valid: true}})
	assert.ErrorIs(t, err, model.ErrChannelModelCostConflict)
}

func TestConvertModelsDevCostsSupportsLargeCatalog(t *testing.T) {
	models := make(map[string]modelsDevModel, maxUpstreamCostModels)
	input := 0.001
	for i := range 5001 {
		models["provider/model-"+strconv.Itoa(i)] = modelsDevModel{
			Cost: modelsDevCost{Input: &input},
		}
	}

	body, err := common.Marshal(map[string]modelsDevProvider{
		"provider": {Models: models},
	})
	require.NoError(t, err)

	costs, err := convertModelsDevCosts(bytes.NewReader(body), 1)
	require.NoError(t, err)
	require.Len(t, costs, 5001)
}
