package stdlib_test

import "fmt"

var generateSeriesTemplate = `WITH RECURSIVE generate_series(%[3]s)
AS (SELECT cast(%[1]s as integer) FROM RDB$DATABASE UNION ALL SELECT (%[3]s + 1)
FROM generate_series WHERE %[3]s < cast(%[2]s as integer))
SELECT %[3]s FROM generate_series`

func generateSeriesQuery(begin, end, fieldName string) string {
	if len(fieldName) == 0 {
		fieldName = "generate_series"
	}
	return fmt.Sprintf(generateSeriesTemplate, begin, end, fieldName)
}
