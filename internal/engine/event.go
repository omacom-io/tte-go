package engine

import (
	"fmt"

	"tte-go/internal/utils"
)

type EventType int

type ActionType int

const (
	EventSegmentEntered EventType = iota + 1
	EventSegmentExited
	EventPathActivated
	EventPathComplete
	EventPathHolding
	EventSceneActivated
	EventSceneComplete
)

const (
	ActionActivatePath ActionType = iota + 1
	ActionActivateScene
	ActionDeactivatePath
	ActionDeactivateScene
	ActionResetAppearance
	ActionSetLayer
	ActionSetCoordinate
	ActionCallback
)

type Callback struct {
	Fn   func(*EffectCharacter, ...any)
	Args []any
}

type eventKey struct {
	Event  EventType
	Caller any
}

type eventAction struct {
	Action ActionType
	Target any
}

type EventHandler struct {
	Character *EffectCharacter
	Events    map[eventKey][]eventAction
}

func NewEventHandler(character *EffectCharacter) *EventHandler {
	return &EventHandler{Character: character, Events: map[eventKey][]eventAction{}}
}

func (h *EventHandler) RegisterEvent(event EventType, caller any, action ActionType, target any) error {
	if caller == nil {
		return fmt.Errorf("event caller is required")
	}
	key := eventKey{Event: event, Caller: caller}
	entry := eventAction{Action: action, Target: target}
	for _, existing := range h.Events[key] {
		if existing.Action == entry.Action && existing.Target == entry.Target {
			return fmt.Errorf("duplicate event registration")
		}
	}
	h.Events[key] = append(h.Events[key], entry)
	return nil
}

func (h *EventHandler) Handle(event EventType, caller any) {
	key := eventKey{Event: event, Caller: caller}
	actions, ok := h.Events[key]
	if !ok {
		return
	}
	for _, action := range actions {
		h.dispatch(action)
	}
}

func (h *EventHandler) dispatch(action eventAction) {
	switch action.Action {
	case ActionActivatePath:
		if path, ok := action.Target.(*Path); ok {
			h.Character.Motion.ActivatePath(path)
		}
	case ActionDeactivatePath:
		if path, ok := action.Target.(*Path); ok {
			h.Character.Motion.DeactivatePath(path)
		}
	case ActionActivateScene:
		if scene, ok := action.Target.(*Scene); ok {
			h.Character.Animation.ActivateSceneRef(scene)
		}
	case ActionDeactivateScene:
		h.Character.Animation.DeactivateScene()
	case ActionResetAppearance:
		visual := h.Character.current
		visual.Symbol = h.Character.Symbol
		if h.Character.InputColors != nil {
			visual.Colors = h.Character.InputColors
		}
		h.Character.SetVisual(visual)
	case ActionSetLayer:
		if layer, ok := action.Target.(int); ok {
			h.Character.Layer = layer
		}
	case ActionSetCoordinate:
		if coord, ok := action.Target.(utils.Coord); ok {
			h.Character.Coord = coord
		}
	case ActionCallback:
		if callback, ok := action.Target.(Callback); ok {
			callback.Fn(h.Character, callback.Args...)
		}
	}
}
