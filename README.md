# github-releasediff

Simple go package to get number of releases between two releases.

## Usage
```go

// Create a github client
tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
  &oauth2.Token{AccessToken: "<INSERT GITHUB TOKEN HERE>", TokenType: "token"},
))
client := github.NewClient(tc)


ghr := releasediff.GitHubReleases{
  Owner:    "goharbor",
  Repo:     "harbor",
  Release1: "v2.0.2",
}
ghr.New(client)

diff, resp, err := ghr.Diff()
if err != nil {
  panic(err)
}
fmt.Printf("There are %d releases between %s and %s\n", diff, ghr.Release1, ghr.Release2)
fmt.Printf("%v\n", resp.Rate)

// Output:
// There are 2 releases between v2.0.2 and v2.1.0
// github.Rate{Limit:5000, Remaining:4685, Reset:github.Timestamp{2020-10-07 09:30:12 +0200 CEST}}
```