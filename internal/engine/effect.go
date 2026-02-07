package engine

type Effect interface {
	Next() (string, bool)
	CanvasHeight() int
	CanvasWidth() int
}
