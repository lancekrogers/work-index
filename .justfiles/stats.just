# Stats recipes for work-index

# List stats recipes
[group('stats')]
@stats:
    just --list --justfile {{source_file()}}

# Show project counts by category, status, and visibility
[group('stats')]
stats-summary:
    #!/usr/bin/env bash
    set -euo pipefail
    count=$(yq '.projects | length' projects.yaml)
    echo "Total projects: $count"
    echo ""
    echo "By category:"
    yq '.projects[].category' projects.yaml | sort | uniq -c | sort -rn | awk '{printf "  %s: %d\n", $2, $1}'
    echo ""
    echo "By status:"
    yq '.projects[].status' projects.yaml | sort | uniq -c | sort -rn | awk '{printf "  %s: %d\n", $2, $1}'
    echo ""
    echo "By visibility:"
    yq '.projects[].visibility' projects.yaml | sort | uniq -c | sort -rn | awk '{printf "  %s: %d\n", $2, $1}'

# Show all unique stacks/technologies used
[group('stats')]
stats-stacks:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Technology usage:"
    yq '.projects[].stack[]' projects.yaml | sort | uniq -c | sort -rn | awk '{printf "  %s: %d\n", $2, $1}'

# Show all unique tags
[group('stats')]
stats-tags:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Tag usage:"
    yq '.projects[].tags[]' projects.yaml 2>/dev/null | sort | uniq -c | sort -rn | awk '{printf "  %s: %d\n", $2, $1}' || echo "  (no tags yet)"
