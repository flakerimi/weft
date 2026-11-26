package data

import "context"

// Count is how many records the store has. Called with a transaction's
// context it counts inside that transaction.
func (s *Store[T]) Count(ctx context.Context) (int, error) {
	q, d, err := s.ready(ctx)
	if err != nil {
		return 0, err
	}
	var n int
	if err := q.QueryRowContext(ctx, s.t.countSQL(d)).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (t *table) countSQL(d dialect) string {
	return "SELECT count(*) FROM " + d.quote(t.name)
}
