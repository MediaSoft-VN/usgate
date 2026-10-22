package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ListFinancialWallets exposes the real financial wallet balances separately
// from legacy user quota. Only administrators can access this inventory.
func ListFinancialWallets(c *gin.Context) {
	ownerType := c.DefaultQuery("owner_type", model.FinancialOwnerUser)
	ownerID := int64(0)
	if rawID := c.Query("owner_id"); rawID != "" {
		parsedID, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil || parsedID <= 0 {
			common.ApiErrorT(c, "Invalid wallet owner")
			return
		}
		ownerID = parsedID
	}
	wallets, err := model.ListWalletAccounts(ownerType, ownerID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": wallets})
}
