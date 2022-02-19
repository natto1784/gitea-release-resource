package resource

import (
	"sort"

	"code.gitea.io/sdk/gitea"
	"github.com/cppforlife/go-semi-semantic/version"
)

type CheckCommand struct {
	gitea Gitea
}

func NewCheckCommand(gitea Gitea) *CheckCommand {
	return &CheckCommand{
		gitea: gitea,
	}
}

func (c *CheckCommand) Run(request CheckRequest) ([]Version, error) {
	var tags []*gitea.Tag
	var err error
	if (request.Version == Version{}) {
		tags, err = c.gitea.ListTags()
	} else {
		tags, err = c.gitea.ListTagsUntil(request.Version.Tag)
	}

	if err != nil {
		return []Version{}, err
	}

	if len(tags) == 0 {
		return []Version{}, nil
	}

	var filteredTags []*gitea.Tag

	// TODO: make ListTagsUntil work better with this
	versionParser, err := newVersionParser(request.Source.TagFilter)
	if err != nil {
		return []Version{}, err
	}

	for _, tag := range tags {
		if _, err := version.NewVersionFromString(versionParser.parse(tag.Name)); err != nil {
			continue
		}

		/*		if tag.Release == nil {
				continue
			}*/

		filteredTags = append(filteredTags, tag)
	}

	sort.Slice(filteredTags, func(i, j int) bool {
		first, err := version.NewVersionFromString(versionParser.parse(filteredTags[i].Name))
		if err != nil {
			return true
		}

		second, err := version.NewVersionFromString(versionParser.parse(filteredTags[j].Name))
		if err != nil {
			return false
		}

		return first.IsLt(second)
	})

	if len(filteredTags) == 0 {
		return []Version{}, nil
	}
	latestTag := filteredTags[len(filteredTags)-1]
	latestResource, err := c.gitea.GetReleaseByTag(latestTag.Name)
	if err != nil {
		return []Version{}, err
	}

	if (request.Version == Version{}) {
		return []Version{versionFromTag(latestResource)}, nil
	}

	if latestTag.Name == request.Version.Tag {
		return []Version{versionFromTag(latestResource)}, nil
	}

	upToLatest := false
	nextVersions := []Version{} // contains the requested version and all newer ones

	for _, release := range filteredTags {
		if !upToLatest {
			version := release.Name
			upToLatest = request.Version.Tag == version
		}

		if upToLatest {
			nextVersions = append(nextVersions, Version{Tag: release.Name})
		}
	}

	if !upToLatest {
		// current version was removed; start over from latest
		resource, err := c.gitea.GetReleaseByTag(filteredTags[len(filteredTags)-1].Name)
		if err != nil {
			return []Version{}, err
		}
		nextVersions = append(
			nextVersions,
			versionFromTag(resource),
		)
	}

	return nextVersions, nil
}
