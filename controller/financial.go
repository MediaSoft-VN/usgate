package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func ListFinancialTransactions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page = max(page, 1)
	pageSize = min(max(pageSize, 1), 100)
	ownerID, _ := strconv.ParseInt(c.Query("owner_id"), 10, 64)
	walletID, _ := strconv.ParseInt(c.Query("wallet_account_id"), 10, 64)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	transactions, total, err := model.ListFinancialTransactions(model.FinancialTransactionFilter{
		OwnerType:       strings.TrimSpace(c.Query("owner_type")),
		OwnerID:         ownerID,
		WalletAccountID: walletID,
		Type:            strings.TrimSpace(c.Query("type")),
		Status:          strings.TrimSpace(c.Query("status")),
		Currency:        c.Query("currency"),
		StartTimestamp:  startTimestamp,
		EndTimestamp:    endTimestamp,
		Page:            page,
		PageSize:        pageSize,
	})
	if err != nil {
		common.ApiErrorT(c, "Failed to load financial transactions")
		return
	}
	common.ApiSuccess(c, gin.H{
		"items":     transactions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
