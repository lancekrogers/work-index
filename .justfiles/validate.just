# Validation recipes for work-index

# List validation recipes
[group('validate')]
@validate:
    just --list --justfile {{source_file()}}

# Validate projects.yaml required fields and enum values
[group('validate')]
validate-yaml:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Validating projects.yaml..."
    count=$(yq '.projects | length' projects.yaml)
    if [ "$count" = "0" ] || [ "$count" = "null" ]; then
        echo "  OK: 0 projects (empty catalog)"
        exit 0
    fi
    errors=0
    # Check required fields
    for field in name slug category status stack repo visibility org summary; do
        missing=$(yq ".projects[] | select(.$field == null) | .name // .slug // \"unknown\"" projects.yaml)
        if [ -n "$missing" ]; then
            echo "  ERROR: missing '$field' in: $missing"
            errors=1
        fi
    done
    # Check category enum
    bad_cat=$(yq '.projects[] | select(.category != "ai-agents" and .category != "backend-infra" and .category != "blockchain-fintech" and .category != "devtools-workflows" and .category != "experiments") | .name + ": " + .category' projects.yaml)
    if [ -n "$bad_cat" ]; then
        echo "  ERROR: invalid category: $bad_cat"
        errors=1
    fi
    # Check status enum
    bad_status=$(yq '.projects[] | select(.status != "active" and .status != "maintained" and .status != "complete" and .status != "archived" and .status != "experiment") | .name + ": " + .status' projects.yaml)
    if [ -n "$bad_status" ]; then
        echo "  ERROR: invalid status: $bad_status"
        errors=1
    fi
    # Check visibility enum
    bad_vis=$(yq '.projects[] | select(.visibility != "public" and .visibility != "private" and .visibility != "org-only") | .name + ": " + .visibility' projects.yaml)
    if [ -n "$bad_vis" ]; then
        echo "  ERROR: invalid visibility: $bad_vis"
        errors=1
    fi
    # Check duplicate slugs
    dupes=$(yq '.projects[].slug' projects.yaml | sort | uniq -d)
    if [ -n "$dupes" ]; then
        echo "  ERROR: duplicate slugs: $dupes"
        errors=1
    fi
    if [ "$errors" = "0" ]; then
        echo "  OK: $count projects validated"
    else
        exit 1
    fi

# Check that deep-dive files exist for projects with has_deep_dive: true
[group('validate')]
validate-deep-dives:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Checking deep-dive file references..."
    errors=0
    for slug in $(yq '.projects[] | select(.has_deep_dive == true) | .slug' projects.yaml 2>/dev/null); do
        if [ ! -f "projects/$slug.md" ]; then
            echo "  MISSING: projects/$slug.md"
            errors=1
        fi
    done
    if [ "$errors" = "0" ]; then
        echo "  OK: all deep-dive files exist"
    fi
    exit $errors

# Run all validations
[group('validate')]
validate-all: validate-yaml validate-deep-dives
    @echo "All validations passed."
