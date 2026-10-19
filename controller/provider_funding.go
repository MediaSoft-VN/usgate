package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

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
	summary, err := model.SummarizeProviderFundings()
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
		ChannelID: req.ChannelID, Reference: strings.TrimSpace(req.Reference), AmountMinor: req.AmountMinor,
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
