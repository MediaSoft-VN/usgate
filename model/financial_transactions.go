package model

import "strings"

type FinancialTransactionFilter struct {
	OwnerType       string
	OwnerID         int64
	WalletAccountID int64
	Type            string
	Status          string
	Currency        string
	StartTimestamp  int64
	EndTimestamp    int64
	Page            int
	PageSize        int
}

func ListFinancialTransactions(filter FinancialTransactionFilter) ([]FinancialTransaction, int64, error) {
	query := DB.Model(&FinancialTransaction{})
	if filter.OwnerType != "" {
		query = query.Where("wallet_account_id IN (?)", DB.Model(&WalletAccount{}).Select("id").Where("owner_type = ?", filter.OwnerType))
	}
	if filter.OwnerID > 0 {
		query = query.Where("wallet_account_id IN (?)", DB.Model(&WalletAccount{}).Select("id").Where("owner_id = ?", filter.OwnerID))
	}
	if filter.WalletAccountID > 0 {
		query = query.Where("wallet_account_id = ?", filter.WalletAccountID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Currency != "" {
		query = query.Where("currency = ?", strings.ToUpper(strings.TrimSpace(filter.Currency)))
	}
	if filter.StartTimestamp > 0 {
		query = query.Where("created_at >= ?", filter.StartTimestamp)
	}
	if filter.EndTimestamp > 0 {
		query = query.Where("created_at <= ?", filter.EndTimestamp)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := max(filter.Page, 1)
	pageSize := min(max(filter.PageSize, 1), 100)
	var transactions []FinancialTransaction
	err := query.Select("id", "transaction_no", "wallet_account_id", "idempotency_key", "type", "status", "amount_minor", "currency", "provider", "external_reference", "created_at", "completed_at").
		Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&transactions).Error
	return transactions, total, err
}
