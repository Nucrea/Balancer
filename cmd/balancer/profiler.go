package main

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

func NewProfiler() *Profiler {
	return &Profiler{
		mutex: &sync.Mutex{},
	}
}

type Profiler struct {
	mutex    *sync.Mutex
	file     *os.File
	stopChan chan struct{}
}

func (p *Profiler) Start(ctx context.Context) (string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.file != nil {
		return "", fmt.Errorf("profiling already enabled")
	}

	var err error
	p.file, err = os.Create(fmt.Sprintf("./%d.pprof", time.Now().UnixMilli()))
	if err != nil {
		return "", fmt.Errorf("failed creating pprof file: %w", err)
	}
	fileName := p.file.Name()

	p.stopChan = make(chan struct{})

	go func() {
		defer func() {
			pprof.StopCPUProfile()
			p.file.Close()
			p.file = nil
		}()

		//TODO: add error handling
		pprof.StartCPUProfile(p.file)

		select {
		case <-ctx.Done():
		case <-p.stopChan:
		}
	}()

	return fileName, nil
}

func (p *Profiler) Stop() (string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.file == nil {
		return "", fmt.Errorf("profiling not enabled")
	}
	fileName := p.file.Name()

	close(p.stopChan)
	return fileName, nil
}
