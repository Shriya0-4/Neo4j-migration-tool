#!/usr/bin/env bash
# generate_report.sh
# Generates a detailed plain-text migration audit report by:
#   1. Reading pre/post migration snapshots
#   2. Querying Neo4j for migration history nodes
#   3. Reading the migration output log
#   4. Diffing the snapshots
#   5. Writing everything into a single timestamped report file
#
# Environment variables required:
#   NEO4J_URL, NEO4J_USERNAME, NEO4J_PASSWORD, NEO4J_DATABASE
#   GIT_SHA, GIT_REF, RUN_ID, TRIGGERED_BY
#   Optional: DRY_RUN, ROLLBACK_TO

set -euo pipefail

TIMESTAMP=$(date -u '+%Y%m%d_%H%M%S')
REPORT_DIR="reports"
REPORT_FILE="$REPORT_DIR/migration_report_${TIMESTAMP}.txt"

mkdir -p "$REPORT_DIR"

HOST=$(echo "$NEO4J_URL" | sed 's|bolt://||' | cut -d: -f1)
HTTP_BASE="http://${HOST}:7474/db/${NEO4J_DATABASE:-neo4j}/tx/commit"
AUTH="${NEO4J_USERNAME}:${NEO4J_PASSWORD}"

run_query() {
  local cypher="$1"
  curl -sf -X POST "$HTTP_BASE" \
    -H "Content-Type: application/json" \
    -u "$AUTH" \
    -d "{\"statements\": [{\"statement\": \"$cypher\"}]}" 2>/dev/null || echo '{"results":[]}'
}

echo "→ Generating migration report..."

