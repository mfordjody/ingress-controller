package controller

type Event struct {
	Type      EventType
	Obj       any
	OldObj    any
	Tombstone any
}

type EventType int

const (
	Add = iota + 1
	Update
	Delete
)

func (e EventType) String() string {
	switch e {
	case Add:
		return "Add"
	case Update:
		return "Update"
	case Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}
