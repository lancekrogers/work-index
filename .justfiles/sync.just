# Sync recipes for work-index
#
# Flow:
#   1. `just sync-repos` pulls all public repos from owners in sync-config.yaml
#      into repos-raw.yaml. Each repo gets a computed status:
#        included    — has an entry in projects.yaml
#        excluded    — listed in curation.yaml `excluded:`
#        todo        — listed in curation.yaml `todo:`
#        unreviewed  — default; hasn't been triaged
#   2. Use `just uncurated` to see what needs a decision.
#   3. Use `just exclude <org/name>` to permanently hide junk.
#   4. Use `just mark-todo <org/name>` to shortlist for future curation.
#   5. Curate into projects.yaml when you're ready to write up an entry.
#
# curation.yaml is the persistent store. repos-raw.yaml is regenerated wholesale
# on every sync — do not edit it.

# List sync recipes
[group('sync')]
@sync:
    just --list --justfile {{source_file()}}

# Pull all public, non-fork repos from configured owners into repos-raw.yaml
[group('sync')]
sync-repos:
    @bash scripts/sync-repos.sh

# Show public repos that still need a curation decision
[group('sync')]
uncurated:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f repos-raw.yaml ]; then
        echo "repos-raw.yaml not found. Run 'just sync-repos' first."
        exit 1
    fi
    count=$(yq '[.repos[] | select(.status == "unreviewed")] | length' repos-raw.yaml)
    echo "Unreviewed repos: ${count}"
    echo ""
    yq -r '.repos[] | select(.status == "unreviewed") | "  " + .org + "/" + .name + "  [" + .language + "]  " + .description' repos-raw.yaml

# Show unreviewed repos from a specific org (usage: just uncurated-org lancekrogers)
[group('sync')]
uncurated-org org:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f repos-raw.yaml ]; then
        echo "repos-raw.yaml not found. Run 'just sync-repos' first."
        exit 1
    fi
    yq -r '.repos[] | select(.status == "unreviewed" and .org == "{{org}}") | "  " + .name + "  [" + .language + "]  " + .description' repos-raw.yaml

# Show repos explicitly excluded from the catalog
[group('sync')]
excluded:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f repos-raw.yaml ]; then
        echo "repos-raw.yaml not found. Run 'just sync-repos' first."
        exit 1
    fi
    count=$(yq '[.repos[] | select(.status == "excluded")] | length' repos-raw.yaml)
    echo "Excluded repos: ${count}"
    echo ""
    yq -r '.repos[] | select(.status == "excluded") | "  " + .org + "/" + .name' repos-raw.yaml

# Show repos marked as todo (shortlist for curation into projects.yaml)
[group('sync')]
todo:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f repos-raw.yaml ]; then
        echo "repos-raw.yaml not found. Run 'just sync-repos' first."
        exit 1
    fi
    count=$(yq '[.repos[] | select(.status == "todo")] | length' repos-raw.yaml)
    echo "Todo repos: ${count}"
    echo ""
    yq -r '.repos[] | select(.status == "todo") | "  " + .org + "/" + .name + "  [" + .language + "]  " + .description' repos-raw.yaml

# Append a repo to curation.yaml excluded list (usage: just exclude lancekrogers/nvim-config)
[group('sync')]
exclude id:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f curation.yaml ]; then
        echo "ERROR: curation.yaml not found." >&2
        exit 1
    fi
    if yq -e '.excluded[]? | select(. == "{{id}}")' curation.yaml > /dev/null 2>&1; then
        echo "Already excluded: {{id}}"
        exit 0
    fi
    yq -i '.todo = ((.todo // []) | map(select(. != "{{id}}")))' curation.yaml
    yq -i '.excluded = ((.excluded // []) + ["{{id}}"])' curation.yaml
    echo "Excluded: {{id}}"
    echo "(run 'just sync-repos' to refresh repos-raw.yaml)"

# Append a repo to curation.yaml todo list (usage: just mark-todo lancekrogers/tcount)
[group('sync')]
mark-todo id:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f curation.yaml ]; then
        echo "ERROR: curation.yaml not found." >&2
        exit 1
    fi
    if yq -e '.todo[]? | select(. == "{{id}}")' curation.yaml > /dev/null 2>&1; then
        echo "Already in todo: {{id}}"
        exit 0
    fi
    yq -i '.excluded = ((.excluded // []) | map(select(. != "{{id}}")))' curation.yaml
    yq -i '.todo = ((.todo // []) + ["{{id}}"])' curation.yaml
    echo "Marked as todo: {{id}}"
    echo "(run 'just sync-repos' to refresh repos-raw.yaml)"

# Remove a repo from both curation lists, returning it to unreviewed state
[group('sync')]
unmark id:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f curation.yaml ]; then
        echo "ERROR: curation.yaml not found." >&2
        exit 1
    fi
    yq -i '.excluded = ((.excluded // []) | map(select(. != "{{id}}")))' curation.yaml
    yq -i '.todo = ((.todo // []) | map(select(. != "{{id}}")))' curation.yaml
    echo "Unmarked: {{id}}"
    echo "(run 'just sync-repos' to refresh repos-raw.yaml)"

# Show sync metadata: last sync time + counts by status
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
    included=$(yq '[.repos[] | select(.status == "included")] | length' repos-raw.yaml)
    excluded=$(yq '[.repos[] | select(.status == "excluded")] | length' repos-raw.yaml)
    todo=$(yq '[.repos[] | select(.status == "todo")] | length' repos-raw.yaml)
    unreviewed=$(yq '[.repos[] | select(.status == "unreviewed")] | length' repos-raw.yaml)
    echo "Last synced:  ${last}"
    echo "Total repos:  ${total}"
    echo "  included:   ${included}"
    echo "  todo:       ${todo}"
    echo "  unreviewed: ${unreviewed}"
    echo "  excluded:   ${excluded}"
