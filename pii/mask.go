package pii

import (
	"fmt"
	"net/netip"
	"strings"
)

func maskEmail(s string) string {
	at := strings.Index(s, "@")
	if at <= 0 {
		return "***"
	}
	local := s[:at]
	domain := s[at:]
	if len(local) == 1 {
		return local + "***" + domain
	}
	return local[:1] + strings.Repeat("*", len(local)-1) + domain
}

func maskPhone(s string) string {
	// keep first 3 and last 3 chars, mask middle
	// +919876543210 → +91*******210
	r := []rune(s)
	if len(r) <= 6 {
		return "***"
	}
	masked := make([]rune, len(r))
	copy(masked, r)
	for i := 3; i < len(r)-3; i++ {
		if masked[i] != '+' && masked[i] != '-' && masked[i] != ' ' {
			masked[i] = '*'
		}
	}
	return string(masked)
}

func maskName(s string) string {
	r := []rune(strings.TrimSpace(s))
	switch len(r) {
	case 0:
		return "***"
	case 1:
		return string(r[0]) + "*"
	case 2:
		return string(r[0]) + "*"
	default:
		return string(r[0]) + strings.Repeat("*", len(r)-2) + string(r[len(r)-1])
	}
}

func maskIP(addr netip.Addr) string {
	if !addr.IsValid() {
		return "xxx"
	}
	if addr.Is4() {
		a := addr.As4()
		return fmt.Sprintf("%d.%d.%d.xxx", a[0], a[1], a[2])
	}
	// v6: keep first 4 groups, mask last 4
	s := addr.StringExpanded() // full 8-group form
	groups := strings.Split(s, ":")
	if len(groups) != 8 {
		return "xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx"
	}
	return strings.Join(groups[:4], ":") + ":xxxx:xxxx:xxxx:xxxx"
}

func maskTaxID(s string) string {
	r := []rune(s)
	if len(r) <= 4 {
		return "***"
	}
	// keep first 2 and last 2, mask middle
	return string(r[:2]) + strings.Repeat("*", len(r)-4) + string(r[len(r)-2:])
}
