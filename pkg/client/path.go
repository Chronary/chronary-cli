package client

import (
	"errors"
	"net/url"
	"strings"
)

// invalidPathSegment stands in for a refused path parameter. It holds a NUL
// byte, which url.PathEscape never produces, so Client.do can recognise it and
// refuse the request before anything is sent.
const invalidPathSegment = "\x00invalid-path-segment\x00"

// ErrInvalidPathSegment is returned, before any request, when an ID used in a
// URL path is empty, "." or "..".
var ErrInvalidPathSegment = errors.New(`invalid ID: a path parameter must not be empty, "." or "..". No request was sent`)

// PathSegment escapes one caller-supplied ID (an argument or an ID flag) as a
// single URL path segment (#2519). url.PathEscape keeps "/", "?", "#" and "%"
// inside the segment, so an ID like "../agents/agt_1" or "evt_1?expand=all"
// can't reach another resource or add a query string. Escaping leaves dots
// alone, and a "." or ".." segment would be resolved by the edge, so those
// and "" are refused: PathSegment returns a marker and the request fails with
// ErrInvalidPathSegment.
func PathSegment(id string) string {
	if id == "" || id == "." || id == ".." {
		return invalidPathSegment
	}
	return url.PathEscape(id)
}

func hasInvalidPathSegment(path string) bool {
	return strings.Contains(path, invalidPathSegment)
}
