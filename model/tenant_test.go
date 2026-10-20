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

	_, err = CreateTenantWithOwner(db, "missing-owner", "Missing Owner", 99999)
	require.ErrorIs(t, err, ErrInvalidTenantOwner)
}

func TestNormalizeTenantHostname(t *testing.T) {
	tests := []struct { input, expected string; valid bool }{
		{"Console.Example.com.", "console.example.com", true},
		{"console.example.com:443", "", false},
		{"user@example.com", "", false},
		{"-console.example.com", "", false},
	}
	for _, test := range tests {
		got, err := NormalizeTenantHostname(test.input)
		if test.valid { require.NoError(t, err); require.Equal(t, test.expected, got) } else { require.Error(t, err) }
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
