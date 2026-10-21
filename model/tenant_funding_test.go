package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFundTenantMemberIsAtomicScopedAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:tenant-funding-test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Tenant{}, &TenantMembership{}, &WalletAccount{}, &TenantFunding{}))
	previous := DB
	DB = db
	t.Cleanup(func() { DB = previous })
	owner := User{Username: "tenant-funding-owner", Password: "unused", AffCode: "tenant-funding-owner", Status: common.UserStatusEnabled, Quota: 1000}
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
	funding, replay, err := FundTenantMember(tenant.Id, member.Id, "fund-1", "USD", 40, 400, "1", "owner")
	require.NoError(t, err)
	require.False(t, replay)
	require.Equal(t, int64(40), funding.AmountMinor)
	var updated User
	require.NoError(t, db.First(&updated, member.Id).Error)
	require.Equal(t, 400, updated.Quota)
	updated = User{}
	require.NoError(t, db.First(&updated, owner.Id).Error)
	require.Equal(t, 600, updated.Quota)
	require.Equal(t, owner.Id, funding.OwnerUserID)
	wallet, err := GetTenantFundingWallet(tenant.Id, "USD")
	require.NoError(t, err)
	require.Equal(t, owner.Id, wallet.OwnerUserID)
	require.Equal(t, 600, wallet.AvailableQuota)
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
	updated = User{}
	require.NoError(t, db.First(&updated, member.Id).Error)
	require.Equal(t, 400, updated.Quota)
	updated = User{}
	require.NoError(t, db.First(&updated, owner.Id).Error)
	require.Equal(t, 600, updated.Quota)
	var organizationWallet WalletAccount
	require.ErrorIs(t, db.Where("owner_type = ? AND owner_id = ?", FinancialOwnerOrganization, tenant.Id).First(&organizationWallet).Error, gorm.ErrRecordNotFound)
	_, _, err = FundTenantMember(tenant.Id, owner.Id, "fund-self", "USD", 1, 1, "1", "owner")
	require.ErrorIs(t, err, ErrTenantFundingSelf)
}

func TestFundTenantMemberDoesNotOverspendOwnerQuotaConcurrently(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:tenant-funding-concurrency?mode=memory&cache=shared&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Tenant{}, &TenantMembership{}, &TenantFunding{}))
	previous := DB
	DB = db
	t.Cleanup(func() { DB = previous })
	owner := User{Username: "tenant-funding-concurrent-owner", Password: "unused", Status: common.UserStatusEnabled, Quota: 400}
	memberOne := User{Username: "tenant-funding-concurrent-one", Password: "unused", AffCode: "tenant-funding-concurrent-one", Status: common.UserStatusEnabled}
	memberTwo := User{Username: "tenant-funding-concurrent-two", Password: "unused", AffCode: "tenant-funding-concurrent-two", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&memberOne).Error)
	require.NoError(t, db.Create(&memberTwo).Error)
	tenant, err := CreateTenantWithOwner(db, "concurrent-funding-tenant", "Concurrent Funding Tenant", owner.Id)
	require.NoError(t, err)
	require.NoError(t, db.Create(&TenantMembership{TenantId: tenant.Id, UserId: memberOne.Id, RoleKey: TenantRoleMember, Status: MembershipActive}).Error)
	require.NoError(t, db.Create(&TenantMembership{TenantId: tenant.Id, UserId: memberTwo.Id, RoleKey: TenantRoleMember, Status: MembershipActive}).Error)

	type result struct {
		err error
	}
	results := make(chan result, 2)
	var wait sync.WaitGroup
	for _, member := range []User{memberOne, memberTwo} {
		wait.Add(1)
		go func(member User) {
			defer wait.Done()
			_, _, err := FundTenantMember(tenant.Id, member.Id, "concurrent-"+member.Username, "USD", 30, 300, "1", "owner")
			results <- result{err: err}
		}(member)
	}
	wait.Wait()
	close(results)
	successes := 0
	for result := range results {
		if result.err == nil {
			successes++
		}
	}
	require.Equal(t, 1, successes)
	var updated User
	require.NoError(t, db.First(&updated, owner.Id).Error)
	require.Equal(t, 100, updated.Quota)
}
