package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"chr_sk_abcdef1234567890", "chr_sk_abcd..."},
		{"chr_sk_xyz1234567890", "chr_sk_xyz1..."},
		{"chr_sk_live_abcdef1234567890", "chr_sk_live_abcd..."},
		{"chr_sk_ab", "chr_sk_ab"},
		{"short", "short"},
		{"longnonprefixedkey123", "longnonp..."},
	}

	for _, tt := range tests {
		t.Run(tt.key[:min(10, len(tt.key))], func(t *testing.T) {
			assert.Equal(t, tt.want, maskKey(tt.key))
		})
	}
}
