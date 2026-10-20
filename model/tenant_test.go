package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
