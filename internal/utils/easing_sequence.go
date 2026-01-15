package utils

type EasingTracker struct {
	EasingFunction EasingFunction
	TotalSteps     int
	CurrentStep    int
	ProgressRatio  float64
	StepDelta      float64
	EasedValue     float64
	lastEased      float64
	Clamp          bool
}

func NewEasingTracker(fn EasingFunction, totalSteps int, clamp bool) *EasingTracker {
	if totalSteps <= 0 {
		totalSteps = 100
	}
	return &EasingTracker{EasingFunction: fn, TotalSteps: totalSteps, Clamp: clamp}
}

func (t *EasingTracker) Step() float64 {
	if t.CurrentStep < t.TotalSteps {
		t.CurrentStep++
		t.ProgressRatio = float64(t.CurrentStep) / float64(t.TotalSteps)
		t.EasedValue = t.EasingFunction(t.ProgressRatio)
		if t.Clamp {
			if t.EasedValue < 0 {
				t.EasedValue = 0
			}
			if t.EasedValue > 1 {
				t.EasedValue = 1
			}
		}
		t.StepDelta = t.EasedValue - t.lastEased
		t.lastEased = t.EasedValue
	}
	return t.EasedValue
}

func (t *EasingTracker) IsComplete() bool {
	return t.CurrentStep >= t.TotalSteps
}

func (t *EasingTracker) Reset() {
	t.CurrentStep = 0
	t.ProgressRatio = 0
	t.StepDelta = 0
	t.EasedValue = 0
	t.lastEased = 0
}

type SequenceEaser[T any] struct {
	Sequence       []T
	EasingFunction EasingFunction
	TotalSteps     int
	Added          []T
	Removed        []T
	Total          []T
	tracker        *EasingTracker
}

func NewSequenceEaser[T any](sequence []T, easing EasingFunction) *SequenceEaser[T] {
	easer := &SequenceEaser[T]{Sequence: sequence, EasingFunction: easing, TotalSteps: 100}
	easer.tracker = NewEasingTracker(easing, easer.TotalSteps, true)
	return easer
}

func (s *SequenceEaser[T]) Step() []T {
	previous := s.tracker.EasedValue
	eased := s.tracker.Step()
	seqLen := len(s.Sequence)
	if seqLen == 0 {
		s.Added = s.Sequence[:0]
		s.Removed = s.Sequence[:0]
		s.Total = s.Sequence[:0]
		return s.Added
	}
	length := int(eased * float64(seqLen))
	prevLength := int(previous * float64(seqLen))
	if length > prevLength {
		s.Added = s.Sequence[prevLength:length]
		s.Removed = s.Sequence[:0]
	} else if length < prevLength {
		s.Added = s.Sequence[:0]
		s.Removed = s.Sequence[length:prevLength]
	} else {
		s.Added = s.Sequence[:0]
		s.Removed = s.Sequence[:0]
	}
	s.Total = s.Sequence[:length]
	return s.Added
}

func (s *SequenceEaser[T]) IsComplete() bool {
	return s.tracker.IsComplete()
}

func (s *SequenceEaser[T]) Reset() {
	s.tracker.Reset()
	s.Added = s.Sequence[:0]
	s.Removed = s.Sequence[:0]
	s.Total = s.Sequence[:0]
}
