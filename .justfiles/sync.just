# Sync recipes for work-index
#
# The sync flow:
#   1. `just sync-repos` pulls all public repos from your GitHub orgs into repos-raw.yaml
#   2. `just uncurated` shows which repos aren't yet in projects.yaml
#   3. You hand-curate into projects.yaml — that's what actually displays

# List sync recipes
[group('sync')]
@sync:
    just --list --justfile {{source_file()}}

# Pull all public, non-fork repos from your GitHub orgs into repos-raw.yaml
[group('sync')]
sync-repos:
    @bash scripts/sync-repos.sh

# Show public repos that aren't yet curated into projects.yaml
[group('sync')]
uncurated:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f repos-raw.yaml ]; then
        echo "repos-raw.yaml not found. Run 'just sync-repos' first."
        exit 1
    fi
    count=$(yq '[.repos[] | select(.in_catalog == false)] | length' repos-raw.yaml)
    echo "Uncurated public repos: ${count}"
    echo ""
    yq -r '.repos[] | select(.in_catalog == false) | "  " + .org + "/" + .name + "  [" + .language + "]  " + .description' repos-raw.yaml

# Show only uncurated repos from a specific org (usage: just uncurated-org lancekrogers)
[group('sync')]
uncurated-org org:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f repos-raw.yaml ]; then
        echo "repos-raw.yaml not found. Run 'just sync-repos' first."
        exit 1
    fi
    yq -r '.repos[] | select(.in_catalog == false and .org == "{{org}}") | "  " + .name + "  [" + .language + "]  " + .description' repos-raw.yaml

# Show sync metadata: when was the raw inventory last updated
[group('sync')]
sync-status:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f repos-raw.yaml ]; then
        echo "Never synced. Run 'just sync-repos' to create repos-raw.yaml."
        exit 0
    fi
    last=$(grep '^# Last synced:' repos-raw.yaml | sed 's/^# Last synced: //')
    total=$(yq '.repos | length' repos-raw.yaml)
    curated=$(yq '[.repos[] | select(.in_catalog == true)] | length' repos-raw.yaml)
    uncurated=$((total - curated))
    echo "Last synced:  ${last}"
    echo "Total repos:  ${total}"
    echo "Curated:      ${curated}"
    echo "Uncurated:    ${uncurated}"
