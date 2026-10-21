package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
	"gorm.io/gorm"
)

const (
	TenantActive        = "active"
	TenantSuspended     = "suspended"
	TenantDeleted       = "deleted"
	TenantRoleOwner     = "owner"
	TenantRoleAdmin     = "admin"
	TenantRoleMember    = "member"
	TenantRoleViewer    = "viewer"
	MembershipActive    = "active"
	MembershipInvited   = "invited"
	MembershipSuspended = "suspended"
	MembershipRemoved   = "removed"
	DomainPending       = "pending"
	DomainVerified      = "verified"
	DomainDisabled      = "disabled"
)

type Tenant struct {
	Id          int     `json:"id"`
	Slug        string  `json:"slug" gorm:"size:64;uniqueIndex"`
	PartnerCode *string `json:"partner_code" gorm:"size:64;uniqueIndex:idx_tenant_partner_code"`
	Name        string  `json:"name" gorm:"size:128"`
	Status      string  `json:"status" gorm:"size:16;index"`
	CreatedBy   int     `json:"created_by" gorm:"index"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
	DeletedAt   *int64  `json:"deleted_at" gorm:"index"`
}

type TenantMembership struct {
	Id         int    `json:"id"`
	TenantId   int    `json:"tenant_id" gorm:"uniqueIndex:idx_tenant_membership"`
	UserId     int    `json:"user_id" gorm:"uniqueIndex:idx_tenant_membership;index"`
	RoleKey    string `json:"role_key" gorm:"size:16"`
	Status     string `json:"status" gorm:"size:16;index"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	LastSeenAt *int64 `json:"last_seen_at"`
}

type TenantDomain struct {
	Id                    int    `json:"id"`
	TenantId              int    `json:"tenant_id" gorm:"index"`
	Hostname              string `json:"hostname" gorm:"size:253;uniqueIndex"`
	Status                string `json:"status" gorm:"size:16;index"`
	VerificationMethod    string `json:"verification_method" gorm:"size:16"`
	VerificationTokenHash string `json:"-" gorm:"size:64"`
	VerifiedAt            *int64 `json:"verified_at"`
	LastCheckedAt         *int64 `json:"last_checked_at"`
	CreatedAt             int64  `json:"created_at"`
	UpdatedAt             int64  `json:"updated_at"`
}

var ErrTenantNotFound = errors.New("tenant not found")
var ErrTenantForbidden = errors.New("tenant access denied")
var ErrFinalOwner = errors.New("the final active owner cannot be removed or demoted")
var ErrInvalidTenantOwner = errors.New("tenant owner must be an existing enabled user")
var ErrInvalidTenantMember = errors.New("tenant member must be an existing enabled user")
var ErrPartnerCodeTaken = errors.New("partner code already exists")
var ErrTenantDomainAlreadyAssigned = errors.New("domain is already assigned to this tenant")
var ErrTenantDomainTaken = errors.New("domain is already assigned to another tenant")

func normalizeTenantSlug(slug string) (string, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if len(slug) == 0 || len(slug) > 64 || strings.Trim(slug, "-") != slug {
		return "", errors.New("invalid tenant slug")
	}
	for _, r := range slug {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return "", errors.New("invalid tenant slug")
		}
	}
	return slug, nil
}

func NormalizeTenantPartnerCode(code string) (string, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if len(code) == 0 || len(code) > 64 || strings.Trim(code, "-") != code {
		return "", errors.New("invalid partner code")
	}
	for _, r := range code {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return "", errors.New("invalid partner code")
		}
	}
	return code, nil
}

func GenerateTenantPartnerCode(name string) (string, error) {
	var builder strings.Builder
	separator := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if separator && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			builder.WriteRune(r)
			separator = false
		} else {
			separator = true
		}
	}
	code := builder.String()
	if len(code) > 64 {
		code = strings.TrimRight(code[:64], "-")
	}
	return NormalizeTenantPartnerCode(code)
}

