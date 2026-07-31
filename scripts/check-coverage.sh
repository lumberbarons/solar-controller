#!/usr/bin/env bash
#
# Enforces both a per-package coverage floor and an overall total.
#
# A total-only gate lets a badly tested package hide behind well tested ones:
# internal/controllers/epever holds most of the statements in this module, so a
# change that guts its coverage can still raise the total by adding a small,
# heavily tested package. The per-package floors in scripts/coverage-floors.txt
# close that gap.
#
# Usage: check-coverage.sh <coverage-profile> <floors-file> <total-threshold>

set -euo pipefail

profile="${1:?usage: check-coverage.sh <coverage-profile> <floors-file> <total-threshold>}"
floors="${2:?missing floors file}"
total_threshold="${3:?missing total threshold}"

for f in "$profile" "$floors"; do
  if [ ! -f "$f" ]; then
    echo "check-coverage: $f not found" >&2
    exit 2
  fi
done

module="$(go list -m)"

# Packages deliberately outside the gate:
#   internal/testutil            - test helpers, exercised only via other tests
#   site/node_modules            - third-party Go code vendored inside an npm
#                                  package; ./... picks it up but it is not ours
awk \
  -v module_prefix="${module}/" \
  -v floors_file="$floors" \
  -v total_threshold="$total_threshold" '
function is_excluded(pkg) {
  return pkg ~ /^internal\/testutil/ || pkg ~ /^site\/node_modules/
}

BEGIN {
  while ((getline line < floors_file) > 0) {
    sub(/#.*/, "", line)
    if (split(line, field, " ") < 2) continue
    floor[field[1]] = field[2]
    listed[field[1]] = 1
  }
  close(floors_file)
}

# Skip the profile header, e.g. "mode: set"
/^mode:/ { next }

{
  split($1, location, ":")
  path = location[1]
  sub(module_prefix, "", path)

  # Package is the file path minus the file name
  depth = split(path, segment, "/")
  pkg = segment[1]
  for (i = 2; i < depth; i++) pkg = pkg "/" segment[i]

  if (is_excluded(pkg)) next

  statements[pkg] += $2
  if ($3 > 0) covered[pkg] += $2
  seen[pkg] = 1
}

END {
  # A listed package with no profile entries has no tests at all, so it counts
  # as 0% rather than disappearing from the report.
  for (pkg in floor) if (!(pkg in seen)) { seen[pkg] = 1; statements[pkg] = 0; covered[pkg] = 0 }

  count = 0
  for (pkg in seen) ordered[++count] = pkg
  for (i = 1; i <= count; i++)
    for (j = i + 1; j <= count; j++)
      if (ordered[j] < ordered[i]) { swap = ordered[i]; ordered[i] = ordered[j]; ordered[j] = swap }

  printf "%-40s %8s %8s\n", "PACKAGE", "ACTUAL", "FLOOR"

  failures = 0
  total_statements = 0
  total_covered = 0

  for (i = 1; i <= count; i++) {
    pkg = ordered[i]
    pct = statements[pkg] > 0 ? 100 * covered[pkg] / statements[pkg] : 0
    total_statements += statements[pkg]
    total_covered += covered[pkg]

    if (!(pkg in listed)) {
      printf "%-40s %7.1f%% %8s  UNLISTED\n", pkg, pct, "-"
      unlisted[++unlisted_count] = pkg
      failures++
      continue
    }

    # Compared at one decimal place, matching the reported figure, so a package
    # is never failed for a difference the report cannot show.
    if (sprintf("%.1f", pct) + 0 < floor[pkg] + 0) {
      printf "%-40s %7.1f%% %7d%%  BELOW FLOOR\n", pkg, pct, floor[pkg]
      failures++
    } else {
      printf "%-40s %7.1f%% %7d%%\n", pkg, pct, floor[pkg]
    }
  }

  total_pct = total_statements > 0 ? 100 * total_covered / total_statements : 0
  printf "%-40s %7.1f%% %7d%%\n", "TOTAL", total_pct, total_threshold

  total_failed = (sprintf("%.1f", total_pct) + 0 < total_threshold + 0)

  if (!failures && !total_failed) exit 0

  print ""
  if (unlisted_count > 0) {
    for (i = 1; i <= unlisted_count; i++)
      printf "FAIL: %s is not in %s. Add it with a floor equal to its current coverage.\n", unlisted[i], floors_file
  }
  for (i = 1; i <= count; i++) {
    pkg = ordered[i]
    if (!(pkg in listed)) continue
    pct = statements[pkg] > 0 ? 100 * covered[pkg] / statements[pkg] : 0
    if (sprintf("%.1f", pct) + 0 < floor[pkg] + 0)
      printf "FAIL: %s coverage %.1f%% is below its floor of %d%%.\n", pkg, pct, floor[pkg]
  }
  if (total_failed)
    printf "FAIL: total coverage %.1f%% is below the threshold of %d%%.\n", total_pct, total_threshold

  exit 1
}
' "$profile"
