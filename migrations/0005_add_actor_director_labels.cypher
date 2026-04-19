// Migration: 0005_add_actor_director_labels
// Reason: Ensure ALL persons with ACTED_IN/DIRECTED/WROTE/PRODUCED
// relationships carry the correct secondary label. The dataset applies
// these inconsistently. SET p:Actor on an already-labelled node is a no-op.
// Also adds indexes on Actor.name and Director.name — no existing index
// covers these label-specific queries (index_5c0607ad covers Person.name
// but NOT Actor.name or Director.name separately).

MATCH (p:Person)-[:ACTED_IN]->(:Movie)
SET p:Actor;

MATCH (p:Person)-[:DIRECTED]->(:Movie)
SET p:Director;

MATCH (p:Person)-[:WROTE]->(:Movie)
SET p:Writer;

MATCH (p:Person)-[:PRODUCED]->(:Movie)
SET p:Producer;

CREATE INDEX actor_name_idx IF NOT EXISTS
FOR (p:Actor) ON (p.name);

CREATE INDEX director_name_idx IF NOT EXISTS
FOR (p:Director) ON (p.name)
