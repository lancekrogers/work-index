# work-index justfile
# Master catalog of projects across all GitHub organizations

import '.justfiles/generate.just'
import '.justfiles/sync.just'
import '.justfiles/validate.just'
import '.justfiles/stats.just'

# List available recipes
@default:
    just --list
