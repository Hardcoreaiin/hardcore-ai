package debug

import (
	"fmt"
	"sync"
	"time"
)

type BreakpointType string

const (
	BreakpointHardware  BreakpointType = "HARDWARE"
	BreakpointSoftware  BreakpointType = "SOFTWARE"
	BreakpointTemporary BreakpointType = "TEMPORARY"
)

type Breakpoint struct {
	ID        int
	Address   uint32
	Type      BreakpointType
	Enabled   bool
	Temporary bool

	HitCount  int
	CreatedAt time.Time
}

type BreakpointHitEvent struct {
	BreakpointID int
	Address      uint32
	PC           uint32
	Time         time.Time
}

type BreakpointManager struct {
	debugger Debugger

	mu sync.RWMutex

	nextID int

	breakpoints map[uint32]*Breakpoint

	hitEvents chan BreakpointHitEvent
}

func NewBreakpointManager(
	debugger Debugger,
) *BreakpointManager {

	return &BreakpointManager{
		debugger: debugger,

		nextID: 1,

		breakpoints: make(map[uint32]*Breakpoint),

		hitEvents: make(chan BreakpointHitEvent, 128),
	}
}

func (b *BreakpointManager) SetBreakpoint(
	addr uint32,
) (*Breakpoint, error) {

	b.mu.Lock()
	defer b.mu.Unlock()

	if existing, ok := b.breakpoints[addr]; ok {
		return existing, nil
	}

	err := b.debugger.SetBreakpoint(addr)
	if err != nil {
		return nil, err
	}

	bp := &Breakpoint{
		ID:        b.nextID,
		Address:   addr,
		Type:      BreakpointHardware,
		Enabled:   true,
		Temporary: false,
		HitCount:  0,
		CreatedAt: time.Now(),
	}

	b.breakpoints[addr] = bp

	b.nextID++

	return bp, nil
}

func (b *BreakpointManager) SetTemporaryBreakpoint(
	addr uint32,
) (*Breakpoint, error) {

	b.mu.Lock()
	defer b.mu.Unlock()

	err := b.debugger.SetBreakpoint(addr)
	if err != nil {
		return nil, err
	}

	bp := &Breakpoint{
		ID:        b.nextID,
		Address:   addr,
		Type:      BreakpointTemporary,
		Enabled:   true,
		Temporary: true,
		HitCount:  0,
		CreatedAt: time.Now(),
	}

	b.breakpoints[addr] = bp

	b.nextID++

	return bp, nil
}

func (b *BreakpointManager) RemoveBreakpoint(
	addr uint32,
) error {

	b.mu.Lock()
	defer b.mu.Unlock()

	err := b.debugger.RemoveBreakpoint(addr)
	if err != nil {
		return err
	}

	delete(b.breakpoints, addr)

	return nil
}

func (b *BreakpointManager) RemoveAll() error {

	b.mu.Lock()
	defer b.mu.Unlock()

	for addr := range b.breakpoints {

		err := b.debugger.RemoveBreakpoint(addr)
		if err != nil {
			return err
		}
	}

	b.breakpoints = make(map[uint32]*Breakpoint)

	return nil
}

func (b *BreakpointManager) List() []*Breakpoint {

	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]*Breakpoint, 0)

	for _, bp := range b.breakpoints {
		out = append(out, bp)
	}

	return out
}

func (b *BreakpointManager) StartMonitor() {

	go func() {

		for {

			time.Sleep(100 * time.Millisecond)

			state := b.debugger.GetState()

			if state != StateHalted {
				continue
			}

			regs, err := b.debugger.ReadRegisters()
			if err != nil {
				continue
			}

			pc := regs.PC

			b.mu.Lock()

			bp, ok := b.breakpoints[pc]

			if ok {

				bp.HitCount++

				event := BreakpointHitEvent{
					BreakpointID: bp.ID,
					Address:      bp.Address,
					PC:           pc,
					Time:         time.Now(),
				}

				select {
				case b.hitEvents <- event:
				default:
				}

				fmt.Printf(
					"[BREAKPOINT HIT] ID=%d PC=0x%08X COUNT=%d\n",
					bp.ID,
					pc,
					bp.HitCount,
				)

				if bp.Temporary {

					_ = b.debugger.RemoveBreakpoint(pc)

					delete(b.breakpoints, pc)
				}
			}

			b.mu.Unlock()
		}
	}()
}

func (b *BreakpointManager) Events() <-chan BreakpointHitEvent {
	return b.hitEvents
}
