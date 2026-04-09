# Generation recipes for work-index

# List generation recipes
[group('generate')]
@generate:
    just --list --justfile {{source_file()}}

# Generate category page project listings from projects.yaml
[group('generate')]
generate-categories:
    @echo "TODO: Generate category pages from projects.yaml"

# Generate README project counts and selected work table
[group('generate')]
generate-readme:
    @echo "TODO: Generate README sections from projects.yaml"

# Generate all derived content
[group('generate')]
generate-all: generate-categories generate-readme
    @echo "All content generated."

# Export public-only projects as JSON
[group('generate')]
export-public:
    @echo "TODO: Export public subset of projects.yaml as JSON"
