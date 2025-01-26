package ciauth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	Unspecified   = "unspecified"
	Buildkite     = "buildkite"
	GitHubActions = "github_actions"
	GitLab        = "gitlab"
)

var (
	ErrInvalidProvider   = errors.New("invalid provider")
	DefaultOIDCProviders = map[string]OIDCProvider{
		GitHubActions: {
			Issuer:  "https://token.actions.githubusercontent.com",
			JWKSURL: "https://token.actions.githubusercontent.com/.well-known/jwks",
		},
		GitLab: {
			Issuer:  "https://gitlab.com",
			JWKSURL: "https://gitlab.com/oauth/discovery/keys",
		},
		Buildkite: {
			Issuer:  "https://agent.buildkite.com",
			JWKSURL: "https://agent.buildkite.com/.well-known/jwks",
		},
	}

	DefaultProviderNames = []string{
		GitHubActions,
		GitLab,
		Buildkite,
	}
)

type OIDCProvider struct {
	Issuer  string
	JWKSURL string
}

// BuildkiteClaims is the struct for the claims in the Buildkite OIDC token
type BuildkiteClaims struct {
	OrganizationSlug  string `json:"organization_slug"`
	PipelineSlug      string `json:"pipeline_slug"`
	BuildBranch       string `json:"build_branch"`
	BuildTag          string `json:"build_tag"`
	BuildCommit       string `json:"build_commit"`
	StepKey           string `json:"step_key"`
	JobId             string `json:"job_id"`
	AgentId           string `json:"agent_id"`
	BuildSource       string `json:"build_source"`
	RunnerEnvironment string `json:"runner_environment"`
	BuildNumber       int    `json:"build_number"`
}

type GitHubActionsClaims struct {
	Ref                  string `json:"ref"`
	Sha                  string `json:"sha"`
	Repository           string `json:"repository"`
	RepositoryOwner      string `json:"repository_owner"`
	RepositoryOwnerID    string `json:"repository_owner_id"`
	RunId                string `json:"run_id"`
	RunNumber            string `json:"run_number"`
	RunAttempt           string `json:"run_attempt"`
	RepositoryVisibility string `json:"repository_visibility"`
	RepositoryID         string `json:"repository_id"`
	ActorId              string `json:"actor_id"`
	Actor                string `json:"actor"`
	Workflow             string `json:"workflow"`
	HeadRef              string `json:"head_ref"`
	BaseRef              string `json:"base_ref"`
	EventName            string `json:"event_name"`
	RefProtected         string `json:"ref_protected"`
	RefType              string `json:"ref_type"`
	WorkflowRef          string `json:"workflow_ref"`
	WorkflowSha          string `json:"workflow_sha"`
	JobWorkflowRef       string `json:"job_workflow_ref"`
	JobWorkflowSha       string `json:"job_workflow_sha"`
	RunnerEnvironment    string `json:"runner_environment"`
}

func extractBearerToken(header http.Header) (string, error) {
	reqToken := header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer")
	if len(splitToken) != 2 {
		return "", fmt.Errorf("malformed token")
	}

	return strings.TrimSpace(splitToken[1]), nil
}
