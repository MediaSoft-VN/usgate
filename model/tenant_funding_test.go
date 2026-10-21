package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFundTenantMemberIsAtomicScopedAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:tenant-funding-test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Tenant{}, &TenantMembership{}, &WalletAccount{}, &FinancialTransaction{}, &LedgerEntry{}, &TenantFunding{}))
	previous := DB
	DB = db
	t.Cleanup(func() { DB = previous })
	owner := User{Username: "tenant-funding-owner", Password: "unused", AffCode: "tenant-funding-owner", Status: common.UserStatusEnabled}
	member := User{Username: "tenant-funding-member", Password: "unused", AffCode: "tenant-funding-member", Status: common.UserStatusEnabled}
	other := User{Username: "tenant-funding-other", Password: "unused", AffCode: "tenant-funding-other", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&member).Error)
	require.NoError(t, db.Create(&other).Error)
	tenant, err := CreateTenantWithOwner(db, "funding-tenant", "Funding Tenant", owner.Id)
	require.NoError(t, err)
	otherTenant, err := CreateTenantWithOwner(db, "other-funding-tenant", "Other Funding Tenant", other.Id)
	require.NoError(t, err)
	require.NoError(t, db.Create(&TenantMembership{TenantId: tenant.Id, UserId: member.Id, RoleKey: TenantRoleMember, Status: MembershipActive}).Error)
	wallet, err := GetOrCreateWalletAccount(FinancialOwnerOrganization, int64(tenant.Id), "USD")
	require.NoError(t, err)
	require.NoError(t, db.Model(wallet).Update("available_minor", int64(100)).Error)

	funding, replay, err := FundTenantMember(tenant.Id, member.Id, "fund-1", "USD", 40, 400, "1", "owner")
	require.NoError(t, err)
	require.False(t, replay)
	require.Equal(t, int64(40), funding.AmountMinor)
	var updated User
	require.NoError(t, db.First(&updated, member.Id).Error)
	require.Equal(t, 400, updated.Quota)
	funding, replay, err = FundTenantMember(tenant.Id, member.Id, "fund-1", "USD", 40, 400, "1", "owner")
	require.NoError(t, err)
	require.True(t, replay)
	require.Equal(t, int64(1), funding.ID)
	_, _, err = FundTenantMember(tenant.Id, member.Id, "fund-1", "USD", 41, 410, "1", "owner")
	require.ErrorIs(t, err, ErrTenantFundingConflict)
	_, _, err = FundTenantMember(otherTenant.Id, member.Id, "other-1", "USD", 1, 10, "1", "other")
	require.ErrorIs(t, err, ErrTenantFundingForbidden)
	_, _, err = FundTenantMember(tenant.Id, member.Id, "fund-2", "USD", 61, 610, "1", "owner")
	require.ErrorIs(t, err, ErrTenantFundingInsufficient)
	require.NoError(t, db.First(&updated, member.Id).Error)
	require.Equal(t, 400, updated.Quota)
	require.NoError(t, db.First(wallet, wallet.ID).Error)
	require.Equal(t, int64(60), wallet.AvailableMinor)
}
