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
	repos["harbor"] = "v2.0.2"
	repos["harbor-helm"] = "v1.5.0"

	// repos := []string{"harbor", "harbor-helm"}
	var wg sync.WaitGroup
	for name, version := range repos {

		wg.Add(1)
		go func(name string, version string) {
			defer wg.Done()

			ghr, err := New(client, "goharbor", name, version, "", nil)
			if err != nil {
				panic(err)
			}

			diff, resp, err := ghr.Diff()
			if err != nil {
				panic(err)
			}
			fmt.Printf("There are %d releases between %s and %s\n", diff, ghr.Release1, ghr.Release2)
			fmt.Printf("%v\n", resp.Rate)

		}(name, version)

	}
	wg.Wait()
}
