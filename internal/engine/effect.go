package engine

type Effect interface {
	Next() (string, bool)
}
