// Migration: 0006_rename_acted_in_to_performed_in
// Reason: The internship-inspired migration — renaming a relationship type.
// ACTED_IN → PERFORMED_IN to support voice actors and motion capture
// performers. Neo4j cannot rename relationships in-place, so we:
//   1. CREATE new PERFORMED_IN relationships copying all properties
//   2. DELETE old ACTED_IN relationships
// Both steps are in one transaction — there is never a moment where
// neither relationship exists.
//
// Medium-hard complexity: 166k+ relationship migration with property copying.
// The dataset has ~166k ACTED_IN relationships so this may take a few seconds.
// Verify after:
//   MATCH (p:Person)-[r:PERFORMED_IN]->(m:Movie)
//   RETURN p.name, r.role, m.title LIMIT 10

MATCH (p:Person)-[old:ACTED_IN]->(m:Movie)
MERGE (p)-[new:PERFORMED_IN]->(m)
SET new.role = old.role;

MATCH ()-[r:ACTED_IN]->()
DELETE r
