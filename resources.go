package resource

type Source struct {
	Repository string `json:"repository"`

	GiteaAPIURL string `json:"gitea_api_url"`
	AccessToken string `json:"access_token"`
	Insecure    bool   `json:"insecure"`

	TagFilter string `json:"tag_filter"`
}

type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

func NewCheckRequest() CheckRequest {
	res := CheckRequest{}
	return res
}

func NewOutRequest() OutRequest {
	res := OutRequest{}
	return res
}

func NewInRequest() InRequest {
	res := InRequest{}
	return res
}

type InRequest struct {
	Source  Source   `json:"source"`
	Version *Version `json:"version"`
	Params  InParams `json:"params"`
}

type InParams struct {
	Globs                []string `json:"globs"`
	IncludeSourceTarball bool     `json:"include_source_tarball"`
	IncludeSourceZip     bool     `json:"include_source_zip"`
}

type InResponse struct {
	Version  Version        `json:"version"`
	Metadata []MetadataPair `json:"metadata"`
}

type OutRequest struct {
	Source Source    `json:"source"`
	Params OutParams `json:"params"`
}

type OutParams struct {
	NamePath      string `json:"name"`
	BodyPath      string `json:"body"`
	TagPath       string `json:"tag"`
	TitlePath     string `json:"title"`
	CommitishPath string `json:"commitish"`
	TagPrefix     string `json:"tag_prefix"`

	Globs []string `json:"globs"`
}

type OutResponse struct {
	Version  Version        `json:"version"`
	Metadata []MetadataPair `json:"metadata"`
}

type Version struct {
	Tag       string `json:"tag,omitempty"`
	CommitSHA string `json:"commit_sha,omitempty"`
}

type MetadataPair struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	URL      string `json:"url"`
	Markdown bool   `json:"markdown"`
}
