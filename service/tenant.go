package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

func CreateTenant(slug, name string, userID int) (*model.Tenant, error) {
	var tenant *model.Tenant
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		tenant, err = model.CreateTenantWithOwner(tx, slug, name, userID)
		return err
	})
	return tenant, err
}

func CreateDomain(tenantID int, hostname string) (*model.TenantDomain, string, error) {
	normalized, err := model.NormalizeTenantHostname(hostname); if err != nil { return nil, "", err }
	token, hash, err := model.NewDomainChallenge(); if err != nil { return nil, "", err }
	domain := &model.TenantDomain{TenantId: tenantID, Hostname: normalized, Status: model.DomainPending, VerificationMethod: "dns_txt", VerificationTokenHash: hash, CreatedAt: common.GetTimestamp(), UpdatedAt: common.GetTimestamp()}
	if err := model.DB.Create(domain).Error; err != nil { return nil, "", err }
	return domain, token, nil
}

func VerifyDomain(domainID, tenantID int, token string) error {
	var domain model.TenantDomain
	if err := model.DB.Where("id = ? AND tenant_id = ?", domainID, tenantID).First(&domain).Error; err != nil { return err }
	if !model.VerifyTenantDomain(&domain, strings.TrimSpace(token)) { return errors.New("domain verification failed") }
	now := common.GetTimestamp(); domain.Status, domain.VerifiedAt, domain.LastCheckedAt, domain.UpdatedAt = model.DomainVerified, &now, &now, now
	return model.DB.Save(&domain).Error
}
