// hide the retrieval of events from different sources behind this
package event

type EventType uint8

const (
	EventNone               EventType = iota // no event was given
	EventFinished                            // please exit the program
	EventViewNext                            // show me the next view
	EventViewPrev                            // show me the previous view
	EventDecreasePollTime                    // reduce the poll time (if possible)
	EventIncreasePollTime                    // increase the poll time
	EventHelp                                // provide me with help
	EventToggleWantRelative                  // toggle beween wanting absolute or relative stats
	EventResetStatistics                     // reset the current stats back to zero
	EventResizeScreen                        // not really a event but a state change
	EventUnknown                             // something weird has happened
	EventError                               // some error
)

type Event struct {
	Type   EventType
	Width  int
	Height int
}

const event_chan_size = 100 // arbitrary size. Maybe should be 0?

var EventChan chan Event

// create an empty event channel
func init() {
	EventChan = make(chan Event, event_chan_size)
}

// read an event from the channel
func ReadEvent() Event {
	e := <-EventChan
	return e
}

// write an event to the channel
func WriteEvent(event Event) {
	EventChan <- event
}
