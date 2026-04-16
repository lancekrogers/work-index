package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/lancekrogers/work-index/internal/catalog"
	"github.com/lancekrogers/work-index/internal/config"
	"github.com/lancekrogers/work-index/internal/github"
	"gopkg.in/yaml.v3"
)

const (
	syncConfigPath = "config/sync-config.yaml"
	curationPath   = "config/curation.yaml"
	rawPath        = "config/repos-raw.yaml"
	categoriesDir  = "categories"
	readmePath     = "README.md"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "sync":
		err = cmdSync()
	case "generate":
		err = cmdGenerate()
	case "add":
		err = cmdAdd()
	case "exclude":
		err = cmdExclude()
	case "status":
		err = cmdStatus()
	case "uncurated":
		err = cmdUncurated()
	case "profile":
		err = cmdProfile()
	default:
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: workindex <command>

Commands:
  sync           Pull repos from GitHub + render category pages
  generate       Render category pages from current data (no GitHub fetch)
  add <id> <cat> [summary]   Add a project to the catalog
  exclude <id>   Exclude a repo from the catalog
  status         Show sync status and counts
  uncurated      List unreviewed repos
  profile        Generate GitHub profile README with links to catalog`)
}

// cmdSync pulls repos from GitHub, writes repos-raw.yaml, and renders categories.
func cmdSync() error {
	cfg, err := config.LoadSyncConfig(syncConfigPath)
	if err != nil {
		return err
	}
	cur, err := config.LoadCuration(curationPath)
	if err != nil {
		return err
	}

	fmt.Printf("Syncing public repos from owners in %s:\n", syncConfigPath)
	for _, o := range cfg.Owners {
		if cfg.AllowsForks(o) {
			fmt.Printf("  - %s  (forks included)\n", o)
		} else {
			fmt.Printf("  - %s\n", o)
		}
	}
	fmt.Println()

	var allRepos []github.Repo
	totalSkippedForks := 0
	totalSkippedParent := 0

	for _, owner := range cfg.Owners {
		repos, skForks, skParent, _, fetchErr := github.FetchRepos(owner, cfg)
		if fetchErr != nil {
			return fmt.Errorf("fetch %s: %w", owner, fetchErr)
		}

		// Sort by pushed date descending within owner.
		sort.Slice(repos, func(i, j int) bool {
			return repos[i].Pushed > repos[j].Pushed
		})

		allRepos = append(allRepos, repos...)
		totalSkippedForks += skForks
		totalSkippedParent += skParent
		fmt.Printf("Fetching public repos from %s...\n  %d repos\n", owner, len(repos))
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	if err := catalog.WriteRawFile(rawPath, allRepos, cur, timestamp); err != nil {
		return err
	}

	fmt.Printf("\nSynced %d public repos to %s\n", len(allRepos), rawPath)
	if totalSkippedForks > 0 {
		fmt.Printf("Filtered out %d drive-by forks (0 commits by owner).\n", totalSkippedForks)
	}
	if totalSkippedParent > 0 {
		fmt.Printf("Filtered out %d forks from excluded parent orgs.\n", totalSkippedParent)
	}

	// Now generate category pages.
	projects := catalog.MergeProjects(allRepos, cur)
	if err := catalog.RenderCategories(categoriesDir, projects); err != nil {
		return err
	}
	if err := catalog.RenderREADME(readmePath, projects); err != nil {
		return err
	}

	fmt.Printf("Generated %d category pages + README (%d cataloged projects).\n", len(catalog.DefaultCategories), len(projects))
	return nil
}

// cmdGenerate reads existing repos-raw.yaml and re-renders category pages.
func cmdGenerate() error {
	cur, err := config.LoadCuration(curationPath)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(rawPath)
	if err != nil {
		return fmt.Errorf("read %s: %w (run 'workindex sync' first)", rawPath, err)
	}

	var raw catalog.RawFile
	if err := parseYAML(data, &raw); err != nil {
		return err
	}

	projects := catalog.MergeProjects(raw.Repos, cur)
	if err := catalog.RenderCategories(categoriesDir, projects); err != nil {
		return err
	}
	if err := catalog.RenderREADME(readmePath, projects); err != nil {
		return err
	}

	fmt.Printf("Generated %d category pages + README (%d cataloged projects).\n", len(catalog.DefaultCategories), len(projects))
	return nil
}

// cmdAdd adds a project to curation.yaml catalog.
func cmdAdd() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: workindex add <org/name> <category> [summary]")
	}
	id := os.Args[2]
	category := os.Args[3]

	if _, ok := catalog.DefaultCategories[category]; !ok {
		valid := make([]string, 0, len(catalog.DefaultCategories))
		for k := range catalog.DefaultCategories {
			valid = append(valid, k)
		}
		sort.Strings(valid)
		return fmt.Errorf("invalid category %q. valid: %s", category, strings.Join(valid, ", "))
	}

	var summary string
	if len(os.Args) > 4 {
		summary = strings.Join(os.Args[4:], " ")
	}

	if err := config.AddCatalogEntry(curationPath, id, category, summary); err != nil {
		return err
	}

	fmt.Printf("Added: %s → %s\n", id, category)
	if summary != "" {
		fmt.Printf("Summary: %s\n", summary)
	}
	fmt.Println("(run 'workindex sync' or 'workindex generate' to update category pages)")
	return nil
}

// cmdExclude adds a repo to the excluded list.
func cmdExclude() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: workindex exclude <org/name>")
	}
	id := os.Args[2]

	if err := config.AddExcluded(curationPath, id); err != nil {
		return err
	}

	fmt.Printf("Excluded: %s\n", id)
	fmt.Println("(run 'workindex sync' to update repos-raw.yaml)")
	return nil
}

// cmdStatus shows sync status and counts.
func cmdStatus() error {
	cur, err := config.LoadCuration(curationPath)
	if err != nil {
		return err
	}

	data, readErr := os.ReadFile(rawPath)
	if readErr != nil {
		fmt.Println("Never synced. Run 'workindex sync' to create repos-raw.yaml.")
		return nil
	}

	var raw catalog.RawFile
	if err := parseYAML(data, &raw); err != nil {
		return err
	}

	included := 0
	excluded := 0
	todo := 0
	unreviewed := 0

	for _, r := range raw.Repos {
		switch cur.ComputeStatus(r.ID()) {
		case "included":
			included++
		case "excluded":
			excluded++
		case "todo":
			todo++
		default:
			unreviewed++
		}
	}

	fmt.Printf("Total repos:  %d\n", len(raw.Repos))
	fmt.Printf("  included:   %d\n", included)
	fmt.Printf("  todo:       %d\n", todo)
	fmt.Printf("  unreviewed: %d\n", unreviewed)
	fmt.Printf("  excluded:   %d\n", excluded)
	return nil
}

// cmdUncurated lists repos that haven't been categorized or excluded.
func cmdUncurated() error {
	cur, err := config.LoadCuration(curationPath)
	if err != nil {
		return err
	}

	data, readErr := os.ReadFile(rawPath)
	if readErr != nil {
		return fmt.Errorf("read %s: %w (run 'workindex sync' first)", rawPath, readErr)
	}

	var raw catalog.RawFile
	if err := parseYAML(data, &raw); err != nil {
		return err
	}

	count := 0
	for _, r := range raw.Repos {
		if cur.ComputeStatus(r.ID()) == "unreviewed" {
			fmt.Printf("  %s/%s  [%s]  %s\n", r.Org, r.Name, r.Language, r.Description)
			count++
		}
	}
	fmt.Printf("\n%d unreviewed repos.\n", count)
	return nil
}

// cmdProfile generates a GitHub profile README, writes it to the personal-readme
// repo, and pushes the change. The personal-readme repo must be a sibling project
// at ../personal-readme/ (added via `camp project add`).
func cmdProfile() error {
	cur, err := config.LoadCuration(curationPath)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(rawPath)
	if err != nil {
		return fmt.Errorf("read %s: %w (run 'workindex sync' first)", rawPath, err)
	}

	var raw catalog.RawFile
	if err := parseYAML(data, &raw); err != nil {
		return err
	}

	projects := catalog.MergeProjects(raw.Repos, cur)

	repoURL := "https://github.com/lancekrogers/work-index"

	// Resolve the personal-readme repo path relative to work-index.
	profileRepoPath := "../personal-readme/README.md"
	if len(os.Args) > 2 {
		profileRepoPath = os.Args[2]
	}

	if err := catalog.RenderProfile(profileRepoPath, repoURL, projects); err != nil {
		return err
	}

	fmt.Printf("Generated profile README (%d projects across categories).\n", len(projects))

	// Commit and push to the personal-readme repo.
	profileDir := "../personal-readme"
	if len(os.Args) > 3 {
		profileDir = os.Args[3]
	}

	gitAdd := exec.Command("git", "-C", profileDir, "add", "README.md")
	if out, err := gitAdd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s: %w", out, err)
	}

	// Check if there are changes to commit.
	gitDiff := exec.Command("git", "-C", profileDir, "diff", "--cached", "--quiet")
	if err := gitDiff.Run(); err == nil {
		fmt.Println("Profile README unchanged — nothing to push.")
		return nil
	}

	gitCommit := exec.Command("git", "-C", profileDir, "commit", "-m", "Sync profile README from work-index catalog")
	if out, err := gitCommit.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %s: %w", out, err)
	}

	gitPush := exec.Command("git", "-C", profileDir, "push")
	if out, err := gitPush.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %s: %w", out, err)
	}

	fmt.Println("Pushed profile README to personal-readme repo.")
	return nil
}

func parseYAML(data []byte, v any) error {
	// Strip comment header lines before parsing.
	lines := strings.Split(string(data), "\n")
	var clean []string
	for _, l := range lines {
		if !strings.HasPrefix(l, "#") {
			clean = append(clean, l)
		}
	}

	return yaml.Unmarshal([]byte(strings.Join(clean, "\n")), v)
}