func NormalizeTenantHostname(host string) (string, error) {
	host, err := idna.Lookup.ToASCII(strings.TrimSuffix(strings.TrimSpace(host), "."))
	host = strings.ToLower(host)
	if err != nil || host == "" || host == "localhost" || strings.ContainsAny(host, "@/?#\\") {
		return "", errors.New("invalid hostname")
	}
	if parsed := net.ParseIP(host); parsed != nil || strings.Contains(host, ":") {
		return "", errors.New("invalid hostname")
	}
	if len(host) > 253 || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return "", errors.New("invalid hostname")
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", errors.New("invalid hostname")
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return "", errors.New("invalid hostname")
			}
		}
	}
	if strings.Count(host, ".") == 0 {
		return "", errors.New("invalid hostname")
	}
	tld := host[strings.LastIndexByte(host, '.')+1:]
	if tld == "example" || tld == "invalid" || tld == "localhost" || tld == "test" {
		return "", errors.New("invalid hostname")
	}
	if _, err := publicsuffix.EffectiveTLDPlusOne(host); err != nil {
		return "", errors.New("invalid hostname")
	}
	return host, nil
}

func CreateTenantWithOwner(tx *gorm.DB, slug, name string, userID int) (*Tenant, error) {
	return CreateTenantWithOwnerAndCode(tx, slug, name, "", userID)
}

func CreateTenantWithOwnerAndCode(tx *gorm.DB, slug, name, partnerCode string, userID int) (*Tenant, error) {
	var err error
	slug, err = normalizeTenantSlug(slug)
	if err != nil || strings.TrimSpace(name) == "" || len(name) > 128 {
		return nil, errors.New("invalid tenant")
	}
	if strings.TrimSpace(partnerCode) == "" {
		partnerCode, err = GenerateTenantPartnerCode(name)
	} else {
		partnerCode, err = NormalizeTenantPartnerCode(partnerCode)
	}
	if err != nil {
		return nil, err
	}
	var existing Tenant
	if err = tx.Where("partner_code = ?", partnerCode).First(&existing).Error; err == nil {
		return nil, ErrPartnerCodeTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var user User
	if err = tx.Where("id = ? AND status = ?", userID, common.UserStatusEnabled).First(&user).Error; err != nil {
		return nil, ErrInvalidTenantOwner
	}
	now := common.GetTimestamp()
	tenant := &Tenant{Slug: slug, PartnerCode: &partnerCode, Name: strings.TrimSpace(name), Status: TenantActive, CreatedBy: userID, CreatedAt: now, UpdatedAt: now}
	if err = tx.Create(tenant).Error; err != nil {
		var duplicate Tenant
		if tx.Where("partner_code = ?", partnerCode).First(&duplicate).Error == nil {
			return nil, ErrPartnerCodeTaken
		}
		return nil, err
	}
	membership := &TenantMembership{TenantId: tenant.Id, UserId: userID, RoleKey: TenantRoleOwner, Status: MembershipActive, CreatedAt: now, UpdatedAt: now}
	if err = tx.Create(membership).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}

func ActiveMembership(tenantID, userID int) (TenantMembership, error) {
	var membership TenantMembership
	err := DB.Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, MembershipActive).First(&membership).Error
	return membership, err
}

func TenantPermission(role, action string) bool {
	if role == TenantRoleOwner {
		return true
	}
	switch action {
	case "tenant_update":
		return false
	case "read":
		return role == TenantRoleAdmin || role == TenantRoleMember || role == TenantRoleViewer
	case "member_admin", "domain_admin":
		return role == TenantRoleAdmin
	default:
		return false
	}
}

func ValidTenantRole(role string) bool {
	return role == TenantRoleOwner || role == TenantRoleAdmin || role == TenantRoleMember || role == TenantRoleViewer
}

func ActiveOwnerCount(tx *gorm.DB, tenantID int) (int64, error) {
	var count int64
	err := tx.Model(&TenantMembership{}).Where("tenant_id = ? AND role_key = ? AND status = ?", tenantID, TenantRoleOwner, MembershipActive).Count(&count).Error
	return count, err
}

func ChangeMembership(tx *gorm.DB, tenantID, userID int, role, status string) error {
	var membership TenantMembership
	if err := tx.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&membership).Error; err != nil {
		return err
	}
	if membership.RoleKey == TenantRoleOwner && membership.Status == MembershipActive && (role != TenantRoleOwner || status != MembershipActive) {
		count, err := ActiveOwnerCount(tx, tenantID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrFinalOwner
		}
	}
	membership.RoleKey, membership.Status, membership.UpdatedAt = role, status, common.GetTimestamp()
	return tx.Save(&membership).Error
}

