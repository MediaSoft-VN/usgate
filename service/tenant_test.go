package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
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

func TestTenantDomainLifecycleRunsHostProvisioning(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Tenant{}, &model.TenantMembership{}, &model.TenantDomain{}))
	previousDB := model.DB
	previousResolver := TenantDomainDNSResolver
	previousHostOperation := TenantDomainHostOperation
	model.DB = db
	TenantDomainDNSResolver = tenantDNSResolverStub{ips: []net.IP{net.ParseIP("203.0.113.10")}}
	var operations []string
	failCheck := false
	failProvision := true
	failCleanup := false
	TenantDomainHostOperation = func(operation, hostname string) error {
		operations = append(operations, operation+":"+hostname)
		if operation == "check" && failCheck {
			return errors.New("hostname already exists in nginx")
		}
		if operation == "cleanup" && failCleanup {
			return errors.New("host cleanup failure")
		}
		if operation == "provision" && failProvision {
			failProvision = false
			return errors.New("host command failure")
		}
		return nil
	}
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	common.OptionMap["TenantDomainDNSRecordCNAME"] = ""
	common.OptionMap["TenantDomainDNSRecordA"] = "203.0.113.10"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		model.DB = previousDB
		TenantDomainDNSResolver = previousResolver
		TenantDomainHostOperation = previousHostOperation
		common.OptionMapRWMutex.Lock()
		common.OptionMap["TenantDomainDNSRecordCNAME"] = ""
		common.OptionMap["TenantDomainDNSRecordA"] = ""
		common.OptionMapRWMutex.Unlock()
	})

	owner := model.User{Username: "host-domain-owner", Password: "unused", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	tenant, err := model.CreateTenantWithOwner(db, "host-domain-tenant", "Host Domain Tenant", owner.Id)
	require.NoError(t, err)
	domain := model.TenantDomain{TenantId: tenant.Id, Hostname: "old.example.com", Status: model.DomainVerified}
	require.NoError(t, db.Create(&domain).Error)

	failCleanup = true
	_, err = UpdateDomain(domain.Id, tenant.Id, owner.Id, "new.example.com")
	require.ErrorIs(t, err, ErrTenantDomainHostOperation)
	var stored model.TenantDomain
	require.NoError(t, db.First(&stored, domain.Id).Error)
	require.Equal(t, "old.example.com", stored.Hostname)
	failCleanup = false
	updated, err := UpdateDomain(domain.Id, tenant.Id, owner.Id, "new.example.com")
	require.NoError(t, err)
	require.Equal(t, model.DomainPending, updated.Status)
	require.Equal(t, "cleanup:old.example.com", operations[1])
	matchingResolver := TenantDomainDNSResolver
	TenantDomainDNSResolver = tenantDNSResolverStub{}
	failCheck = true
	require.ErrorIs(t, VerifyDomain(domain.Id, tenant.Id), ErrTenantDomainHostOperation)
	require.NoError(t, db.First(&stored, domain.Id).Error)
	require.Equal(t, model.DomainPending, stored.Status)
	failCheck = false
	TenantDomainDNSResolver = matchingResolver
	require.EqualError(t, VerifyDomain(domain.Id, tenant.Id), ErrTenantDomainHostOperation.Error())
	require.NoError(t, db.First(&stored, domain.Id).Error)
	require.Equal(t, model.DomainVerified, stored.Status)
	require.NoError(t, VerifyDomain(domain.Id, tenant.Id))
	failCleanup = true
	require.ErrorIs(t, DeleteDomain(domain.Id, tenant.Id), ErrTenantDomainHostOperation)
	require.NoError(t, db.First(&stored, domain.Id).Error)
	failCleanup = false
	require.NoError(t, DeleteDomain(domain.Id, tenant.Id))
	require.Equal(t, []string{
		"cleanup:old.example.com",
		"cleanup:old.example.com",
		"check:new.example.com",
		"check:new.example.com",
		"provision:new.example.com",
		"check:new.example.com",
		"provision:new.example.com",
		"cleanup:new.example.com",
		"cleanup:new.example.com",
	}, operations)
	require.ErrorIs(t, db.First(&stored, domain.Id).Error, gorm.ErrRecordNotFound)
}

