package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTenantDomainDNSOption(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr string
	}{
		{name: "valid cname hostname", key: "TenantDomainDNSRecordCNAME", value: "proxy.example.com"},
		{name: "valid ipv4", key: "TenantDomainDNSRecordA", value: "203.0.113.10"},
		{name: "valid ipv6", key: "TenantDomainDNSRecordA", value: "2001:db8::10"},
		{name: "cname rejects ip", key: "TenantDomainDNSRecordCNAME", value: "203.0.113.10", wantErr: "CNAME target"},
		{name: "cname rejects url", key: "TenantDomainDNSRecordCNAME", value: "https://proxy.example.com", wantErr: "CNAME target"},
		{name: "a record rejects hostname", key: "TenantDomainDNSRecordA", value: "proxy.example.com", wantErr: "A record IP"},
		{name: "empty values are rejected", key: "TenantDomainDNSRecordA", value: " ", wantErr: "required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTenantDomainDNSOption(tt.key, tt.value)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
