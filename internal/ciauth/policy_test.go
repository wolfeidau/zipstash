package ciauth

import (
	"encoding/json"
	"testing"

	"github.com/cedar-policy/cedar-go"
	"github.com/cedar-policy/cedar-go/types"
	"github.com/stretchr/testify/require"
)

var (
	cedarPolicy = `
permit(
	principal,
	action in [Action::"save", Action::"restore"],
	resource
)
when {
	context.subject like "repo:wolfeidau/*" &&
	context.audience == "zipstash.wolfe.id.au"
};
`
)

func TestPolicies(t *testing.T) {
	assert := require.New(t)

	var policy cedar.Policy
	err := policy.UnmarshalCedar([]byte(cedarPolicy))
	assert.NoError(err)

	ps := cedar.NewPolicySet()
	ps.Add("policy0", &policy)

	entitiesJSON := `
	[
		{
			"id": "wolfeidau/zipstash",
			"type": "Repo"
		},
		{
			"id": "abc123",
			"type": "Cache"
		}
	]
	`
	var entities cedar.EntityMap
	err = json.Unmarshal([]byte(entitiesJSON), &entities)
	assert.NoError(err)

	req := cedar.Request{
		Principal: cedar.NewEntityUID("Repo", "wolfeidau/zipstash"),
		Action:    cedar.NewEntityUID("Action", "save"),
		Resource:  cedar.NewEntityUID("Cache", "abc123"),
		Context: cedar.NewRecord(cedar.RecordMap{
			"subject":  cedar.String("repo:wolfeidau/zipstash:ref:refs/heads/main"),
			"audience": cedar.String("zipstash.wolfe.id.au"),
		}),
	}
	ok, _ := ps.IsAuthorized(entities, req)
	assert.Equal(types.Allow, ok)
}
