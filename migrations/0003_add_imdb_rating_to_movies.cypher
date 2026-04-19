// Migration: 0003_add_computed_rating_band
// Reason: imdbRating, released, tagline, year indexes already exist in the
// dataset. What's missing is a computed classification property that lets
// the app do simple faceted filtering without range queries.
// We add imdbRatingBand ('low'/'mid'/'high') and an index on it.
// Also adds a countries index — no existing index covers Movie.countries.

MATCH (m:Movie)
WHERE m.imdbRating IS NOT NULL
SET m.imdbRatingBand =
  CASE
    WHEN m.imdbRating >= 8.0 THEN 'high'
    WHEN m.imdbRating >= 6.0 THEN 'mid'
    ELSE 'low'
  END;

CREATE INDEX movie_rating_band_idx IF NOT EXISTS
FOR (m:Movie) ON (m.imdbRatingBand);

CREATE INDEX movie_countries_idx IF NOT EXISTS
FOR (m:Movie) ON (m.countries)
