package resource_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.gitea.io/sdk/gitea"

	"github.com/natto1784/gitea-release-resource"
	"github.com/natto1784/gitea-release-resource/fakes"
)

func file(path, contents string) {
	Ω(ioutil.WriteFile(path, []byte(contents), 0644)).Should(Succeed())
}

var _ = Describe("Out Command", func() {
	var (
		command      *resource.OutCommand
		giteaClient *fakes.FakeGitea

		sourcesDir string

		request resource.OutRequest
	)

	BeforeEach(func() {
		var err error

		giteaClient = &fakes.FakeGitea{}
		command = resource.NewOutCommand(giteaClient, ioutil.Discard)

		sourcesDir, err = ioutil.TempDir("", "gitea-release")
		Ω(err).ShouldNot(HaveOccurred())

		giteaClient.CreateReleaseStub = func(gh gitea.Tag) (*gitea.Tag, error) {
			createdRel := gh
			createdRel.ID = gitea.Int(112)
			createdRel.HTMLURL = gitea.String("http://google.com")
			createdRel.Name = gitea.String("release-name")
			createdRel.Body = gitea.String("*markdown*")
			return &createdRel, nil
		}

		giteaClient.UpdateReleaseStub = func(gh gitea.Tag) (*gitea.Tag, error) {
			return &gh, nil
		}
	})

	AfterEach(func() {
		Ω(os.RemoveAll(sourcesDir)).Should(Succeed())
	})

	Context("when the release has already been created", func() {
		existingAssets := []gitea.ReleaseAsset{
			{
				ID:   gitea.Int(456789),
				Name: gitea.String("unicorns.txt"),
			},
			{
				ID:    gitea.Int(3450798),
				Name:  gitea.String("rainbows.txt"),
				State: gitea.String("new"),
			},
		}

		existingReleases := []gitea.Tag{
			{
				ID:    gitea.Int(1),
				Draft: gitea.Bool(true),
			},
			{
				ID:      gitea.Int(112),
				TagName: gitea.String("some-tag-name"),
				Assets:  []gitea.ReleaseAsset{existingAssets[0]},
				Draft:   gitea.Bool(false),
			},
		}

		BeforeEach(func() {
			giteaClient.ListReleasesStub = func() ([]*gitea.Tag, error) {
				rels := []*gitea.Tag{}
				for _, r := range existingReleases {
					c := r
					rels = append(rels, &c)
				}

				return rels, nil
			}

			giteaClient.ListReleaseAssetsStub = func(gitea.Tag) ([]*gitea.ReleaseAsset, error) {
				assets := []*gitea.ReleaseAsset{}
				for _, a := range existingAssets {
					c := a
					assets = append(assets, &c)
				}

				return assets, nil
			}

			namePath := filepath.Join(sourcesDir, "name")
			bodyPath := filepath.Join(sourcesDir, "body")
			tagPath := filepath.Join(sourcesDir, "tag")

			file(namePath, "v0.3.12")
			file(bodyPath, "this is a great release")
			file(tagPath, "some-tag-name")

			request = resource.OutRequest{
				Params: resource.OutParams{
					NamePath: "name",
					BodyPath: "body",
					TagPath:  "tag",
				},
			}
		})

		It("deletes the existing assets", func() {
			_, err := command.Run(sourcesDir, request)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(giteaClient.ListReleaseAssetsCallCount()).Should(Equal(1))
			Ω(giteaClient.ListReleaseAssetsArgsForCall(0)).Should(Equal(existingReleases[1]))

			Ω(giteaClient.DeleteReleaseAssetCallCount()).Should(Equal(2))

			Ω(giteaClient.DeleteReleaseAssetArgsForCall(0)).Should(Equal(existingAssets[0]))
			Ω(giteaClient.DeleteReleaseAssetArgsForCall(1)).Should(Equal(existingAssets[1]))
		})

		Context("when not set as a draft release", func() {
			BeforeEach(func() {
				request.Source.Drafts = false
			})

			It("updates the existing release to a non-draft", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := giteaClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Draft).Should(Equal(false))
			})
		})

		Context("when set as a draft release", func() {
			BeforeEach(func() {
				request.Source.Drafts = true
			})

			It("updates the existing release to a draft", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := giteaClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Draft).Should(Equal(true))
			})
		})

		Context("when a body is not supplied", func() {
			BeforeEach(func() {
				request.Params.BodyPath = ""
			})

			It("does not blow away the body", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := giteaClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(updatedRelease.Body).Should(BeNil())
			})
		})

		Context("when a commitish is not supplied", func() {
			It("updates the existing release", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := giteaClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Body).Should(Equal("this is a great release"))
				Ω(updatedRelease.TargetCommitish).Should(Equal(gitea.String("")))
			})
		})

		Context("when a commitish is supplied", func() {
			BeforeEach(func() {
				commitishPath := filepath.Join(sourcesDir, "commitish")
				file(commitishPath, "1z22f1")
				request.Params.CommitishPath = "commitish"
			})

			It("updates the existing release", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := giteaClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Body).Should(Equal("this is a great release"))
				Ω(updatedRelease.TargetCommitish).Should(Equal(gitea.String("1z22f1")))
			})
		})
	})

	Context("when the release has not already been created", func() {
		BeforeEach(func() {
			namePath := filepath.Join(sourcesDir, "name")
			tagPath := filepath.Join(sourcesDir, "tag")

			file(namePath, "v0.3.12")
			file(tagPath, "0.3.12")

			request = resource.OutRequest{
				Params: resource.OutParams{
					NamePath: "name",
					TagPath:  "tag",
				},
			}
		})

		Context("with a commitish", func() {
			BeforeEach(func() {
				commitishPath := filepath.Join(sourcesDir, "commitish")
				file(commitishPath, "a2f4a3")
				request.Params.CommitishPath = "commitish"
			})

			It("creates a release on gitea with the commitish", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
				release := giteaClient.CreateReleaseArgsForCall(0)

				Ω(release.TargetCommitish).Should(Equal(gitea.String("a2f4a3")))
			})
		})

		Context("without a commitish", func() {
			It("creates a release on gitea without the commitish", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
				release := giteaClient.CreateReleaseArgsForCall(0)

				// gitea treats empty string the same as not suppying the field.
				Ω(release.TargetCommitish).Should(Equal(gitea.String("")))
			})
		})

		Context("with a body", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Params.BodyPath = "body"
			})

			It("creates a release on gitea", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
				release := giteaClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal("this is a great release"))
			})
		})

		Context("without a body", func() {
			It("works", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
				release := giteaClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
			})
		})

		It("always defaults to non-draft mode", func() {
			_, err := command.Run(sourcesDir, request)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
			release := giteaClient.CreateReleaseArgsForCall(0)

			Ω(*release.Draft).Should(Equal(false))
		})

		Context("when pre-release are set and release are not", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Source.Release = false
				request.Source.PreRelease = true
			})

			It("creates a non-draft pre-release in gitea", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
				release := giteaClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
				Ω(*release.Draft).Should(Equal(false))
				Ω(*release.Prerelease).Should(Equal(true))
			})

			It("has some sweet metadata", func() {
				outResponse, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(outResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "0.3.12"},
					resource.MetadataPair{Name: "pre-release", Value: "true"},
				))
			})
		})

		Context("when release and pre-release are set", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Source.Release = true
				request.Source.PreRelease = true
			})

			It("creates a final release in gitea", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
				release := giteaClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
				Ω(*release.Draft).Should(Equal(false))
				Ω(*release.Prerelease).Should(Equal(false))
			})

			It("has some sweet metadata", func() {
				outResponse, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(outResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "0.3.12"},
				))
			})
		})

		Context("when set as a draft release", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Source.Drafts = true
			})

			It("creates a release on gitea in draft mode", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
				release := giteaClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
				Ω(*release.Draft).Should(Equal(true))
				Ω(*release.Prerelease).Should(Equal(false))
			})

			It("has some sweet metadata", func() {
				outResponse, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(outResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "0.3.12"},
					resource.MetadataPair{Name: "draft", Value: "true"},
				))
			})
		})

		Context("with file globs", func() {
			BeforeEach(func() {
				globMatching := filepath.Join(sourcesDir, "great-file.tgz")
				globNotMatching := filepath.Join(sourcesDir, "bad-file.txt")

				file(globMatching, "matching")
				file(globNotMatching, "not matching")

				request = resource.OutRequest{
					Params: resource.OutParams{
						NamePath: "name",
						BodyPath: "body",
						TagPath:  "tag",

						Globs: []string{
							"*.tgz",
						},
					},
				}

				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "*markdown*")
				request.Params.BodyPath = "body"
			})

			It("uploads matching file globs", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.UploadReleaseAssetCallCount()).Should(Equal(1))
				release, name, file := giteaClient.UploadReleaseAssetArgsForCall(0)

				Ω(*release.ID).Should(Equal(112))
				Ω(name).Should(Equal("great-file.tgz"))
				Ω(file.Name()).Should(Equal(filepath.Join(sourcesDir, "great-file.tgz")))
			})

			It("has some sweet metadata", func() {
				outResponse, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(outResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "0.3.12"},
				))
			})

			It("returns an error if a glob is provided that does not match any files", func() {
				request.Params.Globs = []string{
					"*.tgz",
					"*.gif",
				}

				_, err := command.Run(sourcesDir, request)
				Ω(err).Should(HaveOccurred())
				Ω(err).Should(MatchError("could not find file that matches glob '*.gif'"))
			})

			Context("when upload release asset fails", func() {
				BeforeEach(func() {
					existingAsset := false
					giteaClient.DeleteReleaseAssetStub = func(gitea.ReleaseAsset) error {
						existingAsset = false
						return nil
					}

					giteaClient.ListReleaseAssetsReturns([]*gitea.ReleaseAsset{
						{
							ID:   gitea.Int(456789),
							Name: gitea.String("great-file.tgz"),
						},
						{
							ID:   gitea.Int(3450798),
							Name: gitea.String("whatever.tgz"),
						},
					}, nil)

					giteaClient.UploadReleaseAssetStub = func(rel gitea.Tag, name string, file *os.File) error {
						Expect(ioutil.ReadAll(file)).To(Equal([]byte("matching")))
						Expect(existingAsset).To(BeFalse())
						existingAsset = true
						return errors.New("some-error")
					}
				})

				It("retries 10 times", func() {
					_, err := command.Run(sourcesDir, request)
					Expect(err).To(Equal(errors.New("some-error")))

					Ω(giteaClient.UploadReleaseAssetCallCount()).Should(Equal(10))
					Ω(giteaClient.ListReleaseAssetsCallCount()).Should(Equal(10))
					Ω(*giteaClient.ListReleaseAssetsArgsForCall(9).ID).Should(Equal(112))

					actualRelease, actualName, actualFile := giteaClient.UploadReleaseAssetArgsForCall(9)
					Ω(*actualRelease.ID).Should(Equal(112))
					Ω(actualName).Should(Equal("great-file.tgz"))
					Ω(actualFile.Name()).Should(Equal(filepath.Join(sourcesDir, "great-file.tgz")))

					Ω(giteaClient.DeleteReleaseAssetCallCount()).Should(Equal(10))
					actualAsset := giteaClient.DeleteReleaseAssetArgsForCall(8)
					Expect(*actualAsset.ID).To(Equal(456789))
				})

				Context("when uploading succeeds on the 5th attempt", func() {
					BeforeEach(func() {
						results := make(chan error, 6)
						results <- errors.New("1")
						results <- errors.New("2")
						results <- errors.New("3")
						results <- errors.New("4")
						results <- nil
						results <- errors.New("6")

						giteaClient.UploadReleaseAssetStub = func(gitea.Tag, string, *os.File) error {
							return <-results
						}
					})

					It("succeeds", func() {
						_, err := command.Run(sourcesDir, request)
						Expect(err).ToNot(HaveOccurred())

						Ω(giteaClient.UploadReleaseAssetCallCount()).Should(Equal(5))
						Ω(giteaClient.ListReleaseAssetsCallCount()).Should(Equal(4))
						Ω(*giteaClient.ListReleaseAssetsArgsForCall(3).ID).Should(Equal(112))

						actualRelease, actualName, actualFile := giteaClient.UploadReleaseAssetArgsForCall(4)
						Ω(*actualRelease.ID).Should(Equal(112))
						Ω(actualName).Should(Equal("great-file.tgz"))
						Ω(actualFile.Name()).Should(Equal(filepath.Join(sourcesDir, "great-file.tgz")))

						Ω(giteaClient.DeleteReleaseAssetCallCount()).Should(Equal(4))
						actualAsset := giteaClient.DeleteReleaseAssetArgsForCall(3)
						Expect(*actualAsset.ID).To(Equal(456789))
					})
				})
			})
		})

		Context("when the tag_prefix is set", func() {
			BeforeEach(func() {
				namePath := filepath.Join(sourcesDir, "name")
				tagPath := filepath.Join(sourcesDir, "tag")

				file(namePath, "v0.3.12")
				file(tagPath, "0.3.12")

				request = resource.OutRequest{
					Params: resource.OutParams{
						NamePath:  "name",
						TagPath:   "tag",
						TagPrefix: "version-",
					},
				}
			})

			It("appends the TagPrefix onto the TagName", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(giteaClient.CreateReleaseCallCount()).Should(Equal(1))
				release := giteaClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("version-0.3.12"))
			})
		})
	})
})
