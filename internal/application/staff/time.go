package staff

// TimeOfDay is a wall-clock time stored as microseconds since midnight.
type TimeOfDay struct {
	Microseconds int64
	Valid        bool
}
