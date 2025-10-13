module github.com/BESTSELLER/github-releasediff

go 1.25.0

require (
	github.com/google/go-github/v75 v75.0.0
	github.com/hashicorp/go-version v1.7.0
	golang.org/x/oauth2 v0.32.0
)

require github.com/google/go-querystring v1.1.0 // indirect

replace github.com/hashicorp/go-version => github.com/BESTSELLER/go-version v1.6.0
