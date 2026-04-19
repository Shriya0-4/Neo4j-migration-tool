// Migration: 0001_add_genre_nodes
// Reason: The movies dataset has no genre data. We add Genre nodes and
// wire existing movies to them via IN_GENRE relationships.
// This is graph-idiomatic — genres as nodes lets us query
// "all movies in the same genre" with a single traversal.

MERGE (:Genre {name: "Action"});
MERGE (:Genre {name: "Drama"});
MERGE (:Genre {name: "Sci-Fi"});
MERGE (:Genre {name: "Romance"});
MERGE (:Genre {name: "Thriller"});
MERGE (:Genre {name: "Comedy"});
MERGE (:Genre {name: "Documentary"});

MATCH (m:Movie {title: "The Matrix"}), (g:Genre {name: "Sci-Fi"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "The Matrix"}), (g:Genre {name: "Action"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "The Matrix Reloaded"}), (g:Genre {name: "Sci-Fi"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "The Matrix Reloaded"}), (g:Genre {name: "Action"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "The Matrix Revolutions"}), (g:Genre {name: "Sci-Fi"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "The Devil's Advocate"}), (g:Genre {name: "Thriller"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "A Few Good Men"}), (g:Genre {name: "Drama"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Top Gun"}), (g:Genre {name: "Action"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Jerry Maguire"}), (g:Genre {name: "Drama"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Jerry Maguire"}), (g:Genre {name: "Romance"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Cast Away"}), (g:Genre {name: "Drama"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Apollo 13"}), (g:Genre {name: "Drama"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Sleepless in Seattle"}), (g:Genre {name: "Romance"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Sleepless in Seattle"}), (g:Genre {name: "Comedy"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "You've Got Mail"}), (g:Genre {name: "Romance"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Cloud Atlas"}), (g:Genre {name: "Sci-Fi"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Cloud Atlas"}), (g:Genre {name: "Drama"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "V for Vendetta"}), (g:Genre {name: "Action"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "Speed Racer"}), (g:Genre {name: "Action"})
MERGE (m)-[:IN_GENRE]->(g);

MATCH (m:Movie {title: "The Da Vinci Code"}), (g:Genre {name: "Thriller"})
MERGE (m)-[:IN_GENRE]->(g)
