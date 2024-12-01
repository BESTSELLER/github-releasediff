module github.com/BESTSELLER/github-releasediff

go 1.21.3

require (
	github.com/google/go-github/v56 v56.0.0
	github.com/hashicorp/go-version v1.2.1
	golang.org/x/oauth2 v0.24.0
)

require github.com/google/go-querystring v1.1.0 // indirect

replace github.com/hashicorp/go-version => github.com/BESTSELLER/go-version v1.2.5