func NewDomainChallenge() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(buf)
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

func VerifyTenantDomain(domain *TenantDomain, token string) bool {
	sum := sha256.Sum256([]byte(token))
	return domain.Status == DomainPending && hex.EncodeToString(sum[:]) == domain.VerificationTokenHash
}

func ResolveVerifiedTenant(host string, userID int) (*Tenant, error) {
	tenant, err := ResolveVerifiedTenantForRegistration(host)
	if err != nil {
		return nil, err
	}
	if _, err = ActiveMembership(tenant.Id, userID); err != nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

// ResolveVerifiedTenantForRegistration resolves a tenant from the request
// host without using any client-supplied tenant identifier. Unlike the normal
// tenant resolver, it does not require membership because registration creates
// that membership as part of the same operation.
func ResolveVerifiedTenantForRegistration(host string) (*Tenant, error) {
	normalized, err := NormalizeTenantHostname(host)
	if err != nil {
		return nil, ErrTenantNotFound
	}
	var domain TenantDomain
	if err = DB.Where("hostname = ? AND status = ?", normalized, DomainVerified).First(&domain).Error; err != nil {
		return nil, ErrTenantNotFound
	}
	var tenant Tenant
	if err = DB.Where("id = ? AND status = ? AND deleted_at IS NULL", domain.TenantId, TenantActive).First(&tenant).Error; err != nil {
		return nil, ErrTenantNotFound
	}
	return &tenant, nil
}

// EnsureActiveTenantMembership makes a user an active member of a tenant. It
// is idempotent so retries cannot create duplicate memberships, and it also
// reactivates a prior membership for the same user and tenant.
func EnsureActiveTenantMembership(tx *gorm.DB, tenantID, userID int) error {
	var membership TenantMembership
	err := tx.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&membership).Error
	now := common.GetTimestamp()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&TenantMembership{
			TenantId: tenantID, UserId: userID, RoleKey: TenantRoleMember,
			Status: MembershipActive, CreatedAt: now, UpdatedAt: now,
		}).Error
	}
	if err != nil {
		return err
	}
	if membership.Status == MembershipActive {
		return nil
	}
	return tx.Model(&membership).Updates(map[string]any{
		"status": MembershipActive, "role_key": TenantRoleMember, "updated_at": now,
	}).Error
}

func MigrateTenants() error {
	if err := DB.AutoMigrate(&Tenant{}, &TenantMembership{}, &TenantDomain{}); err != nil {
		return err
	}
	var tenants []Tenant
	if err := DB.Where("partner_code IS NULL OR partner_code = ?", "").Order("id").Find(&tenants).Error; err != nil {
		return err
	}
	for i := range tenants {
		code, err := GenerateTenantPartnerCode(tenants[i].Name)
		if err != nil {
			code = "tenant-" + strings.TrimSpace(strings.ToLower(strconv.Itoa(tenants[i].Id)))
		}
		base := code
		for suffix := 2; ; suffix++ {
			var count int64
			if err = DB.Model(&Tenant{}).Where("partner_code = ? AND id <> ?", code, tenants[i].Id).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				break
			}
			code = base + "-" + strconv.Itoa(suffix)
		}
		if err = DB.Model(&Tenant{}).Where("id = ?", tenants[i].Id).Update("partner_code", code).Error; err != nil {
			return err
		}
	}
	var tenant Tenant
	if err := DB.Where("slug = ?", "default").First(&tenant).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var users []User
	if err := DB.Where("status = ?", common.UserStatusEnabled).Order("id").Find(&users).Error; err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	now := common.GetTimestamp()
	partnerCode := "default"
	tenant = Tenant{Slug: "default", PartnerCode: &partnerCode, Name: "Default", Status: TenantActive, CreatedBy: users[0].Id, CreatedAt: now, UpdatedAt: now}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}
		for i := range users {
			role := TenantRoleMember
			if i == 0 {
				role = TenantRoleOwner
			} else if users[i].Role >= common.RoleAdminUser {
				role = TenantRoleAdmin
			}
			membership := TenantMembership{TenantId: tenant.Id, UserId: users[i].Id, RoleKey: role, Status: MembershipActive, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
