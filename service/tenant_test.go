package service

import (
	"context"
	"net"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type tenantDNSResolverStub struct {
	ips   []net.IP
	cname string
}

func (stub tenantDNSResolverStub) LookupIP(context.Context, string, string) ([]net.IP, error) {
	return stub.ips, nil
}

func (stub tenantDNSResolverStub) LookupCNAME(context.Context, string) (string, error) {
	return stub.cname, nil
}

func TestVerifyDomainMatchesConfiguredDNSRecords(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Tenant{}, &model.TenantMembership{}, &model.TenantDomain{}))
	previousDB := model.DB
	previousResolver := TenantDomainDNSResolver
	model.DB = db
	TenantDomainDNSResolver = tenantDNSResolverStub{ips: []net.IP{net.ParseIP("203.0.113.10")}, cname: "proxy.example.com."}
	t.Cleanup(func() {
		model.DB = previousDB
		TenantDomainDNSResolver = previousResolver
		common.OptionMapRWMutex.Lock()
		if common.OptionMap == nil {
			common.OptionMap = map[string]string{}
		}
		common.OptionMap["TenantDomainDNSRecordCNAME"] = ""
		common.OptionMap["TenantDomainDNSRecordA"] = ""
		common.OptionMapRWMutex.Unlock()
	})

	owner := model.User{Username: "dns-owner", Password: "unused", AffCode: "dns-owner-code", Status: common.UserStatusEnabled}
	other := model.User{Username: "dns-other", Password: "unused", AffCode: "dns-other-code", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&other).Error)
	first, err := model.CreateTenantWithOwner(db, "dns-first", "DNS First", owner.Id)
	require.NoError(t, err)
	second, err := model.CreateTenantWithOwner(db, "dns-second", "DNS Second", other.Id)
	require.NoError(t, err)
	firstDomain := model.TenantDomain{TenantId: first.Id, Hostname: "app.example.com", Status: model.DomainPending}
	secondDomain := model.TenantDomain{TenantId: second.Id, Hostname: "other.example.com", Status: model.DomainPending}
	require.NoError(t, db.Create(&firstDomain).Error)
	require.NoError(t, db.Create(&secondDomain).Error)

	setTargets := func(cname, ip string) {
		common.OptionMapRWMutex.Lock()
		if common.OptionMap == nil {
			common.OptionMap = map[string]string{}
		}
		common.OptionMap["TenantDomainDNSRecordCNAME"] = cname
		common.OptionMap["TenantDomainDNSRecordA"] = ip
		common.OptionMapRWMutex.Unlock()
	}

	setTargets("", "203.0.113.10")
	require.NoError(t, VerifyDomain(firstDomain.Id, first.Id))
	firstDomain.Status = model.DomainPending
	require.NoError(t, db.Save(&firstDomain).Error)
	setTargets("proxy.example.com", "")
	require.NoError(t, VerifyDomain(firstDomain.Id, first.Id))
	firstDomain.Status = model.DomainPending
	require.NoError(t, db.Save(&firstDomain).Error)

	setTargets("wrong.example.com", "")
	require.EqualError(t, VerifyDomain(firstDomain.Id, first.Id), "domain verification failed")
	setTargets("", "")
	require.EqualError(t, VerifyDomain(firstDomain.Id, first.Id), "domain verification failed")
	require.ErrorIs(t, VerifyDomain(secondDomain.Id, first.Id), gorm.ErrRecordNotFound)
}
