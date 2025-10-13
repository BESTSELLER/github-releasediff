package releasediff

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/google/go-github/v75/github"
	"golang.org/x/oauth2"
)

type TestCase struct {
	Owner         string
	Repo          string
	Filter        string
	VerifyRelease bool
	Release       string
}

func TestMain(t *testing.T) {
	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN"), TokenType: "token"},
	))
	client := github.NewClient(tc)

	testCases := []TestCase{
		{
			Owner:         "kubernetes",
			Repo:          "ingress-nginx",
			Filter:        "^controller-.*$",
			VerifyRelease: false,
			Release:       "controller-0.31.0",
		},
		{
			Owner:         "BESTSELLER",
			Repo:          "harpocrates",
			VerifyRelease: false,
			Release:       "1.7.6",
		},
		{
			Owner:         "hashicorp",
			Repo:          "vault",
			VerifyRelease: false,
			Release:       "v1.10.3",
		},
	}

	var rate github.Rate
	var wg sync.WaitGroup
	for _, tc := range testCases {

		wg.Add(1)
		go func(testCase TestCase) {
			defer wg.Done()

			ghr, resp, err := New(client, testCase.Owner, testCase.Repo, testCase.Release, &Options{Filter: testCase.Filter, VerifyRelease: testCase.VerifyRelease})
			if err != nil {
				panic(err)
			}

			diff := ghr.Diff()
			t.Logf("%s/%s:\tThere are %d releases between %s and %s\n", testCase.Owner, testCase.Repo, diff, ghr.Release, ghr.Options.Release)
			rate = resp.Rate

		}(tc)

	}
	wg.Wait()
	t.Logf("%v\n", rate)
}
