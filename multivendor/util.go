package multivendor

import (
	"regexp"
	"strings"
)

var nonAlphanumericRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumericRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
