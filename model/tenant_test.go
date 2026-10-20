package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreateTenantWithOwnerRequiresExistingEnabledUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Tenant{}, &TenantMembership{}))

	owner := User{Username: "partner", Password: "unused", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)

	tenant, err := CreateTenantWithOwner(db, "partner-tenant", "Partner Tenant", owner.Id)
	require.NoError(t, err)
	require.Equal(t, owner.Id, tenant.CreatedBy)

	var membership TenantMembership
	require.NoError(t, db.Where("tenant_id = ? AND user_id = ?", tenant.Id, owner.Id).First(&membership).Error)
	require.Equal(t, TenantRoleOwner, membership.RoleKey)
	require.NotNil(t, tenant.PartnerCode)
	require.Equal(t, "partner-tenant", *tenant.PartnerCode)

	_, err = CreateTenantWithOwner(db, "missing-owner", "Missing Owner", 99999)
	require.ErrorIs(t, err, ErrInvalidTenantOwner)
}

func TestCreateTenantPartnerCodeNormalizesAndEnforcesUniqueness(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Tenant{}, &TenantMembership{}))
	owner := User{Username: "partner-code-owner", Password: "unused", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)

	tenant, err := CreateTenantWithOwnerAndCode(db, "code-tenant", "Code Tenant", "  CUSTOM_CODE ", owner.Id)
	require.Error(t, err)
	require.Nil(t, tenant)
	tenant, err = CreateTenantWithOwnerAndCode(db, "code-tenant", "Code Tenant", " Custom-Code ", owner.Id)
	require.NoError(t, err)
	require.Equal(t, "custom-code", *tenant.PartnerCode)

	_, err = CreateTenantWithOwnerAndCode(db, "another-tenant", "Another Tenant", "CUSTOM-CODE", owner.Id)
	require.ErrorIs(t, err, ErrPartnerCodeTaken)
}

func TestGenerateTenantPartnerCode(t *testing.T) {
	code, err := GenerateTenantPartnerCode("Acme & Partners")
	require.NoError(t, err)
	require.Equal(t, "acme-partners", code)
}

func TestNormalizeTenantHostname(t *testing.T) {
	tests := []struct {
		input, expected string
		valid           bool
	}{
		{"Console.Example.com.", "console.example.com", true},
		{"console.example.com:443", "", false},
		{"user@example.com", "", false},
		{"-console.example.com", "", false},
	}
	for _, test := range tests {
		got, err := NormalizeTenantHostname(test.input)
		if test.valid {
			require.NoError(t, err)
			require.Equal(t, test.expected, got)
		} else {
			require.Error(t, err)
		}
	}
}

func TestVerifyTenantDomainRequiresPendingMatchingToken(t *testing.T) {
	token, hash, err := NewDomainChallenge()
	require.NoError(t, err)
	domain := TenantDomain{Status: DomainPending, VerificationTokenHash: hash}
	require.True(t, VerifyTenantDomain(&domain, token))
	require.False(t, VerifyTenantDomain(&domain, "wrong-token"))
	domain.Status = DomainDisabled
	require.False(t, VerifyTenantDomain(&domain, token))
}

func TestRegistrationTenantResolutionAndMembershipAreHostBoundAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Tenant{}, &TenantMembership{}, &TenantDomain{}))
	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })

	owner := User{Username: "registration-owner", Password: "unused", AffCode: "registration-owner-code", Status: common.UserStatusEnabled}
	newUser := User{Username: "registration-user", Password: "unused", AffCode: "registration-user-code", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&newUser).Error)
	first, err := CreateTenantWithOwner(db, "first-tenant", "First Tenant", owner.Id)
	require.NoError(t, err)
	second, err := CreateTenantWithOwner(db, "second-tenant", "Second Tenant", owner.Id)
	require.NoError(t, err)
	require.NoError(t, db.Create(&TenantDomain{TenantId: first.Id, Hostname: "first.example.com", Status: DomainVerified}).Error)
	require.NoError(t, db.Create(&TenantDomain{TenantId: second.Id, Hostname: "pending.example.com", Status: DomainPending}).Error)

	resolved, err := ResolveVerifiedTenantForRegistration("FIRST.EXAMPLE.COM.")
	require.NoError(t, err)
	require.Equal(t, first.Id, resolved.Id)
	_, err = ResolveVerifiedTenantForRegistration("pending.example.com")
	require.ErrorIs(t, err, ErrTenantNotFound)
	_, err = ResolveVerifiedTenantForRegistration("unknown.example.com")
	require.ErrorIs(t, err, ErrTenantNotFound)

	require.NoError(t, EnsureActiveTenantMembership(db, first.Id, newUser.Id))
	require.NoError(t, EnsureActiveTenantMembership(db, first.Id, newUser.Id))
	var memberships []TenantMembership
	require.NoError(t, db.Where("user_id = ?", newUser.Id).Find(&memberships).Error)
	require.Len(t, memberships, 1)
	require.Equal(t, MembershipActive, memberships[0].Status)
	_, err = ActiveMembership(second.Id, newUser.Id)
	require.Error(t, err)
}
