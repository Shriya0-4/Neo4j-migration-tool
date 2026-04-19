#!/usr/bin/env bash
# snapshot.sh
# Captures a detailed snapshot of the Neo4j database state to a text file.
# Used to diff pre/post migration state in the report.
#
# Usage: bash scripts/snapshot.sh <label> <output_file>
#   label       : "pre-migration" or "post-migration"
#   output_file : path to write the snapshot (e.g. snapshots/pre_migration.txt)
#
# Environment variables required:
#   NEO4J_URL, NEO4J_USERNAME, NEO4J_PASSWORD, NEO4J_DATABASE

set -euo pipefail

LABEL="${1:-snapshot}"
OUTPUT_FILE="${2:-snapshots/snapshot.txt}"

mkdir -p "$(dirname "$OUTPUT_FILE")"

HOST=$(echo "$NEO4J_URL" | sed 's|bolt://||' | cut -d: -f1)
HTTP_BASE="http://${HOST}:7474/db/${NEO4J_DATABASE}/tx/commit"
AUTH="${NEO4J_USERNAME}:${NEO4J_PASSWORD}"

# Helper: run a cypher query and return raw JSON result
run_query() {
  local cypher="$1"
  curl -sf -X POST "$HTTP_BASE" \
    -H "Content-Type: application/json" \
    -u "$AUTH" \
    -d "{\"statements\": [{\"statement\": \"$cypher\"}]}"
}

# Helper: extract a value from JSON using python (available on ubuntu runners)
extract() {
  python3 -c "import sys,json; data=json.load(sys.stdin); $1" 2>/dev/null || echo "N/A"
}

echo "→ Capturing $LABEL snapshot..."

{
  echo "============================================================"
  echo " Neo4j Database Snapshot"
  echo " Label    : $LABEL"
  echo " Captured : $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
  echo " Database : $NEO4J_DATABASE"
  echo "============================================================"
  echo ""

  # ── Node counts by label ─────────────────────────────────────
  echo "NODE COUNTS BY LABEL"
  echo "--------------------"
  run_query "MATCH (n) RETURN labels(n) AS label, count(n) AS count ORDER BY count DESC" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
for row in rows:
    label = row['row'][0]
    count = row['row'][1]
    print(f'  {str(label):<35} {count}')
" 2>/dev/null || echo "  (query failed)"
  echo ""

  # ── Total node count ─────────────────────────────────────────
  echo "TOTAL NODES"
  echo "-----------"
  run_query "MATCH (n) RETURN count(n) AS total" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
total = data['results'][0]['data'][0]['row'][0] if data.get('results') else 'N/A'
print(f'  {total}')
" 2>/dev/null || echo "  (query failed)"
  echo ""

  # ── Relationship counts by type ───────────────────────────────
  echo "RELATIONSHIP COUNTS BY TYPE"
  echo "---------------------------"
  run_query "MATCH ()-[r]->() RETURN type(r) AS rel_type, count(r) AS count ORDER BY count DESC" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
for row in rows:
    print(f'  {row[\"row\"][0]:<35} {row[\"row\"][1]}')
" 2>/dev/null || echo "  (query failed)"
  echo ""

  # ── Total relationship count ──────────────────────────────────
  echo "TOTAL RELATIONSHIPS"
  echo "-------------------"
  run_query "MATCH ()-[r]->() RETURN count(r) AS total" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
total = data['results'][0]['data'][0]['row'][0] if data.get('results') else 'N/A'
print(f'  {total}')
" 2>/dev/null || echo "  (query failed)"
  echo ""

  # ── Schema constraints ────────────────────────────────────────
  echo "ACTIVE CONSTRAINTS"
  echo "------------------"
  run_query "SHOW CONSTRAINTS YIELD name, type, labelsOrTypes, properties RETURN name, type, labelsOrTypes, properties" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
if not rows:
    print('  (none)')
for row in rows:
    r = row['row']
    print(f'  {r[0]:<45} type={r[1]}  labels={r[2]}  props={r[3]}')
" 2>/dev/null || echo "  (query failed)"
  echo ""

  # ── Schema indexes ────────────────────────────────────────────
  echo "ACTIVE INDEXES"
  echo "--------------"
  run_query "SHOW INDEXES YIELD name, type, labelsOrTypes, properties, state RETURN name, type, labelsOrTypes, properties, state" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
if not rows:
    print('  (none)')
for row in rows:
    r = row['row']
    print(f'  {r[0]:<40} type={r[1]}  state={r[4]}  labels={r[2]}  props={r[3]}')
" 2>/dev/null || echo "  (query failed)"
  echo ""

  # ── Applied migrations ────────────────────────────────────────
  echo "APPLIED MIGRATIONS (SchemaMigration nodes)"
  echo "------------------------------------------"
  run_query "MATCH (m:SchemaMigration) RETURN m.version, m.name, m.appliedAt, m.checksum ORDER BY m.version" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
if not rows:
    print('  (none — migrations not yet applied)')
for row in rows:
    r = row['row']
    print(f'  v{str(r[0]).zfill(4)}  {str(r[1]):<45}  applied={r[2]}')
" 2>/dev/null || echo "  (query failed)"
  echo ""

  # ── Property keys in use ──────────────────────────────────────
  echo "PROPERTY KEYS IN USE"
  echo "--------------------"
  run_query "CALL db.propertyKeys() YIELD propertyKey RETURN propertyKey ORDER BY propertyKey" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
keys = [row['row'][0] for row in rows]
print('  ' + ', '.join(keys) if keys else '  (none)')
" 2>/dev/null || echo "  (query failed)"
  echo ""

  echo "============================================================"
  echo " End of snapshot"
  echo "============================================================"

} > "$OUTPUT_FILE"

echo "✓ Snapshot written to $OUTPUT_FILE"
