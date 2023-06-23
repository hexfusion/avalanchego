// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package container

import (
	"context"
	"os/exec"
	"sync"
	"syscall"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime"
	"go.uber.org/zap"
)

func NewStopper(logger logging.Logger, cmd *exec.Cmd) runtime.Stopper {
	return &stopper{
		cmd:    cmd,
		logger: logger,
	}
}

type stopper struct {
	once   sync.Once
	cmd    *exec.Cmd
	logger logging.Logger
}

func (s *stopper) Stop(ctx context.Context) {
	s.once.Do(func() {
		stop(ctx, s.logger, s.cmd)
	})
}


func stop(ctx context.Context, log logging.Logger, cmd *exec.Cmd) {
	waitChan := make(chan error)
	go func() {
		// attempt graceful shutdown
		errs := wrappers.Errs{}
		err := cmd.Process.Signal(syscall.SIGTERM)
		errs.Add(err)
		_, err = cmd.Process.Wait()
		errs.Add(err)
		waitChan <- errs.Err
		close(waitChan)
	}()

	ctx, cancel := context.WithTimeout(ctx, runtime.DefaultGracefulTimeout)
	defer cancel()

	select {
	case err := <-waitChan:
		if err == nil {
			log.Debug("subprocess gracefully shutdown")
		} else {
			log.Error("subprocess graceful shutdown failed",
				zap.Error(err),
			)
		}
	case <-ctx.Done():
		// force kill
		err := cmd.Process.Kill()
		log.Error("subprocess was killed",
			zap.Error(err),
		)
	}
}
