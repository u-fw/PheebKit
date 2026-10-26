package taskbar

// State describes the Windows taskbar progress presentation.
type State int

const (
	Hidden State = iota
	Normal
	Paused
	Error
	Complete
)

type Snapshot struct {
	State State
	Value int
}

type Controller interface {
	Set(Snapshot)
	Close()
}

func clamp(value int) uint64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return uint64(value)
}
