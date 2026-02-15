package mysql

import "strings"

// tableFilter allowlists db.table names (lowercase keys). Empty means allow all.
type tableFilter map[string]struct{}

func newTableFilter(tables []string) tableFilter {
	if len(tables) == 0 {
		return nil
	}
	f := make(tableFilter, len(tables))
	for _, t := range tables {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		f[strings.ToLower(t)] = struct{}{}
	}
	return f
}

func (f tableFilter) allow(schema, table string) bool {
	if f == nil {
		return true
	}
	key := strings.ToLower(schema) + "." + strings.ToLower(table)
	_, ok := f[key]
	return ok
}
