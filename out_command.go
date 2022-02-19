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

	tag_name, err := c.fileContents(filepath.Join(sourceDir, request.Params.TagPath))
	if err != nil {
		return OutResponse{}, err
	}

	tag_name = request.Params.TagPrefix + tag_name

	title := tag_name
	if request.Params.TitlePath != "" {
		title, err = c.fileContents(filepath.Join(sourceDir, request.Params.TitlePath))
		if err != nil {
			return OutResponse{}, err
		}
	}

	body := "Auto-generated from Concourse Gitea Release Resource"
	if request.Params.BodyPath != "" {
		body, err = c.fileContents(filepath.Join(sourceDir, request.Params.TitlePath))
		if err != nil {
			return OutResponse{}, err
		}
	}

	release, err := c.gitea.GetReleaseByTag(tag_name)

	if release == nil {
		release, err = c.gitea.CreateRelease(title, tag_name, body)
		if err != nil {
			return OutResponse{}, err
		}
	}

	// upload files
	for _, fileGlob := range params.Globs {
		matches, err := filepath.Glob(filepath.Join(sourceDir, fileGlob))
		if err != nil {
			return OutResponse{}, err
		}

		if len(matches) == 0 {
			return OutResponse{}, fmt.Errorf("could not find file that matches glob '%s'", fileGlob)
		}

		for _, filePath := range matches {
			_, err := c.gitea.CreateAttachment(filePath, release.ID)
			if err != nil {
				return OutResponse{}, err
			}
		}
	}

	// update the release
	_, err = c.gitea.EditRelease(title, tag_name, release.ID, body)
	if err != nil {
		return OutResponse{}, errors.New("could not get saved tag")
	}

	return OutResponse{
		Version:  versionFromRelease(release),
		Metadata: metadataFromRelease(release),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
