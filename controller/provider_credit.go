package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type providerCreditThresholdRequest struct {
	Threshold float64 `json:"threshold" binding:"gte=0"`
}

func GetProviderCredit(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		common.ApiErrorT(c, "Channel not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": model.ProviderCreditSnapshot(channel)})
}

func UpdateProviderCredit(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	var req providerCreditThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	channel, err := model.SetProviderCreditThreshold(id, req.Threshold)
	if err != nil {
		if err == model.ErrInvalidProviderCreditThreshold {
			common.ApiErrorT(c, "Invalid parameters")
			return
		}
		common.ApiErrorT(c, "Failed to update provider credit settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": model.ProviderCreditSnapshot(channel)})
}
