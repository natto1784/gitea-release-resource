package resource_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"code.gitea.io/sdk/gitea"
)

func TestGithubReleaseResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gitea Release Resource Suite")
}

func newTag(name, sha string) *gitea.Tag {
	return &gitea.Tag{
		Commit: &gitea.Commit{
			ID: *gitea.String(sha),
		},
		Name: *gitea.String(name),
	}
}
