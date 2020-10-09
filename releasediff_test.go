package releasediff

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

func TestMain(m *testing.M) {
	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN"), TokenType: "token"},
	))
	client := github.NewClient(tc)

	repos := make(map[string]string)
	repos["ingress-nginx"] = "controller-0.31.0"

	var rate github.Rate
	var wg sync.WaitGroup
	for name, version := range repos {

		wg.Add(1)
		go func(name string, version string) {
			defer wg.Done()

			ghr, resp, err := New(client, "kubernetes", name, version, &Options{Filter: "^controller-.*$", VerifyRelease: false})
			if err != nil {
				panic(err)
			}

			diff := ghr.Diff()
			fmt.Printf("There are %d releases between %s and %s\n", diff, ghr.Release, ghr.Options.Release)
			rate = resp.Rate

		}(name, version)

	}
	wg.Wait()
	fmt.Printf("%v\n", rate)
}
