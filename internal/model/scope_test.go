package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopeRoles(t *testing.T) {
	tcs := map[string]struct {
		scope       Scope
		wantAdmin   bool
		wantAnalyst bool
		wantViewer  bool
	}{
		"admin":   {scope: Scope{Role: RoleAdmin}, wantAdmin: true},
		"analyst": {scope: Scope{Role: RoleAnalyst}, wantAnalyst: true},
		"viewer":  {scope: Scope{Role: RoleViewer}, wantViewer: true},
		"unknown": {scope: Scope{Role: "OTHER"}},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.wantAdmin, tc.scope.IsAdmin())
			require.Equal(t, tc.wantAnalyst, tc.scope.IsAnalyst())
			require.Equal(t, tc.wantViewer, tc.scope.IsViewer())
		})
	}
}
