package client

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/wolfeidau/zipstash/api/zipstash/v1/zipstashv1connect"
)

const audience = "zipstash.wolfe.id.au"

type Globals struct {
	Debug   bool
	Version string
	Client  zipstashv1connect.ZipStashServiceClient
}

func SplitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

type Local struct {
	Branch     string `help:"branch to use for the cache entry" env:"INPUT_BRANCH" and:"local"`
	Repository string `help:"repository to use for the cache entry" env:"INPUT_REPOSITORY" and:"local"`
}

type GitHub struct {
	Branch     string `help:"branch to use for the cache entry" env:"INPUT_BRANCH" and:"github"`
	Repository string `help:"repository to use for the cache entry" env:"INPUT_REPOSITORY" and:"github"`
}

func getRepoAndBranch(github GitHub, local Local) (string, string, error) {
	if github.Repository != "" && github.Branch != "" {
		return github.Repository, github.Branch, nil
	}

	if local.Repository != "" && local.Branch != "" {
		return local.Repository, local.Branch, nil
	}

	return "", "", fmt.Errorf("repository and branch must be provided")
}
