package resource_test

import (
	"net/http"

	. "github.com/natto1784/gitea-release-resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.gitea.io/sdk/gitea"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Gitea Client", func() {
	var server *ghttp.Server
	var client *GiteaClient
	var source Source

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	JustBeforeEach(func() {
		source.GiteaAPIURL = server.URL()

		var err error
		client, err = NewGiteaClient(source)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Context("with bad URLs", func() {
		BeforeEach(func() {
			source.AccessToken = "hello?"
		})

		It("returns an error if the API URL is bad", func() {
			source.GiteaAPIURL = ":"

			_, err := NewGiteaClient(source)
			Ω(err).Should(HaveOccurred())
		})
	})

	Context("with an OAuth Token", func() {
		BeforeEach(func() {
			source = Source{
				Repository:  "concourse",
				AccessToken: "abc123",
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases"),
					ghttp.RespondWith(200, "[]"),
					ghttp.VerifyHeaderKV("Authorization", "Bearer abc123"),
				),
			)
		})

		It("sends one", func() {
			_, err := client.ListTags()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("without an OAuth Token", func() {
		BeforeEach(func() {
			source = Source{
				Repository: "concourse",
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases"),
					ghttp.RespondWith(200, "[]"),
					ghttp.VerifyHeader(http.Header{"Authorization": nil}),
				),
			)
		})

		It("sends one", func() {
			_, err := client.ListTags()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("GetRelease", func() {
		BeforeEach(func() {
			source = Source{
				Repository: "concourse",
			}
		})
	})

	Describe("GetReleaseByTag", func() {
		BeforeEach(func() {
			source = Source{
				Repository: "concourse",
			}
		})

		Context("When Gitea responds successfully", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases/tags/some-tag"),
						ghttp.RespondWith(200, `{ "id": "1" }`),
					),
				)
			})

			It("Returns a populated github.Tag", func() {
				expectedRelease := &gitea.Tag{
					Name: *gitea.String("1"),
				}

				release, err := client.GetTag("some-tag")

				Ω(err).ShouldNot(HaveOccurred())
				Expect(release).To(Equal(expectedRelease))
			})
		})
	})
})
