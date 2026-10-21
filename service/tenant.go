package service

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

// TenantDomainDNSResolver is replaceable in tests so verification never needs
// to depend on the public DNS network.
var TenantDomainDNSResolver DomainDNSResolver = net.DefaultResolver

type DomainDNSResolver interface {
	LookupIP(context.Context, string, string) ([]net.IP, error)
	LookupCNAME(context.Context, string) (string, error)
}

func PromoteTenant(slug, name, partnerCode string, userID int) (*model.Tenant, error) {
	var tenant *model.Tenant
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		tenant, err = model.CreateTenantWithOwnerAndCode(tx, slug, name, partnerCode, userID)
		return err
	})
	return tenant, err
}

// PromoteUserPartner creates or re-enables the tenant owned by a user. Tenant
// metadata is deliberately generated here so the admin action never accepts
// or displays partner identifiers.
func PromoteUserPartner(userID int) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var membership model.TenantMembership
		err := tx.Where("user_id = ? AND role_key = ?", userID, model.TenantRoleOwner).
			Order("id DESC").First(&membership).Error
		if err == nil {
			now := common.GetTimestamp()
			if err = tx.Model(&model.Tenant{}).Where("id = ?", membership.TenantId).
				Updates(map[string]any{"status": model.TenantActive, "updated_at": now}).Error; err != nil {
				return err
			}
			return tx.Model(&model.TenantMembership{}).Where("id = ?", membership.Id).
				Updates(map[string]any{"status": model.MembershipActive, "updated_at": now}).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		name := "Partner " + strconv.Itoa(userID)
		slug := "partner-" + strconv.Itoa(userID)
		_, err = model.CreateTenantWithOwnerAndCode(tx, slug, name, "", userID)
		if err != nil {
			return err
		}
		return nil
	})
}

// DemoteUserPartner disables all partner tenants owned by a user and removes
// the active owner memberships while retaining tenant data for auditability.
func DemoteUserPartner(userID int) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var memberships []model.TenantMembership
		if err := tx.Where("user_id = ? AND role_key = ? AND status = ?", userID, model.TenantRoleOwner, model.MembershipActive).Find(&memberships).Error; err != nil {
			return err
		}
		now := common.GetTimestamp()
		for _, membership := range memberships {
			if err := tx.Model(&model.Tenant{}).Where("id = ?", membership.TenantId).
				Updates(map[string]any{"status": "disabled", "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.TenantMembership{}).Where("id = ?", membership.Id).
				Updates(map[string]any{"status": model.MembershipRemoved, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func CreateDomain(tenantID int, hostname string) (*model.TenantDomain, error) {
	normalized, err := model.NormalizeTenantHostname(hostname)
	if err != nil {
		return nil, err
	}
	return savePendingDomain(tenantID, normalized, nil)
}

func UpdateDomain(domainID, tenantID int, hostname string) (*model.TenantDomain, error) {
	normalized, err := model.NormalizeTenantHostname(hostname)
	if err != nil {
		return nil, err
	}
	var domain model.TenantDomain
	if err = model.DB.Where("id = ? AND tenant_id = ?", domainID, tenantID).First(&domain).Error; err != nil {
		return nil, err
	}
	return savePendingDomain(tenantID, normalized, &domain)
}

func savePendingDomain(tenantID int, hostname string, domain *model.TenantDomain) (*model.TenantDomain, error) {
	var existing model.TenantDomain
	query := model.DB.Where("hostname = ?", hostname)
	if domain != nil {
		query = query.Where("id <> ?", domain.Id)
	}
	if err := query.First(&existing).Error; err == nil {
		return nil, model.ErrTenantDomainTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := common.GetTimestamp()
	if domain == nil {
		domain = &model.TenantDomain{TenantId: tenantID, CreatedAt: now}
	}
	domain.Hostname = hostname
	domain.Status = model.DomainPending
	domain.VerificationMethod = "dns"
	domain.VerifiedAt = nil
	domain.LastCheckedAt = nil
	domain.UpdatedAt = now
	if err := model.DB.Save(domain).Error; err != nil {
		return nil, err
	}
	return domain, nil
}

func VerifyDomain(domainID, tenantID int) error {
	var domain model.TenantDomain
	if err := model.DB.Where("id = ? AND tenant_id = ?", domainID, tenantID).First(&domain).Error; err != nil {
		return err
	}
	if domain.Status != model.DomainPending || !tenantDomainDNSMatches(domain.Hostname) {
		return errors.New("domain verification failed")
	}
	now := common.GetTimestamp()
	domain.Status, domain.VerifiedAt, domain.LastCheckedAt, domain.UpdatedAt = model.DomainVerified, &now, &now, now
	return model.DB.Save(&domain).Error
}

func tenantDomainDNSMatches(hostname string) bool {
	common.OptionMapRWMutex.RLock()
	cnameTarget := strings.TrimSpace(common.Interface2String(common.OptionMap["TenantDomainDNSRecordCNAME"]))
	aRecordIP := strings.TrimSpace(common.Interface2String(common.OptionMap["TenantDomainDNSRecordA"]))
	common.OptionMapRWMutex.RUnlock()
	if cnameTarget == "" && aRecordIP == "" {
		return false
	}

	ctx := context.Background()
	if cnameTarget != "" {
		cname, err := TenantDomainDNSResolver.LookupCNAME(ctx, hostname)
		if err == nil && normalizeDNSName(cname) == normalizeDNSName(cnameTarget) {
			return true
		}
	}
	if aRecordIP != "" {
		expected := net.ParseIP(aRecordIP)
		if expected == nil {
			return false
		}
		ips, err := TenantDomainDNSResolver.LookupIP(ctx, "ip", hostname)
		if err == nil {
			for _, ip := range ips {
				if expected.Equal(ip) {
					return true
				}
			}
		}
	}
	return false
}

func normalizeDNSName(name string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
}

func DeleteDomain(domainID, tenantID int) error {
	result := model.DB.Where("id = ? AND tenant_id = ?", domainID, tenantID).Delete(&model.TenantDomain{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
