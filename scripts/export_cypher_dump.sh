#!/usr/bin/env bash
# export_cypher_dump.sh
# Exports all nodes and relationships from Neo4j as a Cypher script.
#
# Usage: bash scripts/export_cypher_dump.sh <output_file>
#
# Environment variables required:
#   NEO4J_URL, NEO4J_USERNAME, NEO4J_PASSWORD, NEO4J_DATABASE

set -euo pipefail

OUTPUT_FILE="${1:-dumps/dump.cypher}"
mkdir -p "$(dirname "$OUTPUT_FILE")"

HOST=$(echo "$NEO4J_URL" | sed 's|bolt://||' | cut -d: -f1)
HTTP_BASE="http://${HOST}:7474/db/${NEO4J_DATABASE}/tx/commit"
AUTH="${NEO4J_USERNAME}:${NEO4J_PASSWORD}"

run_query() {
  local cypher="$1"
  curl -sf -X POST "$HTTP_BASE" \
    -H "Content-Type: application/json" \
    -u "$AUTH" \
    -d "{\"statements\": [{\"statement\": \"$cypher\"}]}"
}

echo "→ Exporting Cypher dump to $OUTPUT_FILE..."

{
  echo "// GraphMigrate — Cypher Database Export"
  echo "// Generated : $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
  echo "// Database  : $NEO4J_DATABASE"
  echo ""

  echo "// -- CONSTRAINTS --"
  run_query "SHOW CONSTRAINTS YIELD name, createStatement RETURN createStatement" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
for row in rows:
    stmt = row['row'][0]
    if stmt:
        print(stmt + ';')
" 2>/dev/null || echo "// (none)"
  echo ""

  echo "// -- INDEXES --"
  run_query "SHOW INDEXES YIELD name, createStatement, type WHERE type <> 'LOOKUP' RETURN createStatement" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
for row in rows:
    stmt = row['row'][0]
    if stmt:
        print(stmt + ';')
" 2>/dev/null || echo "// (none)"
  echo ""

  echo "// -- NODES --"
  run_query "MATCH (n) RETURN id(n) AS id, labels(n) AS labels, properties(n) AS props" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
for row in rows:
    node_id = row['row'][0]
    labels = ':'.join(row['row'][1])
    props = row['row'][2]
    prop_parts = []
    for k, v in props.items():
        if isinstance(v, str):
            escaped = v.replace(\"'\", \"\\\\''\")
            prop_parts.append(f\"{k}: '{escaped}'\")
        elif isinstance(v, list):
            items = ', '.join([f\"'{x}'\" if isinstance(x, str) else str(x) for x in v])
            prop_parts.append(f'{k}: [{items}]')
        else:
            prop_parts.append(f'{k}: {v}')
    prop_str = '{' + ', '.join(prop_parts) + '}' if prop_parts else ''
    label_str = ':' + labels if labels else ''
    print(f'MERGE (n{node_id}{label_str} {prop_str});')
" 2>/dev/null
  echo ""

  echo "// -- RELATIONSHIPS --"
  run_query "MATCH (a)-[r]->(b) RETURN id(a) AS src, id(b) AS dst, type(r) AS rel_type, properties(r) AS props" \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
rows = data['results'][0]['data'] if data.get('results') else []
for row in rows:
    src = row['row'][0]
    dst = row['row'][1]
    rel_type = row['row'][2]
    props = row['row'][3]
    prop_parts = []
    for k, v in props.items():
        if isinstance(v, str):
            escaped = v.replace(\"'\", \"\\\\''\")
            prop_parts.append(f\"{k}: '{escaped}'\")
        elif isinstance(v, list):
            items = ', '.join([f\"'{x}'\" if isinstance(x, str) else str(x) for x in v])
            prop_parts.append(f'{k}: [{items}]')
        else:
            prop_parts.append(f'{k}: {v}')
    prop_str = ' {' + ', '.join(prop_parts) + '}' if prop_parts else ''
    print(f'MATCH (a) WHERE id(a) = {src} MATCH (b) WHERE id(b) = {dst} MERGE (a)-[:{rel_type}{prop_str}]->(b);')
" 2>/dev/null

  echo ""
  echo "// -- END OF DUMP --"

} > "$OUTPUT_FILE"

LINES=$(wc -l < "$OUTPUT_FILE")
SIZE=$(du -sh "$OUTPUT_FILE" | cut -f1)
echo "✓ Dump written to $OUTPUT_FILE ($LINES lines, $SIZE)"
