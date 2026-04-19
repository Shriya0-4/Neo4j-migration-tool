// Migration: 0004_add_release_decade
// Reason: Add a computed releaseDecade property for decade-level grouping.
// NOTE: In this dataset `released` is stored as a STRING (e.g. "1999"),
// so we use toInteger() to cast it before the arithmetic.

MATCH (m:Movie)
WHERE m.released IS NOT NULL
SET m.releaseDecade = (toString((toInteger(m.released) / 10) * 10) + 's');

CREATE INDEX movie_release_decade_idx IF NOT EXISTS
FOR (m:Movie) ON (m.releaseDecade)
