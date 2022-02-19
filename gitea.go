package resource

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"

	"context"

	"code.gitea.io/sdk/gitea"
)


type Gitea interface {
	ListTags() ([]*gitea.Tag, error)
	ListTagsUntil(tag_name string) ([]*gitea.Tag, error)
	GetTag(tag_name string) (*gitea.Tag, error)
	CreateTag(tag_name string, ref string) (*gitea.Tag, error)
	CreateRelease(tag_name string, description string) (*gitea.Release, error)
	UpdateRelease(tag_name string, description string) (*gitea.Release, error)
}

type GiteaClient struct {
	client *gitea.Client

	accessToken string
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

	return &GiteaClient{
		client:      client,
		repository:  source.Repository,
		accessToken: source.AccessToken,
	}, nil
}

func (g *GiteaClient) ListTags() ([]*gitea.Tag, error) {
	var allTags []*gitea.Tag

	opt := &gitea.ListTagsOptions{
		ListOptions: gitea.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		OrderBy: gitea.String("updated"),
		Sort:    gitea.String("desc"),
	}

	for {
		tags, res, err := g.client.Tags.ListTags(g.repository, opt)
		if err != nil {
			return []*gitea.Tag{}, err
		}

		allTags = append(allTags, tags...)

		if opt.Page >= res.TotalPages {
			break
		}

		opt.Page = res.NextPage
	}

	return allTags, nil
}

func (g *GiteaClient) ListTagsUntil(tag_name string) ([]*gitea.Tag, error) {
	var allTags []*gitea.Tag

	pageSize := 100

	opt := &gitea.ListTagsOptions{
		ListOptions: gitea.ListOptions{
			PerPage: pageSize,
			Page:    1,
		},
		OrderBy: gitea.String("updated"),
		Sort:    gitea.String("desc"),
	}

	var foundTag *gitea.Tag
	for {
		tags, res, err := g.client.Tags.ListTags(g.repository, opt)
		if err != nil {
			return []*gitea.Tag{}, err
		}

		skipToNextPage := false
		for i, tag := range tags {
			// Some tags might have the same date - if they all have the same date, take
			// all of them
			if foundTag != nil {
				if foundTag.Commit.CommittedDate.Equal(*tag.Commit.CommittedDate) {
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
			if opt.Page >= res.TotalPages {
				break
			}

			opt.Page = res.NextPage
			continue
		}

		if foundTag != nil {
			break
		}

		allTags = append(allTags, tags...)

		if opt.Page >= res.TotalPages {
			break
		}

		opt.Page = res.NextPage
	}

	return allTags, nil
}

func (g *GiteaClient) GetTag(tag_name string) (*gitea.Tag, error) {
	tag, res, err := g.client.Tags.GetTag(g.repository, tag_name)
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
	opt := &gitea.CreateTagOptions{
		TagName: gitea.String(tag_name),
		Ref:     gitea.String(ref),
		Message: gitea.String(tag_name),
	}

	tag, res, err := g.client.Tags.CreateTag(g.repository, opt)
	if err != nil {
		return &gitea.Tag{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (g *GiteaClient) CreateRelease(tag_name string, description string) (*gitea.Release, error) {
	opt := &gitea.CreateReleaseOptions{
		Description: gitea.String(description),
	}

	release, res, err := g.client.Tags.CreateRelease(g.repository, tag_name, opt)
	if err != nil {
		return &gitea.Release{}, err
	}

	// https://docs.gitea.com/ce/api/tags.html#create-a-new-release
	// returns 409 if release already exists
	if res.StatusCode == http.StatusConflict {
		return nil, errors.New("release already exists")
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (g *GiteaClient) UpdateRelease(tag_name string, description string) (*gitea.Release, error) {
	opt := &gitea.UpdateReleaseOptions{
		Description: gitea.String(description),
	}

	release, res, err := g.client.Tags.UpdateRelease(g.repository, tag_name, opt)
	if err != nil {
		return &gitea.Release{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}
