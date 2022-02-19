package resource

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"

	"context"

	"code.gitea.io/sdk/gitea"
)

type Gitea interface {
	ListTags() ([]*gitea.Tag, error)
	ListTagsUntil(tag_name string) ([]*gitea.Tag, error)
	GetTag(tag_name string) (*gitea.Tag, error)
	CreateTag(tag_name string, ref string) (*gitea.Tag, error)
	GetReleaseByTag(tag_name string) (*gitea.Release, error)
	CreateRelease(tag_name string, description string) (*gitea.Release, error)
	EditRelease(tag_name string, release_id int64, description string) (*gitea.Release, error)
	CreateAttachment(filePath string, release_id int64) (*gitea.Attachment, error)
	GetAttachment(filePath, destPath string) error
}

type GiteaClient struct {
	client      *gitea.Client
	baseUrl     *url.URL
	accessToken string
	user        string
	repository  string
}

func NewGiteaClient(source Source) (*GiteaClient, error) {
	var httpClient = &http.Client{}
	var ctx = context.TODO()

	baseUrl, err := url.Parse(source.GiteaAPIURL)
	if err != nil {
		return nil, err
	}

	if source.Insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	client, err := gitea.NewClient(baseUrl.String(), gitea.SetHTTPClient(httpClient), gitea.SetToken(source.AccessToken))
	if err != nil {
		return nil, err
	}

	split := strings.SplitN(source.Repository, "/", 2)
	return &GiteaClient{
		client:      client,
		baseUrl:     baseUrl,
		user:        split[0],
		repository:  split[1],
		accessToken: source.AccessToken,
	}, nil
}

func (g *GiteaClient) ListTags() ([]*gitea.Tag, error) {
	var allTags []*gitea.Tag

	opt := gitea.ListRepoTagsOptions{
		ListOptions: gitea.ListOptions{
			PageSize: 100,
			Page:     1,
		},
	}

	for {
		tags, _, err := g.client.ListRepoTags(g.user, g.repository, opt)

		if len(tags) == 0 {
			break
		}

		if err != nil {
			return []*gitea.Tag{}, err
		}

		allTags = append(allTags, tags...)

		opt.Page++
	}

	return allTags, nil
}

func (g *GiteaClient) ListTagsUntil(tag_name string) ([]*gitea.Tag, error) {
	var allTags []*gitea.Tag

	pageSize := 100

	opt := gitea.ListRepoTagsOptions{
		ListOptions: gitea.ListOptions{
			PageSize: pageSize,
			Page:     1,
		},
	}

	var foundTag *gitea.Tag
	for {
		tags, _, err := g.client.ListRepoTags(g.user, g.repository, opt)

		if len(tags) == 0 {
			break
		}

		if err != nil {
			return []*gitea.Tag{}, err
		}

		skipToNextPage := false
		for i, tag := range tags {
			if foundTag != nil {
				if foundTag.Commit.Created.Format("2006-01-02") == tag.Commit.Created.Format("2006-01-02") {
					allTags = append(allTags, tag)
					if i == (pageSize - 1) {
						skipToNextPage = true
						break
					} else {
						continue
					}
				} else {
					break
				}
			}

			if tag.Name == tag_name {
				allTags = append(allTags, tags[:i+1]...)
				foundTag = tag
				continue
			}
		}
		if skipToNextPage {
			opt.Page++
			continue
		}

		if foundTag != nil {
			break
		}

		allTags = append(allTags, tags...)

		opt.Page++
	}

	return allTags, nil
}

func (g *GiteaClient) GetTag(tag_name string) (*gitea.Tag, error) {
	tag, res, err := g.client.GetTag(g.user, g.repository, tag_name)
	if err != nil {
		return &gitea.Tag{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (g *GiteaClient) CreateTag(ref string, tag_name string) (*gitea.Tag, error) {
	opt := gitea.CreateTagOption{
		TagName: tag_name,
		Message: tag_name,
		Target:  ref,
	}

	tag, res, err := g.client.CreateTag(g.user, g.repository, opt)
	if err != nil {
		return &gitea.Tag{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (g *GiteaClient) GetReleaseByTag(tag_name string) (*gitea.Release, error) {
	release, res, err := g.client.GetReleaseByTag(g.user, g.repository, tag_name)

	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	if err != nil {
		return &gitea.Release{}, err
	}

	return release, nil
}

func (g *GiteaClient) CreateRelease(tag_name string, description string) (*gitea.Release, error) {
	opt := gitea.CreateReleaseOption{
		Note:    description,
		TagName: tag_name,
		Title:   "cock and balls full HD",
	}

	release, res, err := g.client.CreateRelease(g.user, g.repository, opt)
	if err != nil {
		return &gitea.Release{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusConflict {
		return nil, errors.New("release already exists")
	}

	return release, nil
}

func (g *GiteaClient) EditRelease(tag_name string, release_id int64, description string) (*gitea.Release, error) {

	opt := gitea.EditReleaseOption{
		Note:    description,
		TagName: tag_name,
		Title:   "cock and balls full HD",
	}

	release, res, err := g.client.EditRelease(g.user, g.repository, release_id, opt)
	if err != nil {
		return &gitea.Release{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (g *GiteaClient) CreateAttachment(filePath string, release_id int64) (*gitea.Attachment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return &gitea.Attachment{}, err
	}
	var file io.Reader
	file = f
	attachment, _, err := g.client.CreateReleaseAttachment(g.user, g.repository, release_id, file, filepath.Base(filePath))
	return attachment, err
}

func (g *GiteaClient) GetAttachment(filePath, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	filePathRef, err := url.Parse(filePath)
	if err != nil {
		return err
	}

	projectFileUrl := g.baseUrl.ResolveReference(filePathRef)

	client := &http.Client{}
	req, err := http.NewRequest("GET", projectFileUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "token "+g.accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file `%s`: HTTP status %d", filepath.Base(destPath), resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
