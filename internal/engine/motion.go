package engine

import (
	"fmt"
	"math"

	"tte-go/internal/utils"
)

type Motion struct {
	Character     *EffectCharacter
	EventHandler  *EventHandler
	CurrentCoord  utils.Coord
	PreviousCoord utils.Coord
	ActivePath    *Path
	CompletedPath *Path
	Paths         map[string]*Path
}

func NewMotion(character *EffectCharacter) *Motion {
	return &Motion{
		Character:     character,
		CurrentCoord:  character.InputCoord,
		PreviousCoord: utils.Coord{Row: -1, Col: -1},
		Paths:         map[string]*Path{},
	}
}

func (m *Motion) SetCoordinate(coord utils.Coord) {
	m.CurrentCoord = coord
}

func (m *Motion) NewPath(speed float64, ease utils.EasingFunction, layer *int, holdTime int, loop bool, pathID string) (*Path, error) {
	if pathID == "" {
		pathID = fmt.Sprintf("%d", len(m.Paths))
	}
	if _, ok := m.Paths[pathID]; ok {
		return nil, fmt.Errorf("duplicate path id: %s", pathID)
	}
	path := NewPath(pathID, speed)
	path.Ease = ease
	path.HoldTime = holdTime
	path.Loop = loop
	if layer != nil {
		path.Layer = *layer
	}
	m.Paths[pathID] = path
	return path, nil
}

func (m *Motion) QueryPath(pathID string) (*Path, error) {
	path, ok := m.Paths[pathID]
	if !ok {
		return nil, fmt.Errorf("path not found: %s", pathID)
	}
	return path, nil
}

func (m *Motion) MovementComplete() bool {
	return m.ActivePath == nil
}

func (m *Motion) ActivatePath(path *Path) {
	if path == nil || len(path.Waypoints) == 0 {
		return
	}
	m.ActivePath = path
	path.rebuild()
	path.currentStep = 0
	path.holdRemaining = path.HoldTime
	if len(path.Waypoints) > 0 {
		first := path.Waypoints[0]
		distance := utils.Distance(m.CurrentCoord, first.Coord, true)
		origin := pathSegment{Start: m.CurrentCoord, End: first.Coord, Distance: distance}
		path.totalDistance += distance
		if path.originSegment != nil {
			if len(path.segments) > 0 {
				path.segments = path.segments[1:]
			}
		}
		path.originSegment = &origin
		path.segments = append([]pathSegment{origin}, path.segments...)
	}
	if path.Speed > 0 {
		path.maxSteps = int(math.Round(path.totalDistance / path.Speed))
	}
	if path.Layer != 0 {
		m.Character.Layer = path.Layer
	}
	if m.EventHandler != nil {
		m.EventHandler.Handle(EventPathActivated, path)
	}
}

func (m *Motion) DeactivatePath(path *Path) {
	if m.ActivePath == path {
		m.ActivePath = nil
	}
}

func (m *Motion) Move() {
	m.PreviousCoord = m.CurrentCoord
	if m.ActivePath == nil {
		return
	}
	coord, ok := m.ActivePath.Step(m.EventHandler)
	if ok {
		m.CurrentCoord = coord
	}
	if m.ActivePath.currentStep >= m.ActivePath.maxSteps {
		if m.ActivePath.HoldTime > 0 && m.ActivePath.holdRemaining > 0 {
			if m.ActivePath.holdRemaining == m.ActivePath.HoldTime && m.EventHandler != nil {
				m.EventHandler.Handle(EventPathHolding, m.ActivePath)
			}
			m.ActivePath.holdRemaining--
			return
		}
		if m.ActivePath.Loop {
			m.ActivePath.currentStep = 0
			m.ActivePath.holdRemaining = m.ActivePath.HoldTime
		} else {
			m.CompletedPath = m.ActivePath
			m.ActivePath = nil
			if m.EventHandler != nil {
				m.EventHandler.Handle(EventPathComplete, m.CompletedPath)
			}
		}
	}
}
