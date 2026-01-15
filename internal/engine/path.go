package engine

import (
	"fmt"
	"math"

	"tte-go/internal/utils"
)

type Waypoint struct {
	ID      string
	Coord   utils.Coord
	Control *utils.Coord
}

type Path struct {
	ID            string
	Waypoints     []Waypoint
	Speed         float64
	Ease          utils.EasingFunction
	Loop          bool
	HoldTime      int
	Layer         int
	currentStep   int
	maxSteps      int
	segments      []pathSegment
	originSegment *pathSegment
	lastDistance  float64
	holdRemaining int
	totalDistance float64
}

type pathSegment struct {
	Start          utils.Coord
	End            utils.Coord
	Distance       float64
	enterTriggered bool
	exitTriggered  bool
}

func NewPath(id string, speed float64) *Path {
	if speed <= 0 {
		speed = 1
	}
	return &Path{ID: id, Speed: speed}
}

func (p *Path) AddWaypoint(coord utils.Coord) {
	wp := Waypoint{ID: itoa(len(p.Waypoints)), Coord: coord}
	p.Waypoints = append(p.Waypoints, wp)
	p.rebuild()
}

func (p *Path) AddBezierWaypoint(coord utils.Coord, control utils.Coord) {
	wp := Waypoint{ID: itoa(len(p.Waypoints)), Coord: coord, Control: &control}
	p.Waypoints = append(p.Waypoints, wp)
	p.rebuild()
}

func (p *Path) rebuild() {
	p.segments = nil
	p.maxSteps = 0
	p.currentStep = 0
	p.lastDistance = 0
	p.holdRemaining = p.HoldTime
	p.totalDistance = 0
	p.originSegment = nil
	if len(p.Waypoints) < 2 {
		return
	}
	total := 0.0
	for i := 1; i < len(p.Waypoints); i++ {
		start := p.Waypoints[i-1]
		end := p.Waypoints[i]
		var dist float64
		if end.Control != nil {
			dist = utils.BezierLength(start.Coord, *end.Control, end.Coord, 20)
		} else {
			dist = utils.Distance(start.Coord, end.Coord, true)
		}
		total += dist
		p.segments = append(p.segments, pathSegment{Start: start.Coord, End: end.Coord, Distance: dist})
	}
	if p.Speed > 0 {
		p.maxSteps = int(math.Round(total / p.Speed))
	}
	p.totalDistance = total
}

func (p *Path) Step(handler *EventHandler) (utils.Coord, bool) {
	if len(p.segments) == 0 {
		return utils.Coord{}, false
	}
	if p.holdRemaining > 0 {
		p.holdRemaining--
		return p.segments[len(p.segments)-1].End, true
	}
	if p.maxSteps == 0 || p.currentStep >= p.maxSteps {
		if p.Loop {
			p.currentStep = 0
			p.holdRemaining = p.HoldTime
		} else {
			return p.segments[len(p.segments)-1].End, false
		}
	}
	p.currentStep++
	var factor float64
	if p.Ease != nil {
		factor = p.Ease(float64(p.currentStep) / float64(p.maxSteps))
	} else {
		factor = float64(p.currentStep) / float64(p.maxSteps)
	}
	if factor < 0 {
		factor = 0
	}
	if factor > 1 {
		factor = 1
	}
	distanceToTravel := factor * p.totalDistance
	p.lastDistance = distanceToTravel
	for index := range p.segments {
		segment := &p.segments[index]
		if distanceToTravel <= segment.Distance {
			if !segment.enterTriggered && handler != nil {
				segment.enterTriggered = true
				handler.Handle(EventSegmentEntered, segment)
			}
			localT := distanceToTravel / math.Max(segment.Distance, 1)
			return utils.Lerp(segment.Start, segment.End, localT), true
		}
		distanceToTravel -= segment.Distance
		if !segment.exitTriggered && handler != nil {
			segment.exitTriggered = true
			handler.Handle(EventSegmentExited, segment)
		}
	}
	return p.segments[len(p.segments)-1].End, true
}

func (p *Path) calculateTotalDistance() float64 {
	total := 0.0
	for _, segment := range p.segments {
		total += segment.Distance
	}
	return total
}

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
}
