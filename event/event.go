// Package event hides the retrieval of events from different sources
package event

// Type represents an event type
type Type uint8

// Event* hold the different event types as integer values
const (
	EventNone               Type = iota // no event was given
	EventFinished                      // please exit the program
	EventViewNext                      // show me the next view
	EventViewPrev                      // show me the previous view
	EventDecreasePollTime              // reduce the poll time (if possible)
	EventIncreasePollTime              // increase the poll time
	EventHelp                          // provide me with help
	EventToggleWantRelative            // toggle beween wanting absolute or relative stats
	EventResetStatistics               // reset the current stats back to zero
	EventResizeScreen                  // not really a event but a state change
	EventSortNext                      // use the next sort method
	EventUnknown                       // something weird has happened
	EventError                         // some error
)

// Event is one of the earlier list of Event constants and also contains a position
type Event struct {
	Type   Type
	Width  int
	Height int
}

const eventChanSize = 100 // arbitrary size. Maybe should be 0?

// EventChan is a global reference to the channel
var EventChan chan Event

// create an empty event channel
func init() {
	EventChan = make(chan Event, eventChanSize)
}

// Read reads an event from the channel
func Read() Event {
	e := <-EventChan
	return e
}

// Write writes an event to the channel
func Write(event Event) {
	EventChan <- event
}
