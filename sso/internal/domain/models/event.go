package models

type Event struct {
	ID      int64  `db:"id"`
	Type    string `db:"event_type"`
	Payload string `db:"payload"`
}
