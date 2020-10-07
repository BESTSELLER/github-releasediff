package releasediff

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/go-version"
)

// GitHubReleases holds the two releases to compare
type GitHubReleases struct {
	Owner              string         // REQUIRED. Owner of the repo.
	Repo               string         // REQUIRED. Name of the repo.
	Release1           string         // REQUIRED. Tag name of the release you want to check.
	Release2           string         // Tag name of the release you want to compare with. If empty, newest version will be used.
	Filter             string         // Regex to to filter releases on. Keep empty if you want all releases.
	IncludePreReleases bool           // Whether to include pre-releases or not. Default is false.
	Client             *github.Client // Github client used to make the calls to the github api.
}

func isRelase(client *github.Client, owner string, repo string, release string, verifyRelease bool) bool {
	if !verifyRelease {
		return true
	}
	_, _, err := client.Repositories.GetReleaseByTag(context.Background(), owner, repo, release)
	if err != nil {
		return false
	}
	return true
}

// New creates a new GitHubReleases
func New(client *github.Client, owner string, repo string, release1 string, release2 string, filter string, includePreReleases bool, verifyRelease bool) (*GitHubReleases, error) {
	missingFields := []string{}
	if owner == "" {
		missingFields = append([]string{"Owner"}, missingFields...)
	}
	if repo == "" {
		missingFields = append([]string{"Repo"}, missingFields...)
	}
	if release1 == "" {
		missingFields = append([]string{"Release1"}, missingFields...)
	}
	if len(missingFields) > 0 {
		return nil, fmt.Errorf("Missing required field(s): %s", missingFields)
	}

	// Check if Release1 is a valid release
	if !isRelase(client, owner, repo, release1, verifyRelease) {
		return nil, fmt.Errorf("'%s' is not a release on %s/%s", release1, owner, repo)
	}

	// if release2 is empty we will use the latest version
	if release2 == "" {
		latest, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
		if err != nil {
			return nil, err
		}
		release2 = latest.GetTagName()
	} else {
		// Check if Release2 is a valid release
		if !isRelase(client, owner, repo, release1, verifyRelease) {
			return nil, fmt.Errorf("'%s' is not a release on %s/%s", release2, owner, repo)
		}
	}

	return &GitHubReleases{
		Owner:              owner,
		Repo:               repo,
		Release1:           release1,
		Release2:           release2,
		Filter:             filter,
		IncludePreReleases: includePreReleases,
		Client:             client,
	}, nil
}

// Diff will fetch all releases until a specific release
func (ghr *GitHubReleases) Diff() (int, *github.Response, error) {
	ctx := context.Background()
	releases, response, err := getAllReleases(ctx, ghr.Client, ghr.Owner, ghr.Repo, 1)

	if ghr.Filter != "" {
		releases = filterReleases(releases, ghr.Filter)
	}

	if !ghr.IncludePreReleases {
		releases = removePreReleases(releases)
	}

	versions := make([]*version.Version, len(releases))
	for i, raw := range releases {
		v, _ := version.NewVersion(raw.GetTagName())
		versions[i] = v
	}
	sort.Sort(version.Collection(versions))

	index1, index2 := 0, 0
	indexesFound := 0
	for i := len(versions) - 1; i > 0; i-- {
		if indexesFound == 2 {
			break
		}
		if versions[i].Original() == ghr.Release1 {
			index1 = i
		}
		if versions[i].Original() == ghr.Release2 {
			index2 = i
		}
	}
	versionsBehind := int(math.Abs(float64(index1 - index2)))

	return versionsBehind, response, err
}

// getAllReleases will fetch all releases
func getAllReleases(ctx context.Context, client *github.Client, owner string, repo string, page int) ([]*github.RepositoryRelease, *github.Response, error) {

	listOptions := &github.ListOptions{Page: page, PerPage: 100}
	releases, response, err := client.Repositories.ListReleases(ctx, owner, repo, listOptions)
	if err != nil {
		return releases, response, err
	}
	if response.StatusCode != http.StatusOK {
		return releases, response, fmt.Errorf("Response not correct expected: %v got: %v", http.StatusOK, response.StatusCode)
	}

	if response.NextPage != 0 {
		aa, _, _ := getAllReleases(ctx, client, owner, repo, page+1)
		releases = append(aa, releases...)
	}
	return releases, response, err
}

// filterReleases will filter out where tag_name does not contains "filter"
func filterReleases(releases []*github.RepositoryRelease, filter string) []*github.RepositoryRelease {
	if filter == "" {
		return releases
	}

	var filteredReleases []*github.RepositoryRelease
	for _, v := range releases {
		matched, _ := regexp.MatchString(filter, v.GetTagName())
		if matched {
			filteredReleases = append(filteredReleases, v)
		}
	}

	return filteredReleases
}

// removePreReleases removes all pre-releases
func removePreReleases(releases []*github.RepositoryRelease) []*github.RepositoryRelease {
	var nonPreReleases []*github.RepositoryRelease
	for _, v := range releases {
		if v.GetPrerelease() == false {
			nonPreReleases = append(nonPreReleases, v)
		}
	}
	return nonPreReleases
}
