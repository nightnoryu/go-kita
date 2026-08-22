package maybe

// state distinguishes the three possible states of a Maybe value.
type state uint8

const (
	// stateAbsent is the zero value: no value was ever provided (e.g. PATCH: don't touch field).
	stateAbsent state = iota
	// stateNone is an explicit null (e.g. PATCH: clear field).
	stateNone
	// stateJust holds a real value (e.g. PATCH: set field).
	stateJust
)

// Maybe is a monad that provides clear and explicit semantic of null value.
// It has 3 states: Just (has value), None (explicit null), and Absent (no value provided).
type Maybe[T any] struct {
	v     T
	state state
}

func NewJust[T any](v T) Maybe[T] {
	return Maybe[T]{
		v:     v,
		state: stateJust,
	}
}

func NewNone[T any]() Maybe[T] {
	return Maybe[T]{state: stateNone}
}

func NewAbsent[T any]() Maybe[T] {
	return Maybe[T]{}
}

func Valid[T any](maybe Maybe[T]) bool {
	return maybe.state == stateJust
}

func IsNone[T any](maybe Maybe[T]) bool {
	return maybe.state == stateNone
}

func IsAbsent[T any](maybe Maybe[T]) bool {
	return maybe.state == stateAbsent
}

// IsSet reports whether a value was provided at all, either Just or None.
func IsSet[T any](maybe Maybe[T]) bool {
	return maybe.state != stateAbsent
}

func Just[T any](maybe Maybe[T]) T {
	if !Valid(maybe) {
		panic("violated usage of maybe: Just on non Valid Maybe")
	}
	return maybe.v
}

func JustValid[T any](maybe Maybe[T]) (v T, ok bool) {
	if !Valid(maybe) {
		ok = false
		return
	}

	return Just(maybe), true
}
