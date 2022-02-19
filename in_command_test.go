package resource_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"code.gitea.io/sdk/gitea"
	"github.com/natto1784/gitea-release-resource"
	"github.com/natto1784/gitea-release-resource/fakes"
)

var _ = Describe("In Command", func() {
	var (
		command     *resource.InCommand
		giteaClient *fakes.FakeGitea
		giteaServer *ghttp.Server

		inRequest resource.InRequest

		inResponse resource.InResponse
		inErr      error

		tmpDir  string
		destDir string
	)

	BeforeEach(func() {
		var err error

		giteaClient = &fakes.FakeGitea{}
		giteaServer = ghttp.NewServer()
		command = resource.NewInCommand(giteaClient, ioutil.Discard)

		tmpDir, err = ioutil.TempDir("", "gitea-release")
		Ω(err).ShouldNot(HaveOccurred())

		destDir = filepath.Join(tmpDir, "destination")

		inRequest = resource.InRequest{}
	})

	AfterEach(func() {
		Ω(os.RemoveAll(tmpDir)).Should(Succeed())
	})

	buildTag := func(sha, tag string) *gitea.Tag {
		return &gitea.Tag{
			Commit: &gitea.Commit{
				ID: *gitea.String(sha),
			},
			Name: *gitea.String(tag),
		}
	}

	Context("when there is a tagged release", func() {
		Context("when a present version is specified", func() {
			BeforeEach(func() {
				giteaClient.GetTagReturns(buildTag("v0.35.0", "abc123"), nil)

				inRequest.Version = &resource.Version{
					Tag: "v0.35.0",
				}
			})

			Context("when valid asset filename globs are given", func() {
				BeforeEach(func() {
					inRequest.Params = resource.InParams{
						Globs: []string{"*.txt", "*.rtf"},
					}
				})

				It("succeeds", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					Ω(inErr).ShouldNot(HaveOccurred())
				})

				It("returns the fetched version", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					Ω(inResponse.Version).Should(Equal(resource.Version{Tag: "v0.35.0"}))
				})

				It("has some sweet metadata", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					Ω(inResponse.Metadata).Should(ConsistOf(
						resource.MetadataPair{Name: "tag", Value: "v0.35.0"},
						resource.MetadataPair{Name: "commit_sha", Value: "f28085a4a8f744da83411f5e09fd7b1709149eee"},
					))
				})

				It("calls #GetTag with the correct arguments", func() {
					command.Run(destDir, inRequest)

					Ω(giteaClient.GetTagArgsForCall(0)).Should(Equal("v0.35.0"))
				})

				It("does create the body, tag and version files", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					contents, err := ioutil.ReadFile(path.Join(destDir, "tag"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("v0.35.0"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "version"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("0.35.0"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "commit_sha"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("f28085a4a8f744da83411f5e09fd7b1709149eee"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "body"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("*markdown*"))
				})

				Context("when there is a custom tag filter", func() {
					BeforeEach(func() {
						inRequest.Source = resource.Source{
							TagFilter: "package-(.*)",
						}
						giteaClient.GetTagReturns(buildTag("package-0.35.0", "abc123"), nil)
						inResponse, inErr = command.Run(destDir, inRequest)
					})

					It("succeeds", func() {
						inResponse, inErr = command.Run(destDir, inRequest)

						Expect(inErr).ToNot(HaveOccurred())
					})

					It("does create the body, tag and version files", func() {
						inResponse, inErr = command.Run(destDir, inRequest)

						contents, err := ioutil.ReadFile(path.Join(destDir, "tag"))
						Ω(err).ShouldNot(HaveOccurred())
						Ω(string(contents)).Should(Equal("package-0.35.0"))

						contents, err = ioutil.ReadFile(path.Join(destDir, "version"))
						Ω(err).ShouldNot(HaveOccurred())
						Ω(string(contents)).Should(Equal("0.35.0"))
					})
				})

			})

			Context("when no globs are specified", func() {
				BeforeEach(func() {
					inRequest.Source = resource.Source{}
					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("succeeds", func() {
					Ω(inErr).ShouldNot(HaveOccurred())
				})

				It("returns the fetched version", func() {
					Ω(inResponse.Version).Should(Equal(resource.Version{Tag: "v0.35.0"}))
				})

				It("has some sweet metadata", func() {
					Ω(inResponse.Metadata).Should(ConsistOf(
						resource.MetadataPair{Name: "url", Value: "http://google.com"},
						resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
						resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
						resource.MetadataPair{Name: "tag", Value: "v0.35.0"},
						resource.MetadataPair{Name: "commit_sha", Value: "f28085a4a8f744da83411f5e09fd7b1709149eee"},
					))
				})

			})
		})
	})

	Context("when no tagged release is present", func() {
		BeforeEach(func() {
			giteaClient.GetTagReturns(nil, nil)

			inRequest.Version = &resource.Version{
				Tag: "v0.40.0",
			}

			inResponse, inErr = command.Run(destDir, inRequest)
		})

		It("returns an error", func() {
			Ω(inErr).Should(MatchError("no releases"))
		})
	})

	Context("when getting a tagged release fails", func() {
		disaster := errors.New("nope")

		BeforeEach(func() {
			giteaClient.GetTagReturns(nil, disaster)

			inRequest.Version = &resource.Version{
				Tag: "some-tag",
			}
			inResponse, inErr = command.Run(destDir, inRequest)
		})

		It("returns the error", func() {
			Ω(inErr).Should(Equal(disaster))
		})
	})
})
