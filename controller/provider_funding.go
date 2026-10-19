package controller

import (
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

var providerFundingSyncRunning atomic.Bool
var providerFundingSyncStartedAt atomic.Int64
var providerFundingSyncCompletedAt atomic.Int64

type providerFundingRequest struct {
	ProviderPartnerID int64  `json:"provider_partner_id" binding:"required,gt=0"`
	ChannelID         int    `json:"channel_id"`
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
	partnerID, _ := strconv.ParseInt(c.Query("provider_partner_id"), 10, 64)
	items, total, err := model.ListProviderFundings(model.ProviderFundingFilter{
		ProviderPartnerID: partnerID,
		ChannelID:         channelID,
		Currency:          c.Query("currency"),
		Source:            c.Query("source"),
		Page:              page,
		PageSize:          pageSize,
	})
	if err != nil {
		common.ApiErrorT(c, "Failed to load provider funding records")
		return
	}
	summary, err := model.SummarizeProviderFundings(model.ProviderFundingFilter{
		ProviderPartnerID: partnerID,
		Currency:          c.Query("currency"),
	})
	if err != nil {
		common.ApiErrorT(c, "Failed to load provider funding summary")
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": max(page, 1), "page_size": min(max(pageSize, 1), 100), "summary": summary})
}

func CreateProviderFunding(c *gin.Context) {
	var req providerFundingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid provider funding")
		return
	}
	record, replay, err := model.RecordProviderFunding(&model.ProviderFunding{
		ProviderPartnerID: req.ProviderPartnerID, ChannelID: req.ChannelID, Reference: strings.TrimSpace(req.Reference), AmountMinor: req.AmountMinor,
		Currency: req.Currency, Source: req.Source, ProviderReference: strings.TrimSpace(req.ProviderReference), Note: strings.TrimSpace(req.Note),
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

type providerPartnerRequest struct {
	Code        string `json:"code" binding:"max=64"`
	Name        string `json:"name" binding:"required,max=255"`
	ImageURL    string `json:"image_url" binding:"max=524288"`
	WebsiteURL  string `json:"website_url" binding:"max=1024"`
	APIKey      string `json:"api_key" binding:"max=512"`
	APIEndpoint string `json:"api_endpoint" binding:"required,max=1024"`
	Status      int    `json:"status"`
}

func ListProviderPartners(c *gin.Context) {
	partners, err := model.ListProviderPartners()
	if err != nil {
		common.ApiErrorT(c, "Failed to load provider partners")
		return
	}
	common.ApiSuccess(c, partners)
}

func CreateProviderPartner(c *gin.Context) {
	var req providerPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid provider partner")
		return
	}
	partner := &model.ProviderPartner{Code: req.Code, Name: req.Name, ImageURL: req.ImageURL, WebsiteURL: req.WebsiteURL, APIKey: req.APIKey, APIEndpoint: req.APIEndpoint, Status: req.Status}
	if err := model.CreateProviderPartner(partner); err != nil {
		if errors.Is(err, model.ErrProviderPartnerExists) {
			common.ApiErrorT(c, "Provider partner already exists")
			return
		}
		if errors.Is(err, model.ErrInvalidProviderPartner) {
			common.ApiErrorT(c, "Invalid provider partner")
			return
		}
		common.ApiErrorT(c, "Failed to create provider partner")
		return
	}
	common.ApiSuccess(c, partner)
}

func UpdateProviderPartner(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiErrorT(c, "Invalid provider partner")
		return
	}
	var req providerPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid provider partner")
		return
	}
	partner := &model.ProviderPartner{ID: id, Name: req.Name, ImageURL: req.ImageURL, WebsiteURL: req.WebsiteURL, APIKey: req.APIKey, APIEndpoint: req.APIEndpoint, Status: req.Status}
	if err := model.UpdateProviderPartner(partner); err != nil {
		if errors.Is(err, model.ErrProviderPartnerExists) {
			common.ApiErrorT(c, "Provider partner already exists")
			return
		}
		if errors.Is(err, model.ErrInvalidProviderPartner) {
			common.ApiErrorT(c, "Invalid provider partner")
			return
		}
		common.ApiErrorT(c, "Failed to update provider partner")
		return
	}
	updated, err := model.GetProviderPartner(id)
	if err != nil {
		common.ApiErrorT(c, "Failed to load provider partner")
		return
	}
	updated.APIKeyConfigured = updated.APIKey != ""
	common.ApiSuccess(c, updated)
}

func DeleteProviderPartner(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiErrorT(c, "Invalid provider partner")
		return
	}
	if err := model.DeleteProviderPartner(id); err != nil {
		if errors.Is(err, model.ErrProviderPartnerInUse) {
			common.ApiErrorT(c, "Provider partner cannot be deleted while it has channels or funding records")
			return
		}
		common.ApiErrorT(c, "Failed to delete provider partner")
		return
	}
	common.ApiSuccess(c, gin.H{"id": id})
}

type channelProviderPartnerRequest struct {
	ProviderPartnerID *int64 `json:"provider_partner_id"`
}

func SetChannelProviderPartner(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	var req channelProviderPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if err := model.SetChannelProviderPartner(channelID, req.ProviderPartnerID); err != nil {
		common.ApiErrorT(c, "Failed to assign provider partner")
		return
	}
	common.ApiSuccess(c, gin.H{"channel_id": channelID, "provider_partner_id": req.ProviderPartnerID})
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
