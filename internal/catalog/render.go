package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// DefaultCategories returns the canonical category metadata.
var DefaultCategories = map[string]CategoryMeta{
	"ai":                  {Title: "AI", Desc: "AI/LLM tools, agent frameworks, inference infrastructure, and prompt engineering."},
	"backend-infra":       {Title: "Backend & Infrastructure", Desc: "APIs, microservices, databases, cloud infrastructure, and the systems that hold everything together."},
	"blockchain-fintech":  {Title: "Blockchain & Fintech", Desc: "Smart contracts, DeFi protocols, fintech platforms, and web3 tooling."},
	"devtools":            {Title: "Developer Tools", Desc: "CLIs, developer productivity tools, workflow automation, and build systems."},
	"vim-plugins":         {Title: "Vim & Neovim Plugins", Desc: "Editor plugins for Vim and Neovim."},
	"festival-campaigns":  {Title: "Festival Campaigns", Desc: "Public examples of the Festival methodology in action — real campaigns built with camp and fest."},
	"experiments":         {Title: "Experiments & Research", Desc: "Prototypes, explorations, and learning exercises."},
}

const categoryTmpl = `# {{ .Title }}

{{ .Desc }}
{{ if .Content }}
{{ .Content }}
{{ end }}
---

## Projects

| Project | Stack | Created | Summary |
|---------|-------|---------|---------|
{{- range .Projects }}
| [{{ .Name }}]({{ .URL }}) | {{ .Language }} | {{ .Created }} | {{ displaySummary . }} |
{{- end }}
{{- if not .Projects }}
| *No projects yet* | | | |
{{- end }}

---

*{{ len .Projects }} projects in this category. [Back to catalog](../README.md).*
`

var funcMap = template.FuncMap{
	"displaySummary": func(p Project) string {
		s := p.DisplaySummary()
		// Collapse newlines (YAML block scalars leave them).
		s = strings.Join(strings.Fields(s), " ")
		// Escape pipes for markdown table.
		s = strings.ReplaceAll(s, "|", "\\|")
		// Truncate long summaries.
		if len(s) > 120 {
			s = s[:117] + "..."
		}
		return s
	},
}

// RenderCategories writes category .md files from projects.
func RenderCategories(categoriesDir string, projects []Project) error {
	cats := GroupByCategory(projects, DefaultCategories)

	tmpl, err := template.New("category").Funcs(funcMap).Parse(categoryTmpl)
	if err != nil {
		return fmt.Errorf("parse category template: %w", err)
	}

	for i := range cats {
		// Load optional hand-written content from <slug>.content.md.
		// This file is never overwritten by the generator.
		contentPath := filepath.Join(categoriesDir, cats[i].Slug+".content.md")
		if data, err := os.ReadFile(contentPath); err == nil {
			cats[i].Content = strings.TrimSpace(string(data))
		}

		path := filepath.Join(categoriesDir, cats[i].Slug+".md")
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}

		if err := tmpl.Execute(f, cats[i]); err != nil {
			f.Close()
			return fmt.Errorf("render %s: %w", cats[i].Slug, err)
		}
		f.Close()
	}

	return nil
}

// RenderREADME writes the top-level README with category navigation and stats.
func RenderREADME(path string, projects []Project) error {
	cats := GroupByCategory(projects, DefaultCategories)

	var sb strings.Builder
	sb.WriteString("# Lance Rogers — Project Catalog\n\n")
	sb.WriteString("I build software at the intersection of AI, infrastructure, and developer tooling. ")
	sb.WriteString("Over the past year I've shipped 100+ projects across agent frameworks, backend systems, ")
	sb.WriteString("blockchain protocols, and developer tools — most of them finished, all of them real.\n\n")
	sb.WriteString("GitHub only lets you pin 6 repos. This catalog is the rest of the story.\n\n")
	sb.WriteString("---\n\n")

	// Selected work: top projects by priority (set in curation.yaml).
	flagships := topByPriority(projects, 8)
	if len(flagships) > 0 {
		sb.WriteString("## Selected Work\n\n")
		sb.WriteString("| Project | What it is | Stack |\n")
		sb.WriteString("|---------|-----------|-------|\n")
		for _, p := range flagships {
			summary := p.DisplaySummary()
			if len(summary) > 80 {
				summary = summary[:77] + "..."
			}
			fmt.Fprintf(&sb, "| [%s](%s) | %s | %s |\n", p.Name, p.URL, summary, p.Language)
		}
		sb.WriteString("\n> These are representative, not exhaustive. Browse by category below for the full picture.\n\n")
		sb.WriteString("---\n\n")
	}

	// Category navigation.
	sb.WriteString("## Browse by Category\n\n")
	sb.WriteString("| Category | Projects | Description |\n")
	sb.WriteString("|----------|----------|-------------|\n")
	for _, cat := range cats {
		if len(cat.Projects) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "| [%s](categories/%s.md) | %d | %s |\n", cat.Title, cat.Slug, len(cat.Projects), cat.Desc)
	}
	sb.WriteString("\n---\n\n")

	// Stats.
	sb.WriteString("## By the Numbers\n\n")
	fmt.Fprintf(&sb, "- **%d** cataloged projects\n", len(projects))

	langCounts := make(map[string]int)
	for _, p := range projects {
		if p.Language != "" && p.Language != "unknown" {
			langCounts[p.Language]++
		}
	}
	topLangs := topN(langCounts, 5)
	fmt.Fprintf(&sb, "- Primary stacks: %s\n", strings.Join(topLangs, ", "))

	sb.WriteString("\n---\n\n")
	sb.WriteString("## About This Catalog\n\n")
	sb.WriteString("This repository is a structured index of my work. Each project is cataloged in ")
	sb.WriteString("[`curation.yaml`](curation.yaml) with its category, and optionally a curated summary. ")
	sb.WriteString("GitHub metadata (stars, languages, dates) is pulled automatically via [`workindex sync`](cmd/workindex).\n\n")
	sb.WriteString("Category pages are auto-generated. This is the system of record.\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func topByPriority(projects []Project, n int) []Project {
	// Filter to only projects that have an explicit priority > 0.
	var prioritized []Project
	for _, p := range projects {
		if p.Priority > 0 {
			prioritized = append(prioritized, p)
		}
	}
	// Already sorted by priority desc (from MergeProjects), just cap at n.
	if len(prioritized) > n {
		prioritized = prioritized[:n]
	}
	return prioritized
}

func topN(m map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var items []kv
	for k, v := range m {
		items = append(items, kv{k, v})
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].v > items[i].v {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if len(items) > n {
		items = items[:n]
	}
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.k
	}
	return out
}
