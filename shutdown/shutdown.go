package shutdown

import (
	"github.com/hfoxy/cobra-starter/logging"
	"github.com/oklog/ulid/v2"
	"math"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
)

var nonBlockingPriorityHooks = false

func WithNonBlockingPriorityHooks() {
	nonBlockingPriorityHooks = true
}

func WithoutNonBlockingPriorityHooks() {
	nonBlockingPriorityHooks = false
}

var sendSignal = true
var hooks = make(Hooks, 0)

type Hooks []*hook

func (h Hooks) Len() int {
	return len(h)
}

func (h Hooks) Less(i, j int) bool {
	return h[i].priority < h[j].priority
}

func (h Hooks) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

type Hook func() error

type hook struct {
	id       ulid.ULID
	fn       Hook
	complete bool
	priority int64
}

func Add(h Hook) ulid.ULID {
	return AddP(math.MaxInt32, h)
}

func AddP(priority int64, h Hook) ulid.ULID {
	mux.Lock()
	defer mux.Unlock()

	id := ulid.Make()
	hooks = append(hooks, &hook{
		priority: priority,
		id:       id,
		fn:       h,
		complete: false,
	})

	return id
}

func Remove(id ulid.ULID) {
	mux.Lock()
	defer mux.Unlock()

	for i, h := range hooks {
		if h.id == id {
			hooks = append(hooks[:i], hooks[i+1:]...)
			return
		}
	}
}

var sigc = make(chan os.Signal, 1)

func Watch() {
	if sendSignal {
		signal.Notify(sigc,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
			syscall.SIGKILL,
		)

		sig := <-sigc
		if sig != syscall.SIGKILL {
			Shutdown()
			os.Exit(0)
		} else {
			os.Exit(137)
		}
	}
}

var mux = new(sync.Mutex)

func Shutdown() {
	mux.Lock()
	defer mux.Unlock()

	if sendSignal {
		sigc <- os.Interrupt
	}

	sort.Sort(hooks)

	priorities := make([]int64, 0, len(hooks))
	hookPriorityMap := make(map[int64][]*hook)

	for _, h := range hooks {
		if _, ok := hookPriorityMap[h.priority]; !ok {
			priorities = append(priorities, h.priority)
			hookPriorityMap[h.priority] = make([]*hook, 0, len(hooks))
		}

		hookPriorityMap[h.priority] = append(hookPriorityMap[h.priority], h)
	}

	for _, priority := range priorities {
		priorityHooks := hookPriorityMap[priority]

		if nonBlockingPriorityHooks {
			wg := new(sync.WaitGroup)

			for _, h := range priorityHooks {
				wg.Add(1)
				go func(h *hook) {
					runHook(h)
					wg.Done()
				}(h)
			}

			wg.Wait()
		} else {
			for _, h := range priorityHooks {
				runHook(h)
			}
		}
	}
}

func runHook(h *hook) {
	if h.complete {
		return
	}

	err := h.fn()
	if err != nil {
		logging.Logger().Error("error running shutdown hook", "error", err)
	}

	h.complete = true
}