func TestOnlySystemAdminsCanUseUsGateDomains(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Tenant{}, &model.TenantMembership{}, &model.TenantDomain{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	owner := model.User{Username: "reserved-domain-owner", Password: "unused", AffCode: "reserved-domain-owner-code", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
	require.NoError(t, db.Create(&owner).Error)
	tenant, err := model.CreateTenantWithOwner(db, "reserved-domain-tenant", "Reserved Domain Tenant", owner.Id)
	require.NoError(t, err)
	domain := model.TenantDomain{TenantId: tenant.Id, Hostname: "app.example.com", Status: model.DomainPending}
	require.NoError(t, db.Create(&domain).Error)

	for _, hostname := range []string{"usgate.ai", "Api.UsGate.Ai."} {
		_, err = CreateDomain(tenant.Id, owner.Id, hostname)
		require.ErrorIs(t, err, ErrTenantDomainReserved)
		_, err = UpdateDomain(domain.Id, tenant.Id, owner.Id, hostname)
		require.ErrorIs(t, err, ErrTenantDomainReserved)
	}
	var unchanged model.TenantDomain
	require.NoError(t, db.First(&unchanged, domain.Id).Error)
	require.Equal(t, "app.example.com", unchanged.Hostname)

	admin := model.User{Username: "reserved-domain-admin", Password: "unused", AffCode: "reserved-domain-admin-code", Status: common.UserStatusEnabled, Role: common.RoleAdminUser}
	require.NoError(t, db.Create(&admin).Error)
	previousServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://platform.example.com/"
	t.Cleanup(func() { system_setting.ServerAddress = previousServerAddress })
	_, err = CreateDomain(tenant.Id, admin.Id, "PLATFORM.EXAMPLE.COM.")
	require.ErrorIs(t, err, ErrTenantDomainReserved)
	for _, hostname := range []string{"usgate.ai", "api.usgate.ai"} {
		created, createErr := CreateDomain(tenant.Id, admin.Id, hostname)
		require.NoError(t, createErr)
		require.Equal(t, hostname, created.Hostname)
	}
}

func TestTenantDomainHostWrapperProvisionAndCleanupAreScoped(t *testing.T) {
	root := t.TempDir()
	siteDir := filepath.Join(root, "sites-available")
	enabledDir := filepath.Join(root, "sites-enabled")
	binDir := filepath.Join(root, "bin")
	certState := filepath.Join(root, "certbot-state")
	certRenewalDir := filepath.Join(root, "renewal")
	certLiveDir := filepath.Join(root, "live")
	certManifestDir := filepath.Join(root, "manifests")
	acmeWebrootDir := filepath.Join(root, "webroot")
	effectiveConfig := filepath.Join(root, "nginx-config-dump")
	for _, dir := range []string{siteDir, enabledDir, binDir, certRenewalDir, acmeWebrootDir} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	require.NoError(t, os.WriteFile(effectiveConfig, nil, 0o600))
	config := fmt.Sprintf("APP_PORT=%q\nCERTBOT_EMAIL=%q\nNGINX_SITES_AVAILABLE=%q\nNGINX_SITES_ENABLED=%q\nDOMAIN_LOCK_FILE=%q\nCERT_RENEWAL_DIR=%q\nCERT_LIVE_DIR=%q\nCERT_MANIFEST_DIR=%q\nACME_WEBROOT_DIR=%q\n", "3000", "ops@example.com", siteDir, enabledDir, filepath.Join(root, "lock"), certRenewalDir, certLiveDir, certManifestDir, acmeWebrootDir)
	configPath := filepath.Join(root, "provisioner.env")
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))
	fakeCommands := map[string]string{
		"nginx":     "#!/bin/sh\nif [ \"$1\" = -T ]; then cat \"$NGINX_CONFIG_DUMP\"; exit $?; fi\nif [ \"$1\" = -t ] && [ -f \"$FAIL_NGINX_TEST_FILE\" ]; then exit 1; fi\nexit 0\n",
		"systemctl": "#!/bin/sh\nexit 0\n",
		"curl":      "#!/bin/sh\nexit 0\n",
		"certbot":   "#!/bin/sh\ncase \"$1\" in\n  certonly)\n    if [ -f \"$FAIL_CERTBOT_FILE\" ]; then exit 1; fi\n    mkdir -p \"$CERT_RENEWAL_DIR\" \"$CERT_LIVE_DIR/partner-app.example.com\"\n    printf 'authenticator = webroot\\n' > \"$CERT_RENEWAL_DIR/partner-app.example.com.conf\"\n    touch \"$CERT_STATE\" \"$CERT_LIVE_DIR/partner-app.example.com/cert.pem\"\n    ;;\n  delete)\n    if [ -f \"$FAIL_CERTBOT_DELETE_FILE\" ]; then exit 1; fi\n    rm -f \"$CERT_STATE\" \"$CERT_RENEWAL_DIR/partner-app.example.com.conf\" \"$CERT_LIVE_DIR/partner-app.example.com/cert.pem\"\n    ;;\n  *) exit 2 ;;\nesac\n",
		"openssl":   "#!/bin/sh\nprintf 'X509v3 Subject Alternative Name:\\n    %s\\n' \"$CERT_SAN\"\n",
	}
	for name, contents := range fakeCommands {
		path := filepath.Join(binDir, name)
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o755))
	}

	primaryConfig := filepath.Join(siteDir, "new-api.conf")
	primaryContents := []byte("primary-site-untouched\n")
	require.NoError(t, os.WriteFile(primaryConfig, primaryContents, 0o644))
	otherConfig := filepath.Join(siteDir, "partner-other.example.com.conf")
	require.NoError(t, os.WriteFile(otherConfig, []byte("# Managed by new-api tenant domain provisioner\nserver_name other.example.com;\n"), 0o644))
	otherLink := filepath.Join(enabledDir, filepath.Base(otherConfig))
	require.NoError(t, os.Symlink(otherConfig, otherLink))
	failNginxTestFile := filepath.Join(root, "fail-nginx-test")
	failCertbotFile := filepath.Join(root, "fail-certbot")
	failCertbotDeleteFile := filepath.Join(root, "fail-certbot-delete")

	helper := filepath.Join("..", "scripts", "tenant-domain-host.sh")
	run := func(operation string, wantFailure bool) {
		t.Helper()
		cmd := exec.Command("bash", helper, operation, "app.example.com")
		cmd.Env = append(os.Environ(),
			"TENANT_DOMAIN_PROVISIONER_CONFIG="+configPath,
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"CERT_STATE="+certState,
			"FAIL_NGINX_TEST_FILE="+failNginxTestFile,
			"FAIL_CERTBOT_FILE="+failCertbotFile,
			"FAIL_CERTBOT_DELETE_FILE="+failCertbotDeleteFile,
			"NGINX_CONFIG_DUMP="+effectiveConfig,
			"CERT_RENEWAL_DIR="+certRenewalDir,
			"CERT_LIVE_DIR="+certLiveDir,
			"CERT_SAN=DNS:app.example.com",
		)
		output, err := cmd.CombinedOutput()
		if wantFailure {
			require.Error(t, err, string(output))
			return
		}
		require.NoError(t, err, string(output))
	}

	require.NoError(t, os.WriteFile(failCertbotFile, nil, 0o600))
	run("check", false)
	require.NoError(t, os.WriteFile(effectiveConfig, []byte("# configuration file /etc/nginx/sites-enabled/new-api.conf:\nserver {\n    server_name app.example.com;\n}\n"), 0o600))
	run("check", true)
	require.NoFileExists(t, filepath.Join(siteDir, "partner-app.example.com.conf"))
	run("provision", true)
	require.NoFileExists(t, filepath.Join(siteDir, "partner-app.example.com.conf"))
	require.NoError(t, os.WriteFile(effectiveConfig, nil, 0o600))
	run("provision", true)
	partnerConfig := filepath.Join(siteDir, "partner-app.example.com.conf")
	partnerLink := filepath.Join(enabledDir, filepath.Base(partnerConfig))
	require.NoFileExists(t, partnerConfig)
	require.NoFileExists(t, partnerLink)
	require.NoFileExists(t, certState)
	manifest := filepath.Join(certManifestDir, "partner-app.example.com")
	require.NoFileExists(t, manifest)
	require.NoError(t, os.Remove(failCertbotFile))
	run("provision", false)
	contents, err := os.ReadFile(partnerConfig)
	require.NoError(t, err)
	require.Contains(t, string(contents), "server_name app.example.com;")
	require.Contains(t, string(contents), "proxy_set_header Host $host;")
	require.Contains(t, string(contents), "listen 443 ssl;")
	require.Contains(t, string(contents), "/etc/letsencrypt/live/partner-app.example.com/fullchain.pem")
	require.Contains(t, string(contents), "root "+acmeWebrootDir+";")
	target, err := os.Readlink(partnerLink)
	require.NoError(t, err)
	require.Equal(t, partnerConfig, target)
	require.FileExists(t, certState)
	require.FileExists(t, filepath.Join(certLiveDir, "partner-app.example.com", "cert.pem"))
	require.NoError(t, os.WriteFile(effectiveConfig, []byte(fmt.Sprintf("# configuration file %s:\nserver {\n    server_name app.example.com;\n}\n", partnerLink)), 0o600))
	run("check", false)
	require.NoError(t, os.WriteFile(effectiveConfig, nil, 0o600))

	wrongCertificate := exec.Command("bash", helper, "cleanup", "app.example.com")
	wrongCertificate.Env = append(os.Environ(),
		"TENANT_DOMAIN_PROVISIONER_CONFIG="+configPath,
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"CERT_STATE="+certState,
		"CERT_SAN=DNS:app.example.com,DNS:other.example.com",
		"CERT_RENEWAL_DIR="+certRenewalDir,
		"CERT_LIVE_DIR="+certLiveDir,
	)
	output, err := wrongCertificate.CombinedOutput()
	require.Error(t, err, string(output))
	require.FileExists(t, partnerConfig)
	require.FileExists(t, certState)

	require.NoError(t, os.WriteFile(failCertbotDeleteFile, nil, 0o600))
	run("cleanup", true)
	require.FileExists(t, partnerConfig)
	require.FileExists(t, partnerLink)
	require.FileExists(t, certState)
	require.NoError(t, os.Remove(failCertbotDeleteFile))

	require.NoError(t, os.WriteFile(failNginxTestFile, nil, 0o600))
	run("cleanup", true)
	require.FileExists(t, partnerConfig)
	require.FileExists(t, partnerLink)
	require.FileExists(t, certState)
	require.NoError(t, os.Remove(failNginxTestFile))
	run("cleanup", false)
	require.NoFileExists(t, partnerConfig)
	require.NoFileExists(t, partnerLink)
	require.NoFileExists(t, certState)
	require.NoFileExists(t, filepath.Join(certRenewalDir, "partner-app.example.com.conf"))
	require.NoFileExists(t, manifest)
	require.FileExists(t, primaryConfig)
	require.FileExists(t, otherConfig)
	require.FileExists(t, otherLink)
	primaryAfter, err := os.ReadFile(primaryConfig)
	require.NoError(t, err)
	require.Equal(t, primaryContents, primaryAfter)
}

