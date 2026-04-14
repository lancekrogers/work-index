# work-index justfile
# Master catalog of projects across all GitHub organizations

# List available recipes
@default:
    just --list

# Build the workindex binary
build:
    go build -o workindex ./cmd/workindex

# Pull repos from GitHub + render category pages
sync: build
    ./workindex sync

# Render category pages from current data (no GitHub fetch)
generate: build
    ./workindex generate

# Add a project to the catalog (usage: just add lancekrogers/repo devtools "Optional summary")
add id category *summary: build
    ./workindex add {{id}} {{category}} {{summary}}

# Exclude a repo from the catalog (usage: just exclude lancekrogers/repo)
exclude id: build
    ./workindex exclude {{id}}

# Show sync status and counts
status: build
    ./workindex status

# List unreviewed repos
uncurated: build
    ./workindex uncurated

# Generate GitHub profile README with links to catalog
profile: build
    ./workindex profile
