package errors

import (
	"errors"
	"testing"

	"github.com/linkerlin/gotrycatch/errtypes"
)

// Verifies the compatibility layer forwards to errtypes with identical behavior.

func TestForward_AllTypes(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"ValidationError", NewValidationError("f", "m", 1)},
		{"DatabaseError", NewDatabaseError("SELECT", "t", errors.New("cause"))},
		{"NetworkError", NewNetworkError("http://x", 404)},
		{"NetworkTimeoutError", NewNetworkTimeoutError("http://x")},
		{"BusinessLogicError", NewBusinessLogicError("r", "d")},
		{"ConfigError", NewConfigError("k", "v", "r")},
		{"AuthError", NewAuthError("login", "u", "r")},
		{"RateLimitError", NewRateLimitError("res", 1, 2, 3)},
	}
	for _, c := range cases {
		if c.err.Error() == "" {
			t.Errorf("%s: empty Error()", c.name)
		}
	}
}

func TestForward_IdentityWithErrtypes(t *testing.T) {
	// aliases must be the same types so errors.As works across packages
	err := NewDatabaseError("INSERT", "logs", nil)
	var target errtypes.DatabaseError
	if !errors.As(error(err), &target) {
		t.Error("alias types must be identical to errtypes types")
	}
}
