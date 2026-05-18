package leash

import (
	"testing"

	"github.com/leash-build/leash-sdk-go/integrations"
)

// TestAliases_ResolveToSubpackageTypes proves the README example compiles —
// `leash.GmailListParams{...}` MUST be the same type as
// `integrations.GmailListParams`.
func TestAliases_ResolveToSubpackageTypes(t *testing.T) {
	// Type identity at compile time: assigning to a value of the subpackage
	// type would not compile if the alias were a distinct type.
	var p integrations.GmailListParams = GmailListParams{MaxResults: 5}
	if p.MaxResults != 5 {
		t.Errorf("alias did not preserve field")
	}

	var f integrations.LinearListIssuesFilter = LinearListIssuesFilter{
		StateType: LinearStateStarted,
	}
	if f.StateType != integrations.LinearStateStarted {
		t.Errorf("state-type alias did not preserve value")
	}
}
