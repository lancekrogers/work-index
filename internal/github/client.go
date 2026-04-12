package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/lancekrogers/work-index/internal/config"
)

// Repo is the raw data for a single GitHub repo.
type Repo struct {
	Name         string `json:"name"          yaml:"name"`
	Org          string `json:"-"             yaml:"org"`
	URL          string `json:"url"           yaml:"url"`
	Description  string `json:"description"   yaml:"description"`
	Language     string `json:"-"             yaml:"language"`
	Stars        int    `json:"stargazerCount" yaml:"stars"`
	CreatedAt    string `json:"createdAt"     yaml:"-"`
	PushedAt     string `json:"pushedAt"      yaml:"-"`
	Created      string `json:"-"             yaml:"created"`
	Pushed       string `json:"-"             yaml:"pushed"`
	IsFork       bool   `json:"isFork"        yaml:"fork"`
	Visibility   string `json:"visibility"    yaml:"-"`
	OwnerCommits *int   `json:"-"             yaml:"owner_commits,omitempty"`
	Status       string `json:"-"             yaml:"status"`

	RawParent    *parentInfo    `json:"parent"`
	RawLanguage  *languageInfo  `json:"primaryLanguage"`
}

type parentInfo struct {
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type languageInfo struct {
	Name string `json:"name"`
}

// ParentOwner returns the login of the fork's upstream owner, or empty string.
func (r *Repo) ParentOwner() string {
	if r.RawParent == nil {
		return ""
	}
	return r.RawParent.Owner.Login
}

// ID returns the org/name identifier.
func (r *Repo) ID() string {
	return r.Org + "/" + r.Name
}

// Contributor holds a GitHub contributor record.
type Contributor struct {
	Login         string `json:"login"`
	Contributions int    `json:"contributions"`
}

// FetchRepos fetches all public repos for an owner, applying fork/parent filters.
// Returns the filtered list and counts of skipped repos.
func FetchRepos(owner string, cfg *config.SyncConfig) (repos []Repo, skippedForks, skippedParent, skippedVisibility int, err error) {
	allowForks := cfg.AllowsForks(owner)

	sourceFlag := "--source"
	if allowForks {
		sourceFlag = ""
	}

	args := []string{"repo", "list", owner,
		"--limit", "1000",
		"--visibility", "public",
		"--no-archived",
		"--json", "name,visibility,description,url,primaryLanguage,pushedAt,createdAt,stargazerCount,isArchived,isFork,parent",
	}
	if sourceFlag != "" {
		args = append(args, sourceFlag)
	}

	out, execErr := exec.Command("gh", args...).Output()
	if execErr != nil {
		return nil, 0, 0, 0, fmt.Errorf("gh repo list %s: %w", owner, execErr)
	}

	var raw []Repo
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("parse repos for %s: %w", owner, err)
	}

	// Parallel fork contributor lookups.
	type forkResult struct {
		idx    int
		count  int
		err    error
	}

	var (
		forkChecks []forkResult
		mu         sync.Mutex
		wg         sync.WaitGroup
		sem        = make(chan struct{}, 10) // limit concurrency
	)

	for i := range raw {
		r := &raw[i]
		r.Org = owner
		r.Created = dateOnly(r.CreatedAt)
		r.Pushed = dateOnly(r.PushedAt)
		if r.RawLanguage != nil {
			r.Language = r.RawLanguage.Name
		} else {
			r.Language = "unknown"
		}

		// Visibility defense in depth.
		if r.Visibility != "PUBLIC" {
			skippedVisibility++
			continue
		}

		if !r.IsFork {
			repos = append(repos, *r)
			continue
		}

		// Fork: check parent exclusion.
		if cfg.IsExcludedParent(r.ParentOwner()) {
			skippedParent++
			continue
		}

		// Fork: need contributor check — enqueue.
		idx := len(repos)
		repos = append(repos, *r)
		wg.Add(1)
		go func(i, repoIdx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			count, cErr := ownerContributions(owner, raw[i].Name)
			mu.Lock()
			forkChecks = append(forkChecks, forkResult{idx: repoIdx, count: count, err: cErr})
			mu.Unlock()
		}(i, idx)
	}

	wg.Wait()

	// Remove forks with 0 owner commits (iterate in reverse to preserve indices).
	removeSet := make(map[int]bool)
	for _, fc := range forkChecks {
		if fc.err != nil {
			// On error, keep the repo to be safe.
			continue
		}
		if fc.count == 0 {
			removeSet[fc.idx] = true
			skippedForks++
		} else {
			c := fc.count
			repos[fc.idx].OwnerCommits = &c
		}
	}

	if len(removeSet) > 0 {
		filtered := make([]Repo, 0, len(repos)-len(removeSet))
		for i, r := range repos {
			if !removeSet[i] {
				filtered = append(filtered, r)
			}
		}
		repos = filtered
	}

	return repos, skippedForks, skippedParent, skippedVisibility, nil
}

// ownerContributions returns the contribution count for an owner in a repo.
func ownerContributions(owner, name string) (int, error) {
	out, err := exec.Command("gh", "api",
		fmt.Sprintf("repos/%s/%s/contributors", owner, name),
		"--jq", fmt.Sprintf(`.[] | select(.login == "%s") | .contributions`, owner),
	).Output()
	if err != nil {
		return 0, fmt.Errorf("contributors %s/%s: %w", owner, name, err)
	}

	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, nil
	}
	var count int
	if _, err := fmt.Sscanf(s, "%d", &count); err != nil {
		return 0, nil
	}
	return count, nil
}

func dateOnly(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}
