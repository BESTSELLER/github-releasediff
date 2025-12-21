# github-releasediff

> [!IMPORTANT]
> This library is no longer maintained and is archived. It will eventually be deleted.

Simple go package to get number of releases between two releases.

The github client used is [google/go-github](https://github.com/google/go-github)
## Usage
```go
// Create a github client
tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
  &oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN"), TokenType: "token"},
))
client := github.NewClient(tc)

//
ghr, err := releasediff.New(client, "goharbor", "harbor", "v2.0.2", "", nil)
if err != nil {
  panic(err)
}

diff, releaseNotes, err := ghr.Diff()
if err != nil {
  panic(err)
}
fmt.Printf("There are %d releases between %s and %s\n", diff, ghr.Release, ghr.Options.Release)
fmt.Printf("This is the release notes: \n%v", releaseNotes)
fmt.Printf("%v\n", resp.Rate)

// Output:
// There are 2 releases between v2.0.2 and v2.1.0
// github.Rate{Limit:5000, Remaining:4685, Reset:github.Timestamp{2020-10-07 09:30:12 +0200 CEST}}
```
