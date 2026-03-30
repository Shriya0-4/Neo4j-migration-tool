package cypher

// all the migration related cypher queries

func Testquery() string {
	query := "MATCH(n) RETURN n LIMIT 5"
	return query
}
