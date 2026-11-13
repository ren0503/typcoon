package auth

import "testing"

func TestRateLimitsWithDefaultsMatchDocumentedMinimums(t *testing.T) {
	defaults := RateLimits{}.withDefaults()
	if defaults.ExchangeIPPerMin != 10 ||
		defaults.ExchangeIdentityPerMin != 30 ||
		defaults.RefreshIPPerMin != 60 ||
		defaults.RefreshSessionPerMin != 30 ||
		defaults.LinkAccountPerMin != 10 ||
		defaults.LogoutAllAccountPerMin != 5 {
		t.Fatalf("documented rate-limit defaults = %+v", defaults)
	}
}
