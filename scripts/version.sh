#!/bin/sh
# Succeed when $1 is a release version like 0.0.1 or 0.1.0-alpha.1, and
# nothing else: one line of digits, dots, lowercase letters and hyphens.
# Usage: scripts/version.sh VERSION
case "${1:-}" in '' | *[!0-9a-z.-]*) exit 1 ;; esac
printf '%s\n' "$1" | grep -Eqx '[0-9]+\.[0-9]+\.[0-9]+(-[a-z0-9.]+)?'
