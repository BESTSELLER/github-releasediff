package releasediff

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-github/v75/github"
	"github.com/hashicorp/go-version"
)

// GitHubReleases holds the two releases to compare
type GitHubReleases struct {
	Owner        string             // REQUIRED. Owner of the repo.
	Repo         string             // REQUIRED. Name of the repo.
	Release      string             // REQUIRED. Tag name of the release you want to check.
	Client       *github.Client     // Github client used to make the calls to the github api.
	Versions     []*version.Version // Ordered list of all versions.
	ReleaseNotes map[string]string  // A map of releases with release notes.
	Options      *Options           // Optional options
}

// Options is optional stuff that can be sent when calling "New"
type Options struct {
	Release            string // Tag name of the release you want to compare with. If empty, newest version will be used.
	Filter             string // Regex to to filter releases on. Keep empty if you want all releases.
	IncludePreReleases bool   // Whether to include pre-releases or not. Default is false.
	IncludeDrafts      bool   // Whether to include drafts or not. Default is false.
	VerifyRelease      bool   // Whether to verify that the provided versions exists as a release.
}

type ReleaseNote struct {
	Version string
	Body    string
}

func getVersions(releases []*github.RepositoryRelease) ([]*version.Version, *map[string]string, error) {
	errorList := []string{}
	releaseNotes := map[string]string{}
	versions := make([]*version.Version, len(releases))
	for i, raw := range releases {
		v, err := version.NewVersion(raw.GetTagName())
		if err != nil {
			errorList = append(errorList, fmt.Sprintf("%s", err))
			continue
		}
		versions[i] = v
		releaseNotes[v.Original()] = raw.GetBody()
	}

	if len(errorList) > 0 {
		return nil, nil, fmt.Errorf("you have the following errors: %s", strings.Join(errorList, "\n"))
	}

	sort.Sort(version.Collection(versions))

	return versions, &releaseNotes, nil
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
		return nil, nil, fmt.Errorf("missing required field(s): %s", missingFields)
	}

	if options == nil {
		options = &Options{}
	}

	ctx := context.Background()
	releases, response, err := getAllReleases(ctx, client, owner, repo, 1)
	if err != nil {
		return nil, nil, err
	}

	releases = filterReleases(releases, options.Filter, !options.IncludePreReleases, !options.IncludeDrafts)

	// Check if Release1 is a valid release
	if !isRelase(client, owner, repo, release, options.VerifyRelease) {
		return nil, nil, fmt.Errorf("'%s' is not a release on %s/%s", release, owner, repo)
	}

	if len(releases) == 0 {
		return nil, nil, fmt.Errorf("there is no releases")
	}

	versions, releaseNotes, err := getVersions(releases)
	if err != nil {
		return nil, nil, err
	}
	if len(versions) == 0 {
		return nil, nil, fmt.Errorf("no versions was returned")
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
		Owner:        owner,
		Repo:         repo,
		Release:      release,
		Client:       client,
		Versions:     versions,
		ReleaseNotes: *releaseNotes,
		Options:      options,
	}, response, nil
}

func (ghr *GitHubReleases) Diff() int {
	diff, _ := ghr.DiffWithReleaseNotes()
	return diff
}

// Diff will fetch all releases until a specific release
func (ghr *GitHubReleases) DiffWithReleaseNotes() (int, []ReleaseNote) {
	releaseNotes := []ReleaseNote{}
	if ghr.Release == ghr.Options.Release {
		return 0, nil
	}

	index1, index2, indexesFound := 0, 0, 0
	for i := len(ghr.Versions) - 1; i > 0; i-- {
		if indexesFound == 2 {
			break
		}

		currentVersion := ghr.Versions[i]

		if currentVersion.Original() == ghr.Release {
			index1 = i
			indexesFound++
			continue
		}

		releaseNotes = append(releaseNotes, ReleaseNote{
			Version: currentVersion.Original(),
			Body:    ghr.ReleaseNotes[currentVersion.Original()],
		})

		if currentVersion.Original() == ghr.Options.Release {
			index2 = i
			indexesFound++
			continue
		}
	}

	versionsBehind := int(math.Abs(float64(index1 - index2)))
	if versionsBehind == len(ghr.Versions)-1 {
		versionsBehind--
	}

	return versionsBehind, releaseNotes
}

// getAllReleases will fetch all releases
func getAllReleases(ctx context.Context, client *github.Client, owner string, repo string, page int) ([]*github.RepositoryRelease, *github.Response, error) {

	listOptions := &github.ListOptions{Page: page, PerPage: 100}
	releases, response, err := client.Repositories.ListReleases(ctx, owner, repo, listOptions)
	if err != nil {
		return releases, response, err
	}
	if response.StatusCode != http.StatusOK {
		return releases, response, fmt.Errorf("response not correct expected: %v got: %v", http.StatusOK, response.StatusCode)
	}

	if response.NextPage != 0 {
		aa, _, _ := getAllReleases(ctx, client, owner, repo, page+1)
		releases = append(aa, releases...)
	}
	return releases, response, err
}

// filterReleases will filter out where tag_name does not contains "filter"
func filterReleases(releases []*github.RepositoryRelease, filter string, removePreReleases bool, removeDrafts bool) []*github.RepositoryRelease {

	var filteredReleases []*github.RepositoryRelease

	for _, v := range releases {
		var matched bool
		if filter == "" {
			matched = true
		}
		matched, _ = regexp.MatchString(filter, v.GetTagName())
		if matched && v.GetPrerelease() == !removePreReleases && v.GetDraft() == !removeDrafts {
			filteredReleases = append(filteredReleases, v)
		}
	}

	return filteredReleases
}

// isRelease check if provided version is a release
func isRelase(client *github.Client, owner string, repo string, release string, verifyRelease bool) bool {
	if !verifyRelease {
		return true
	}
	_, _, err := client.Repositories.GetReleaseByTag(context.Background(), owner, repo, release)
	return err == nil
}
