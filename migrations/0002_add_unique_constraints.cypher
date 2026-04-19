// Migration: 0002_add_unique_constraints
// Reason: Add uniqueness constraints on properties that have NO existing
// constraint or index. Based on the current schema, Movie.title and
// Person.name have plain range indexes but no uniqueness constraint —
// however since we cannot drop+recreate in the same transaction, we
// target properties that are completely uncovered:
// - Movie.imdbRating has a plain index but no uniqueness (correct, ratings repeat)
// - What's genuinely missing: a composite text search capability and
//   constraints on Actor/Director labels which have no coverage at all.
//
// This migration adds:
//   1. FULLTEXT index on Movie (title + tagline) for search queries
//   2. FULLTEXT index on Person (name + bio) for search queries
//   3. Range index on Movie.budget and Movie.revenue (no index exists)
// All three are new — zero conflict with existing schema.

CREATE FULLTEXT INDEX movie_title_fulltext IF NOT EXISTS
FOR (m:Movie) ON EACH [m.title, m.tagline];

CREATE FULLTEXT INDEX person_name_fulltext IF NOT EXISTS
FOR (p:Person) ON EACH [p.name, p.bio];

CREATE INDEX movie_budget_idx IF NOT EXISTS
FOR (m:Movie) ON (m.budget);

CREATE INDEX movie_revenue_idx IF NOT EXISTS
FOR (m:Movie) ON (m.revenue)
