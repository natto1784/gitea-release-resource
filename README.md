[![Concourse CI](https://ci.weirdnatto.in/api/v1/teams/main/pipelines/gitea-release-resource/badge)](https://ci.weirdnatto.in/teams/main/pipelines/gitea-release-resource) [![Docker](https://img.shields.io/docker/image-size/natto17/gitea-release-resource.svg)](https://hub.docker.com/repository/docker/natto17/gitea-release-resorce)
# WIP
Forked from https://github.com/edtan/gitlab-release-resource


# Gitea Releases Resource

Fetches and creates versioned Gitea resources.  GitLab resources are metadata attached to tags.  Note that `check` will skip tags that do not have associated resources.

Note that this is still in development, and is still undergoing changes.  It may or may not work properly at the moment, but should hopefully be somewhat more stable soon.

You may want to clean up your uploads folder over time if you re-run a put step with the same inputs, as this will simply re-upload the files under a new hash.

## Source Configuration

* `repository`: *Required.* The repository name that contains the releases.

* `access_token`: *Required.* Used for accessing a release in a private-repo
   during an `in` and pushing a release to a repo during an `out`. The access
   token you create is only required to have the `repo` or `public_repo` scope.

* `gitea_api_url`: *Optional.* If you use a non-public Gitea deployment then
  you can set your API URL here.

* `insecure`: *Optional. Default `false`.* When set to `true`, concourse will allow
  insecure connection to your gitea API.

* `tag_filter`: *Optional.* If set, override default tag filter regular
  expression of `v?([^v].*)`. If the filter includes a capture group, the capture
  group is used as the release version; otherwise, the entire matching substring
  is used as the version.

### Example

``` yaml
- name: gl-release
  type: gitea-release
  source:
    repository: group/project
    access_token: abcdef1234567890
```

``` yaml
- get: gl-release
```

``` yaml
- put: gl-release
  params:
    tag: path/to/tag/file
    body: path/to/body/file
    globs:
    - paths/to/files/to/upload-*.tgz
```

To get a specific version of a release:

``` yaml
- get: gl-release
  version: { tag: 'v0.0.1' }
```

To set a custom tag filter:

```yaml
- name: gl-release
  type: gitea-release
  source:
    owner: concourse
    repository: concourse
    tag_filter: "version-(.*)"
```

## Behavior

### `check`: Check for released versions.

Releases are listed and sorted by their tag, using
[semver](http://semver.org) semantics if possible. If `version` is specified, `check` returns releases from the specified version on. Otherwise, `check` returns the latest release.

### `in`: Fetch assets from a release.

Fetches artifacts from the given release version. If the version is not
specified, the latest version is chosen using [semver](http://semver.org)
semantics.

Also creates the following files:

* `tag` containing the git tag name of the release being fetched.
* `version` containing the version determined by the git tag of the release being fetched.
* `body` containing the body text of the release.
* `commit_sha` containing the commit SHA the tag is pointing to.

#### Parameters

* `globs`: *Optional.* A list of globs for files that will be downloaded from
  the release. If not specified, all assets will be fetched.

### `out`: Publish a release.

Given a `commit_sha` and  `tag`, this tags the commit and creates a release on Gitea, then uploads the files
matching the patterns in `globs` to the release.

#### Parameters

* `commitish`: *Optional, if tag is not specified.* A path to a file containing the commitish (SHA, tag,
  branch name) that the new tag and release should be associated with.

* `tag`: *Required.* A path to a file containing the name of the Git tag to use
  for the release.

* `tag_prefix`: *Optional.*  If specified, the tag read from the file will be
prepended with this string. This is useful for adding v in front of version numbers.

* ~~`body`: *Optional.* A path to a file containing the body text of the release.~~ (Not yet supported)

* `globs`: *Optional.* A list of globs for files that will be uploaded alongside
  the created release.

## Development

### Prerequisites

* golang is *required* - version 1.9.x is tested; earlier versions may also
  work.
* docker is *required* - version 17.06.x is tested; earlier versions may also
  work.
* godep is used for dependency management of the golang packages.

### Running the tests

The tests have been embedded with the `Dockerfile`; ensuring that the testing
environment is consistent across any `docker` enabled platform. When the docker
image builds, the test are run inside the docker container, on failure they
will stop the build.

Run the tests with the following command:

```sh
docker build -t gitea-release-resource .
```

### Contributing

Please make all pull requests to the `master` branch and ensure tests pass
locally.
