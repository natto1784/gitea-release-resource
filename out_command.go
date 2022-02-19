package resource

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"code.gitea.io/sdk/gitea"
)

type OutCommand struct {
	gitea Gitea
	writer io.Writer
}

func NewOutCommand(gitea Gitea, writer io.Writer) *OutCommand {
	return &OutCommand{
		gitea: gitea,
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

	tagExists := true
	tag, err := c.gitea.GetTag(tag_name)
	if err != nil {
		//TODO: improve the check to be based on the specific error
		tagExists = false
	}

	// create the tag first, as next sections assume the tag exists
	if !tagExists {
		targetCommitish, err := c.fileContents(filepath.Join(sourceDir, request.Params.CommitishPath))
		if err != nil {
			return OutResponse{}, err
		}
		tag, err = c.gitea.CreateTag(targetCommitish, tag_name)
		if err != nil {
			return OutResponse{}, err
		}
	}

	// create a new release if it doesn't exist yet
	if tag.Release == nil {
		_, err = c.gitea.CreateRelease(tag_name, "Auto-generated from Concourse Gitea Release Resource")
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
			projectFile, err := gitea.CreateReleaseAttachment(filePath)
			if err != nil {
				return OutResponse{}, err
			}
			fileLinks = append(fileLinks, projectFile.Markdown)
		}
	}

	// update the release
	_, err = c.gitea.UpdateRelease(tag_name, strings.Join(fileLinks, "\n"))
	if err != nil {
		return OutResponse{}, errors.New("could not get saved tag")
	}

	return OutResponse{
		Version:  versionFromTag(tag),
		Metadata: metadataFromTag(tag),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
