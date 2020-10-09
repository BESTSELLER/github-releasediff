package releasediff

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/go-version"
)

// GitHubReleases holds the two releases to compare
type GitHubReleases struct {
	Owner    string             // REQUIRED. Owner of the repo.
	Repo     string             // REQUIRED. Name of the repo.
	Release  string             // REQUIRED. Tag name of the release you want to check.
	Client   *github.Client     // Github client used to make the calls to the github api.
	Versions []*version.Version // Ordered list of all versions.
	Options  *Options           // Optional options
}

// Options is optional stuff that can be sent when calling "New"
type Options struct {
	Release            string // Tag name of the release you want to compare with. If empty, newest version will be used.
	Filter             string // Regex to to filter releases on. Keep empty if you want all releases.
	IncludePreReleases bool   // Whether to include pre-releases or not. Default is false.
	VerifyRelease      bool   // Whether to verify that the provided versions exists as a release.
}

func getVersions(releases []*github.RepositoryRelease) ([]*version.Version, error) {
	errorList := []string{}
	versions := make([]*version.Version, len(releases))
	for i, raw := range releases {
		v, err := version.NewVersion(raw.GetTagName())
		if err != nil {
			fmt.Println(err)
			errorList = append(errorList, fmt.Sprintf("%s", err))
			continue
		}
		versions[i] = v
	}

	if len(errorList) > 0 {
		return nil, fmt.Errorf("You have the following errors: %s", strings.Join(errorList, "\n"))
	}

	sort.Sort(version.Collection(versions))

	return versions, nil
}

// New creates a new GitHubReleases
func New(client *github.Client, owner string, repo string, release string, options *Options) (*GitHubReleases, *github.Response, error) {
	missingFields := []string{}
	if owner == "" {
		missingFields = append([]string{"Owner"}, missingFields...)
	}
	if repo == "" {
		missingFields = append([]string{"Repo"}, missingFields...)
	}
	if release == "" {
		missingFields = append([]string{"Release1"}, missingFields...)
	}
	if len(missingFields) > 0 {
		return nil, nil, fmt.Errorf("Missing required field(s): %s", missingFields)
	}

	if options == nil {
		options = &Options{}
	}

	ctx := context.Background()
	releases, response, err := getAllReleases(ctx, client, owner, repo, 1)
	if err != nil {
		return nil, nil, err
	}

	releases = filterReleases(releases, options.Filter)

	if !options.IncludePreReleases {
		releases = removePreReleases(releases)
	}

	// Check if Release1 is a valid release
	if !isRelase(client, owner, repo, release, options.VerifyRelease) {
		return nil, nil, fmt.Errorf("'%s' is not a release on %s/%s", release, owner, repo)
	}

	if len(releases) == 0 {
		return nil, nil, fmt.Errorf("There is no releases")
	}

	versions, err := getVersions(releases)
	if err != nil {
		return nil, nil, err
	}
	if len(versions) == 0 {
		return nil, nil, fmt.Errorf("No versions was returned")
	}

	// if release2 is empty we will use the latest version
	if options.Release == "" {
		options.Release = versions[len(versions)-1].Original()
	} else {
		// Check if options.Release is a valid release
		if !isRelase(client, owner, repo, release, options.VerifyRelease) {
			return nil, nil, fmt.Errorf("'%s' is not a release on %s/%s", options.Release, owner, repo)
		}
	}

	return &GitHubReleases{
		Owner:    owner,
		Repo:     repo,
		Release:  release,
		Client:   client,
		Versions: versions,
		Options:  options,
	}, response, nil
}

// Diff will fetch all releases until a specific release
func (ghr *GitHubReleases) Diff() int {
	if ghr.Release == ghr.Options.Release {
		return 0
	}

	index1, index2, indexesFound := 0, 0, 0
	for i := len(ghr.Versions) - 1; i > 0; i-- {
		if indexesFound == 2 {
			break
		}
		if ghr.Versions[i].Original() == ghr.Release {
			index1 = i
			continue
		}
		if ghr.Versions[i].Original() == ghr.Options.Release {
			index2 = i
			continue
		}
	}

	versionsBehind := int(math.Abs(float64(index1 - index2)))
	if versionsBehind == len(ghr.Versions)-1 {
		versionsBehind--
	}

	return versionsBehind
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

// isRelease check if provided version is a release
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
