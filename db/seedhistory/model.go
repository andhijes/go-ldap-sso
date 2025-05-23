package seedhistory

import "time"

type SeedHistory struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Batch     int       `db:"batch"`
	AppliedAt time.Time `db:"applied_at"`
}
