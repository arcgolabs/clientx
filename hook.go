package clientx

import (
	"time"

	"github.com/samber/lo"
)

// Hook observes dial and I/O events emitted by client implementations.
type Hook interface {
	OnDial(event DialEvent)
	OnIO(event IOEvent)
}

// HookFuncs adapts plain functions to the Hook interface.
type HookFuncs struct {
	OnDialFunc func(event DialEvent)
	OnIOFunc   func(event IOEvent)
}

// OnDial dispatches the dial event to OnDialFunc.
func (h HookFuncs) OnDial(event DialEvent) {
	if h.OnDialFunc != nil {
		h.OnDialFunc(event)
	}
}

// OnIO dispatches the I/O event to OnIOFunc.
func (h HookFuncs) OnIO(event IOEvent) {
	if h.OnIOFunc != nil {
		h.OnIOFunc(event)
	}
}

// DialEvent describes a dial or listen attempt.
type DialEvent struct {
	Protocol Protocol
	Op       string
	Network  string
	Addr     string
	Duration time.Duration
	Err      error
}

// IOEvent describes a read, write, or request execution event.
type IOEvent struct {
	Protocol Protocol
	Op       string
	Addr     string
	Bytes    int
	Duration time.Duration
	Err      error
}

// EmitDial dispatches a dial event to all hooks.
func EmitDial(hooks []Hook, event DialEvent) {
	emitHooks(hooks, event, emitDialSafe)
}

// EmitIO dispatches an I/O event to all hooks.
func EmitIO(hooks []Hook, event IOEvent) {
	emitHooks(hooks, event, emitIOSafe)
}

func emitHooks[T any](hooks []Hook, event T, emit func(Hook, T)) {
	lo.ForEach(hooks, func(h Hook, _ int) {
		emit(h, event)
	})
}

func emitDialSafe(h Hook, event DialEvent) {
	if h == nil {
		return
	}
	defer func() {
		_ = recover() != nil
	}()
	h.OnDial(event)
}

func emitIOSafe(h Hook, event IOEvent) {
	if h == nil {
		return
	}
	defer func() {
		_ = recover() != nil
	}()
	h.OnIO(event)
}
