package resource

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type OutCommand struct {
	gitea  Gitea
	writer io.Writer
}

func NewOutCommand(gitea Gitea, writer io.Writer) *OutCommand {
	return &OutCommand{
		gitea:  gitea,
		writer: writer,
	}
}

func (c *OutCommand) Run(sourceDir string, request OutRequest) (OutResponse, error) {
	params := request.Params

	// name, err := c.fileContents(filepath.Join(sourceDir, request.Params.NamePath))
	// if err != nil {
	// 	return OutResponse{}, err
	// }

	tag_name, err := c.fileContents(filepath.Join(sourceDir, request.Params.TagPath))
	if err != nil {
		return OutResponse{}, err
	}

	tag_name = request.Params.TagPrefix + tag_name

	// if request.Params.BodyPath != "" {
	// 	_, err := c.fileContents(filepath.Join(sourceDir, request.Params.BodyPath))
	// 	if err != nil {
	// 		return OutResponse{}, err
	// 	}
	// }

	release, release_id, err := c.gitea.GetReleaseByTag(tag_name)

	if release == nil {
		_, release_id, err = c.gitea.CreateRelease(tag_name, "Auto-generated from Concourse Gitea Release Resource")
		if err != nil {
			return OutResponse{}, err
		}
	}

	// upload files
	var fileLinks []string
	for _, fileGlob := range params.Globs {
		matches, err := filepath.Glob(filepath.Join(sourceDir, fileGlob))
		if err != nil {
			return OutResponse{}, err
		}

		if len(matches) == 0 {
			return OutResponse{}, fmt.Errorf("could not find file that matches glob '%s'", fileGlob)
		}

		for _, filePath := range matches {
			attachment, err := c.gitea.CreateAttachment(filePath, release_id)
			if err != nil {
				return OutResponse{}, err
			}
			fileLinks = append(fileLinks, attachment.Name)
		}
	}

	// update the release
	_, err = c.gitea.EditRelease(tag_name, release_id, strings.Join(fileLinks, "\n"))
	if err != nil {
		return OutResponse{}, errors.New("could not get saved tag")
	}

	return OutResponse{
		Version:  versionFromTag(release),
		Metadata: metadataFromTag(release),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
