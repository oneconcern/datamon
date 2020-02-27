package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/dlogger"

	"go.uber.org/zap"
)

func writeProfIfNExist(path string, name string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		var fprof *os.File
		fprof, err = os.Create(path)
		if err != nil {
			return err
		}
		defer fprof.Close()
		err = pprof.Lookup(name).WriteTo(fprof, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

type MinProfMB struct {
	Alloc   uint64
	HeapSys uint64
}

type MaybeMemProfParams struct {
	MemStats   *runtime.MemStats
	MinMB      MinProfMB
	DestDir    string
	NamePrefix string
}

func maybeMemProfDefaults(params MaybeMemProfParams) MaybeMemProfParams {
	if params.DestDir == "" {
		params.DestDir = "/home/developer/"
	}
	if params.NamePrefix == "" {
		params.NamePrefix = "mem_" + rand.LetterString(3)
	}
	if params.MemStats == nil {
		mstats := new(runtime.MemStats)
		runtime.ReadMemStats(mstats)
		params.MemStats = mstats
	}
	return params
}

func MaybeMemProf(params MaybeMemProfParams) error {
	params = maybeMemProfDefaults(params)
	if params.MemStats.Alloc/1024/1024 < params.MinMB.Alloc ||
		params.MemStats.HeapSys/1024/1024 < params.MinMB.HeapSys {
		return nil
	}
	if _, err := os.Stat(params.DestDir); !os.IsNotExist(err) {
		basePath := filepath.Join(params.DestDir, strings.Join([]string{
			params.NamePrefix,
			strconv.Itoa(int(params.MinMB.Alloc)),
			strconv.Itoa(int(params.MinMB.HeapSys)),
		}, "-"))
		if err := writeProfIfNExist(basePath+".mem.prof", "heap"); err != nil {
			return err
		}
		if err := writeProfIfNExist(basePath+".alloc.prof", "allocs"); err != nil {
			return err
		}
	}
	return nil
}

type MemPollParams struct {
	PollMs    uint
	LoopLogMs uint
	MinMBs    []MinProfMB
	Logger    *zap.Logger
}

func memPollDefaults(params MemPollParams) (MemPollParams, error) {
	if params.PollMs == 0 {
		params.PollMs = 50
	}
	if params.MinMBs == nil {
		params.MinMBs = make([]MinProfMB, 0)
	}
	if params.Logger == nil {
		logger, err := dlogger.GetLogger("info")
		if err != nil {
			return MemPollParams{}, err
		}
		params.Logger = logger
	}
	return params, nil
}

func memPollGoroutine(params MemPollParams) {
	mstats := new(runtime.MemStats)
	var maxHeapThusFar uint64
	var msSinceLog uint
	for {
		runtime.ReadMemStats(mstats)
		if params.LoopLogMs != 0 && msSinceLog >= params.LoopLogMs {
			params.Logger.Info("mempoll",
				zap.Uint64("MiB for heap (un-GC)", mstats.Alloc/1024/1024),
				zap.Uint64("MiB for heap (max ever)", mstats.HeapSys/1024/1024),
				zap.Int("num go routines", runtime.NumGoroutine()),
			)
			msSinceLog = 0
		}
		if mstats.HeapSys > maxHeapThusFar {
			maxHeapThusFar = mstats.HeapSys
			params.Logger.Info("grew heap",
				zap.Uint64("MiB for heap (un-GC)", mstats.Alloc/1024/1024),
				zap.Uint64("MiB for heap (max ever)", mstats.HeapSys/1024/1024),
			)
		}
		for _, minMB := range params.MinMBs {
			if err := MaybeMemProf(MaybeMemProfParams{
				MemStats:   mstats,
				MinMB:      minMB,
				NamePrefix: "mem_poll",
			}); err != nil {
				params.Logger.Error("memory profiling error",
					zap.Error(err),
				)
			}
		}
		time.Sleep(time.Duration(params.PollMs) * time.Millisecond)
		msSinceLog += params.PollMs
	}
}

func MemPoll(params MemPollParams) error {
	params, err := memPollDefaults(params)
	if err != nil {
		return err
	}
	go memPollGoroutine(params)
	return nil
}
