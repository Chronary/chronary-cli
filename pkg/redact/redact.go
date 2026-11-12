// Package redact masks credentials before the CLI prints or logs them.
//
// Every place the CLI shows an API key, other than the one-time display of a
// newly created key, goes through MaskKey: auth status and list, the --debug
// request dump and structured profile output. Terminal scrollback, CI logs
// and pasted bug reports then never carry a usable key.
package redact

import "strings"

// keyPrefixes are the known Chronary key prefixes, longest first so the
// legacy "_live_" forms match before their shorter counterparts.
var keyPrefixes = []string{"chr_sk_live_", "chr_ak_live_", "chr_sk_", "chr_ak_"}

// visibleSuffix is how many trailing characters of a key stay visible.
const visibleSuffix = 4

// minHiddenForSuffix is the shortest secret part that still shows a suffix.
// Anything shorter is hidden entirely, so the suffix is never most of it.
const minHiddenForSuffix = 12

// Ellipsis separates the kept prefix from the visible suffix.
const Ellipsis = "..."

// MaskKey returns a display-safe form of an API key: the key's prefix, an
// ellipsis and its last four characters (for example "chr_sk_...a1b2"). A key
// without a known prefix keeps only its last four characters. Short values
// are hidden entirely. An empty key stays empty.
func MaskKey(key string) string {
	if key == "" {
		return ""
	}
	prefix := ""
	rest := key
	for _, p := range keyPrefixes {
		if strings.HasPrefix(key, p) {
			prefix = p
			rest = key[len(p):]
			break
		}
	}
	if len(rest) < minHiddenForSuffix {
		return prefix + Ellipsis
	}
	return prefix + Ellipsis + rest[len(rest)-visibleSuffix:]
}
