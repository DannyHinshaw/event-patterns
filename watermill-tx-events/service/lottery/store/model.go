package store

type Winner struct {
	ID   int    `db:"id,pk"`
	Name string `db:"name"`
}