func TestTenantDomainHostOperationUsesRestrictedSocketProtocol(t *testing.T) {
	previousSocketPath := tenantDomainProvisionerSocketPath
	t.Cleanup(func() { tenantDomainProvisionerSocketPath = previousSocketPath })
	t.Setenv("TENANT_DOMAIN_PROVISIONER_ENABLED", "true")

	for _, testCase := range []struct {
		name      string
		response  string
		wantError bool
	}{
		{name: "success", response: "OK\n"},
		{name: "host operation failure", response: "ERROR\n", wantError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			listener, err := net.Listen("unix", filepath.Join(t.TempDir(), "provisioner.sock"))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, listener.Close()) })
			tenantDomainProvisionerSocketPath = listener.Addr().String()
			requestReceived := make(chan string, 1)
			go func() {
				connection, acceptErr := listener.Accept()
				if acceptErr != nil {
					requestReceived <- "accept error"
					return
				}
				defer connection.Close()
				request, _ := bufio.NewReader(connection).ReadString('\n')
				_, _ = connection.Write([]byte(testCase.response))
				requestReceived <- request
			}()

			err = runTenantDomainHostOperation("cleanup", "app.example.com")
			if testCase.wantError {
				require.ErrorIs(t, err, ErrTenantDomainHostOperation)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, "cleanup\tapp.example.com\n", <-requestReceived)
		})
	}
}
