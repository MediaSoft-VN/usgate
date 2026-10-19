package controller

import (
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

var providerFundingSyncRunning atomic.Bool
var providerFundingSyncStartedAt atomic.Int64
var providerFundingSyncCompletedAt atomic.Int64

type providerFundingRequest struct {
	ChannelID         int    `json:"channel_id" binding:"required,gt=0"`
	Reference         string `json:"reference" binding:"required,max=255"`
	AmountMinor       int64  `json:"amount_minor" binding:"required,gt=0"`
	Currency          string `json:"currency" binding:"required,max=16"`
	Source            string `json:"source" binding:"required,oneof=manual sync"`
	ProviderReference string `json:"provider_reference" binding:"max=255"`
	Note              string `json:"note" binding:"max=2000"`
}

func ListProviderFunding(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	items, total, err := model.ListProviderFundings(model.ProviderFundingFilter{
		ChannelID: channelID,
		Currency:  c.Query("currency"),
		Source:    c.Query("source"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		common.ApiErrorT(c, "Failed to load provider funding records")
		return
	}
	wallets, err := model.SummarizeProviderFundingWallets()
	if err != nil {
		common.ApiErrorT(c, "Failed to load provider wallet summary")
		return
	}
	providerWallets, err := model.SummarizeProviderFundingByChannel()
	if err != nil {
		common.ApiErrorT(c, "Failed to load provider wallet summary")
		return
	}
	allocations, err := model.ListProviderFundingAllocationHistory()
	if err != nil {
		common.ApiErrorT(c, "Failed to load provider allocation history")
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": max(page, 1), "page_size": min(max(pageSize, 1), 100), "wallets": wallets, "provider_wallets": providerWallets, "allocations": allocations})
}

type providerFundingAllocationRequest struct {
	UserID      int    `json:"user_id" binding:"required,gt=0"`
	Reference   string `json:"reference" binding:"required,max=255"`
	AmountMinor int64  `json:"amount_minor" binding:"required,gt=0"`
	Currency    string `json:"currency" binding:"required,max=16"`
}

func AllocateProviderFunding(c *gin.Context) {
	var req providerFundingAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid provider funding allocation")
		return
	}
	quota, rate, err := service.ProviderFundingQuota(req.AmountMinor, req.Currency)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	allocation, replay, err := model.AllocateProviderFunding(&model.ProviderFundingAllocation{
		UserID: req.UserID, Reference: strings.TrimSpace(req.Reference), AmountMinor: req.AmountMinor,
		Currency: req.Currency, Quota: quota, ExchangeRate: rate, CreatedBy: c.GetString("username"),
	})
	if err != nil {
		switch {
		case errors.Is(err, model.ErrProviderFundingInsufficient):
			common.ApiErrorT(c, "Insufficient provider funding")
		case errors.Is(err, model.ErrProviderFundingConflict):
			common.ApiErrorT(c, "Provider allocation reference already exists with different details")
		case errors.Is(err, model.ErrWalletQuotaLimitExceeded):
			common.ApiErrorT(c, "User quota exceeds the supported range")
		default: common.ApiError(c, err)
		}
		return
	}
	params := model.AuditFields{"target_user_id": req.UserID, "reference": allocation.Reference, "amount_minor": allocation.AmountMinor, "currency": allocation.Currency, "quota": allocation.Quota, "replay": replay}
	recordManageAuditFor(c, req.UserID, "provider.funding_allocate", params)
	common.ApiSuccess(c, gin.H{"allocation": allocation, "replay": replay})
}

func CreateProviderFunding(c *gin.Context) {
	var req providerFundingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid provider funding")
		return
	}
	createdBy := c.GetString("username")
	if createdBy == "" {
		createdBy = strconv.Itoa(c.GetInt("id"))
	}
	record, replay, err := model.RecordProviderFunding(&model.ProviderFunding{
		ChannelID: req.ChannelID, Reference: strings.TrimSpace(req.Reference), AmountMinor: req.AmountMinor,
		Currency: req.Currency, Source: req.Source, ProviderReference: strings.TrimSpace(req.ProviderReference), Note: strings.TrimSpace(req.Note), CreatedBy: createdBy,
	})
	if err != nil {
		if errors.Is(err, model.ErrProviderFundingConflict) {
			common.ApiErrorT(c, "Provider funding reference already exists with different details")
			return
		}
		if errors.Is(err, model.ErrInvalidProviderFunding) {
			common.ApiErrorT(c, "Invalid provider funding")
			return
		}
		common.ApiErrorT(c, "Failed to record provider funding")
		return
	}
	common.ApiSuccess(c, gin.H{"record": record, "replay": replay})
}

func GetProviderFundingSyncStatus(c *gin.Context) {
	common.ApiSuccess(c, gin.H{
		"running":      providerFundingSyncRunning.Load(),
		"started_at":   providerFundingSyncStartedAt.Load(),
		"completed_at": providerFundingSyncCompletedAt.Load(),
	})
}

func StartProviderFundingSync(c *gin.Context) {
	if !providerFundingSyncRunning.CompareAndSwap(false, true) {
		common.ApiSuccess(c, gin.H{"running": true, "started_at": providerFundingSyncStartedAt.Load()})
		return
	}
	startedAt := time.Now().Unix()
	providerFundingSyncStartedAt.Store(startedAt)
	go func() {
		defer providerFundingSyncRunning.Store(false)
		defer providerFundingSyncCompletedAt.Store(time.Now().Unix())
		_ = updateAllChannelsBalance()
	}()
	common.ApiSuccess(c, gin.H{"running": true, "started_at": startedAt})
}
