package resource

import "code.gitea.io/sdk/gitea"

func metadataFromRelease(release *gitea.Release) []MetadataPair {
	metadata := []MetadataPair{}

	if release.TagName != "" {
		nameMeta := MetadataPair{
			Name:  "tag",
			Value: release.TagName,
		}

		metadata = append(metadata, nameMeta)
	}

	if release != nil && release.Note != "" {
		metadata = append(metadata, MetadataPair{
			Name:     "body",
			Value:    release.Note,
			Markdown: true,
		})
	}

	if release != nil && release.Target != "" {
		metadata = append(metadata, MetadataPair{
			Name:  "commit_sha",
			Value: release.Target,
		})
	}
	return metadata
}