{
# ══════════════════════════════════════════════════════════════════════════════
cat << 'HEADER'
================================================================================
                        GRAPHMIGRATE — MIGRATION AUDIT REPORT
================================================================================
HEADER

echo ""
echo "REPORT METADATA"
echo "---------------"
echo "  Generated At   : $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo "  Git SHA        : ${GIT_SHA:-unknown}"
echo "  Git Ref        : ${GIT_REF:-unknown}"
echo "  Run ID         : ${RUN_ID:-unknown}"
echo "  Triggered By   : ${TRIGGERED_BY:-unknown}"
echo "  Dry Run        : ${DRY_RUN:-false}"
echo "  Rollback To    : ${ROLLBACK_TO:-(not a rollback run)}"
echo "  Neo4j Database : ${NEO4J_DATABASE:-neo4j}"
echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " SECTION 1 — MIGRATION EXECUTION LOG"
echo "================================================================================"
echo ""

if [ -f "migration_output.log" ]; then
  cat migration_output.log
else
  echo "  (migration_output.log not found)"
fi
echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " SECTION 2 — MIGRATION STATUS (POST-RUN)"
echo "================================================================================"
echo ""

if [ -f "migration_status.log" ]; then
  cat migration_status.log
else
  echo "  (migration_status.log not found)"
fi
echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " SECTION 3 — APPLIED MIGRATION HISTORY (from Neo4j)"
echo "================================================================================"
echo ""

run_query "MATCH (m:SchemaMigration) RETURN m.version AS version, m.name AS name, m.appliedAt AS appliedAt, m.checksum AS checksum ORDER BY m.version" \
  | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
if not rows:
    print('  No SchemaMigration nodes found.')
    print('  This could mean migrations were run with --dry-run,')
    print('  or the SchemaMigration nodes are in a different database.')
else:
    print(f'  Total applied: {len(rows)}')
    print()
    print(f'  {\"VERSION\":<10} {\"NAME\":<45} {\"APPLIED AT\":<25} {\"CHECKSUM\"}')
    print(f'  {\"-\"*10} {\"-\"*45} {\"-\"*25} {\"-\"*32}')
    for row in rows:
        r = row['row']
        version = str(r[0]).zfill(4) if r[0] is not None else 'N/A'
        name = str(r[1]) if r[1] else 'N/A'
        applied_at = str(r[2]) if r[2] else 'N/A'
        checksum = str(r[3])[:32] if r[3] else 'N/A'
        print(f'  {version:<10} {name:<45} {applied_at:<25} {checksum}')
" 2>/dev/null || echo "  (query failed — Neo4j may not be reachable)"
echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " SECTION 4 — DATABASE STATE CHANGES (Pre vs Post Migration)"
echo "================================================================================"
echo ""

PRE="snapshots/pre_migration.txt"
POST="snapshots/post_migration.txt"

if [ ! -f "$PRE" ] || [ ! -f "$POST" ]; then
  echo "  Snapshot files not found — skipping diff."
  echo "  Expected: $PRE and $POST"
else

  # ── Node count changes ──────────────────────────────────────────────────────
  echo "  NODE COUNT CHANGES"
  echo "  ------------------"

  python3 << 'PYEOF'
import re

def parse_section(filepath, section_header):
    """Extract lines from a section of a snapshot file."""
    lines = []
    in_section = False
    try:
        with open(filepath) as f:
            for line in f:
                if section_header in line:
                    in_section = True
                    continue
                if in_section:
                    if line.strip().startswith('---') or (line.strip() and all(c == '-' for c in line.strip())):
                        continue
                    if line.strip() == '' and lines:
                        break
                    if line.strip():
                        lines.append(line.rstrip())
    except FileNotFoundError:
        pass
    return lines

def parse_counts(lines):
    counts = {}
    for line in lines:
        parts = line.strip().rsplit(None, 1)
        if len(parts) == 2:
            try:
                counts[parts[0].strip("[]' ")] = int(parts[1])
            except ValueError:
                pass
    return counts

pre_nodes = parse_counts(parse_section('snapshots/pre_migration.txt', 'NODE COUNTS BY LABEL'))
post_nodes = parse_counts(parse_section('snapshots/post_migration.txt', 'NODE COUNTS BY LABEL'))

all_labels = set(list(pre_nodes.keys()) + list(post_nodes.keys()))
changes = []

for label in sorted(all_labels):
    before = pre_nodes.get(label, 0)
    after = post_nodes.get(label, 0)
    diff = after - before
    if diff != 0:
        sign = '+' if diff > 0 else ''
        changes.append(f'  {label:<35}  {before:>6} → {after:>6}  ({sign}{diff})')
    else:
        changes.append(f'  {label:<35}  {before:>6} → {after:>6}  (no change)')

if changes:
    for c in changes:
        print(c)
else:
    print('  (no label data found in snapshots)')
PYEOF

  echo ""

  # ── Relationship count changes ──────────────────────────────────────────────
  echo "  RELATIONSHIP COUNT CHANGES"
  echo "  --------------------------"

  python3 << 'PYEOF'
def parse_section(filepath, section_header):
    lines = []
    in_section = False
    try:
        with open(filepath) as f:
            for line in f:
                if section_header in line:
                    in_section = True
                    continue
                if in_section:
                    if line.strip().startswith('---') or (line.strip() and all(c == '-' for c in line.strip())):
                        continue
                    if line.strip() == '' and lines:
                        break
                    if line.strip():
                        lines.append(line.rstrip())
    except FileNotFoundError:
        pass
    return lines

def parse_counts(lines):
    counts = {}
    for line in lines:
        parts = line.strip().rsplit(None, 1)
        if len(parts) == 2:
            try:
                counts[parts[0].strip()] = int(parts[1])
            except ValueError:
                pass
    return counts

pre_rels = parse_counts(parse_section('snapshots/pre_migration.txt', 'RELATIONSHIP COUNTS BY TYPE'))
post_rels = parse_counts(parse_section('snapshots/post_migration.txt', 'RELATIONSHIP COUNTS BY TYPE'))

all_types = set(list(pre_rels.keys()) + list(post_rels.keys()))

for rel_type in sorted(all_types):
    before = pre_rels.get(rel_type, 0)
    after = post_rels.get(rel_type, 0)
    diff = after - before
    sign = '+' if diff > 0 else ''

    if before > 0 and after == 0:
        status = '  ← REMOVED'
    elif before == 0 and after > 0:
        status = '  ← ADDED'
    elif diff != 0:
        status = f'  ({sign}{diff})'
    else:
        status = '  (no change)'

    print(f'  {rel_type:<35}  {before:>6} → {after:>6}{status}')
PYEOF

  echo ""

  # ── Total counts ────────────────────────────────────────────────────────────
  echo "  TOTAL COUNTS SUMMARY"
  echo "  --------------------"
  python3 << 'PYEOF'
import re

def get_total(filepath, section_header):
    in_section = False
    try:
        with open(filepath) as f:
            for line in f:
                if section_header in line:
                    in_section = True
                    continue
                if in_section:
                    stripped = line.strip()
                    if stripped:
                        try:
                            return int(stripped)
                        except ValueError:
                            return stripped
    except FileNotFoundError:
        pass
    return 'N/A'

pre_nodes  = get_total('snapshots/pre_migration.txt', 'TOTAL NODES')
post_nodes = get_total('snapshots/post_migration.txt', 'TOTAL NODES')
pre_rels   = get_total('snapshots/pre_migration.txt', 'TOTAL RELATIONSHIPS')
post_rels  = get_total('snapshots/post_migration.txt', 'TOTAL RELATIONSHIPS')

def fmt_diff(before, after):
    try:
        d = int(after) - int(before)
        sign = '+' if d > 0 else ''
        return f'{before} → {after}  ({sign}{d})'
    except:
        return f'{before} → {after}'

print(f'  Total nodes         : {fmt_diff(pre_nodes, post_nodes)}')
print(f'  Total relationships : {fmt_diff(pre_rels, post_rels)}')
PYEOF

  echo ""

  # ── Constraint changes ──────────────────────────────────────────────────────
  echo "  CONSTRAINT CHANGES"
  echo "  ------------------"
  python3 << 'PYEOF'
def parse_section(filepath, section_header):
    lines = []
    in_section = False
    try:
        with open(filepath) as f:
            for line in f:
                if section_header in line:
                    in_section = True
                    continue
                if in_section:
                    if line.strip().startswith('---') or (line.strip() and all(c == '-' for c in line.strip())):
                        continue
                    if line.strip() == '' and lines:
                        break
                    if line.strip():
                        lines.append(line.strip())
    except FileNotFoundError:
        pass
    return lines

pre_constraints  = set(parse_section('snapshots/pre_migration.txt', 'ACTIVE CONSTRAINTS'))
post_constraints = set(parse_section('snapshots/post_migration.txt', 'ACTIVE CONSTRAINTS'))

added   = post_constraints - pre_constraints
removed = pre_constraints - post_constraints

if not added and not removed:
    print('  No constraint changes.')
for c in sorted(added):
    print(f'  ADDED   : {c}')
for c in sorted(removed):
    print(f'  REMOVED : {c}')
PYEOF

  echo ""

  # ── Property key changes ────────────────────────────────────────────────────
  echo "  PROPERTY KEY CHANGES"
  echo "  --------------------"
  python3 << 'PYEOF'
def get_props(filepath):
    try:
        with open(filepath) as f:
            for line in f:
                if 'PROPERTY KEYS IN USE' in line:
                    next(f)  # skip dashes
                    val_line = next(f, '').strip()
                    if val_line and val_line != '(none)':
                        return set(k.strip() for k in val_line.split(','))
    except:
        pass
    return set()

pre_props  = get_props('snapshots/pre_migration.txt')
post_props = get_props('snapshots/post_migration.txt')

added   = post_props - pre_props
removed = pre_props - post_props

if not added and not removed:
    print('  No property key changes.')
if added:
    print(f'  ADDED   : {", ".join(sorted(added))}')
if removed:
    print(f'  REMOVED : {", ".join(sorted(removed))}')
PYEOF

fi  # end if snapshots exist

echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " SECTION 5 — PRE-MIGRATION SNAPSHOT"
echo "================================================================================"
echo ""
if [ -f "snapshots/pre_migration.txt" ]; then
  cat snapshots/pre_migration.txt
else
  echo "  (not found)"
fi
echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " SECTION 6 — POST-MIGRATION SNAPSHOT"
echo "================================================================================"
echo ""
if [ -f "snapshots/post_migration.txt" ]; then
  cat snapshots/post_migration.txt
else
  echo "  (not found)"
fi
echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " SECTION 7 — MIGRATION FILE INVENTORY"
echo "================================================================================"
echo ""

echo "  Files found in migrations/ directory:"
echo ""
if [ -d "migrations" ]; then
  for f in migrations/*.cypher; do
    [ -f "$f" ] || continue
    filename=$(basename "$f")
    lines=$(wc -l < "$f")
    size=$(du -sh "$f" | cut -f1)
    if [[ "$filename" == *.down.cypher ]]; then
      echo "    [DOWN] $filename  ($lines lines, $size)"
    else
      echo "    [ UP ] $filename  ($lines lines, $size)"
    fi
  done
else
  echo "  (migrations/ directory not found)"
fi
echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " SECTION 8 — ARTIFACT INVENTORY"
echo "================================================================================"
echo ""
echo "  The following artifacts were uploaded for this run:"
echo ""
echo "    migration-report-${RUN_ID:-?}"
echo "      └── reports/migration_report_${TIMESTAMP}.txt   (this file)"
echo ""
echo "    neo4j-dumps-${RUN_ID:-?}"
echo "      ├── dumps/pre_migration_dump.cypher"
echo "      └── dumps/post_migration_dump.cypher"
echo ""
echo "    db-snapshots-${RUN_ID:-?}"
echo "      ├── snapshots/pre_migration.txt"
echo "      └── snapshots/post_migration.txt"
echo ""
echo "    migration-logs-${RUN_ID:-?}"
echo "      ├── migration_output.log"
echo "      └── migration_status.log"
echo ""

# ══════════════════════════════════════════════════════════════════════════════
echo "================================================================================"
echo " END OF REPORT"
echo "================================================================================"

} > "$REPORT_FILE"

echo "✓ Report written to $REPORT_FILE"
echo ""
echo "── Report preview (first 30 lines) ──────────────────────"
head -30 "$REPORT_FILE"
echo "..."
echo "──────────────────────────────────────────────────────────"
