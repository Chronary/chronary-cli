package redact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"empty", "", ""},
		{"org key", "chr_sk_abcdef1234567890", "chr_sk_...7890"},
		{"agent key", "chr_ak_abcdef1234567890", "chr_ak_...7890"},
		{"legacy org key", "chr_sk_live_abcdef1234567890", "chr_sk_live_...7890"},
		{"legacy agent key", "chr_ak_live_abcdef1234567890", "chr_ak_live_...7890"},
		{"short prefixed key", "chr_sk_ab", "chr_sk_..."},
		{"prefix only", "chr_sk_", "chr_sk_..."},
		{"short unknown", "short", "..."},
		{"long unknown", "longnonprefixedkey123", "...y123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MaskKey(tt.key))
		})
	}
}

func TestMaskKeyNeverRevealsTheSecretBody(t *testing.T) {
	key := "chr_sk_SECRETBODYvalue9876"
	masked := MaskKey(key)
	assert.NotContains(t, masked, "SECRETBODY")
	assert.True(t, strings.HasSuffix(masked, "9876"))
	assert.Less(t, len(masked), len(key))
}
