package provider

const (
	Buildkite     = "buildkite"
	GitHubActions = "github_actions"
	GitLab        = "gitlab"
)

var (
	DefaultEndpoints = map[string]string{
		GitHubActions: "https://token.actions.githubusercontent.com",
		GitLab:        "https://gitlab.com",
		Buildkite:     "https://agent.buildkite.com",
	}

	DefaultProviders = []string{
		GitHubActions,
		GitLab,
		Buildkite,
	}
)
