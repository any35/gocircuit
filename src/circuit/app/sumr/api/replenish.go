package api

import (
	"circuit/kit/sched/limiter"
	"circuit/use/anchorfs"
	"circuit/use/circuit"
	"path"
	"strconv"
	"sync"
)

// Replenished holds the return values of a call to Replenish.
type Replenished struct {
	Config      *WorkerConfig	// Config specifies a worker configuration passed to Replenish.
	Replenished bool		// Replenished is true if the API worker on this host needed replenishing.
	Err         error		// Err is non-nil if the operation failed.
}

// durableFile is the name of the durable file describing the SUMR server cluster
func Replenish(durableFile string, c *Config) []*Replenished {
	var (
		lk   sync.Mutex
		lmtr limiter.Limiter
	)
	r := make([]*Replenished, len(c.Workers))
	lmtr.Init(20)
	for i_, wcfg_ := range c.Workers {
		i, wcfg := i_, wcfg_
		lmtr.Go(
			func() {
				re, err := replenishWorker(durableFile, c, i)
				lk.Lock()
				defer lk.Unlock()
				r[i] = &Replenished{Config: wcfg, Replenished: re, Err: err}
			},
		)
	}
	lmtr.Wait()
	return r
}

func replenishWorker(durableFile string, c *Config, i int) (replenished bool, err error) {

	// Check if worker already running
	anchor := path.Join(c.Anchor, strconv.Itoa(i))
	dir, e := anchorfs.OpenDir(anchor)
	if e != nil {
		return false, e
	}
	_, files, err := dir.Files()
	if e != nil {
		return false, e
	}
	if len(files) > 0 {
		return false, nil
	}

	// If not, start a new worker
	retrn, _, err := circuit.Spawn(c.Workers[i].Host, []string{anchor}, durableFile, c.Workers[i].Port, c.ReadOnly)
	if err != nil {
		return false, err
	}
	if retrn[1] != nil {
		err = retrn[1].(error)
		return false, err
	}

	return true, nil
}

// start is a worker function for starting an API worker
type start struct{}

func (start) Start(durableFile string, port int, readOnly bool) (circuit.XPerm, error) {
	println("WHOAAA")
	a, err := New(durableFile, port, readOnly)
	if err != nil {
		println("FLOPPP")
		return nil, err
	}
	return circuit.PermRef(a), nil
}


func init() {
	circuit.RegisterFunc(start{})
}
