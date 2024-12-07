// Package event hides the retrieval of events from different sources
package event

// Type represents an event type
type Type uint8

// Event* hold the different event types as integer values
const (
	EventNone               Type = iota // no event was given
	EventAnonymise                      // toggle anonymising data.
	EventFinished                       // please exit the program
	EventViewNext                       // show me the next view
	EventViewPrev                       // show me the previous view
	EventDecreasePollTime               // reduce the poll time (if possible)
	EventIncreasePollTime               // increase the poll time
	EventHelp                           // provide me with help
	EventToggleWantRelative             // toggle between wanting absolute or relative stats
	EventResetStatistics                // reset the current stats back to zero
	EventResizeScreen                   // not really a event but a state change
	EventUnknown                        // something weird has happened
	EventError                          // some error
)

// Event is one of the earlier list of Event constants and also contains a position
type Event struct {
	Type   Type
	Width  int
	Height int
}
