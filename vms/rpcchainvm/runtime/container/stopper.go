// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package container

import (
	"bytes"
	"context"
	"sync"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/kube"
	"go.uber.org/zap"
)

func NewStopper(logger logging.Logger, socketPath string, podBytes []byte) runtime.Stopper {
	return &stopper{
		socketPath: socketPath,
		logger:     logger,
		podBytes:   podBytes,
	}
}

type stopper struct {
	once       sync.Once
	socketPath string
	podBytes   []byte
	logger     logging.Logger
}

func (s *stopper) Stop(ctx context.Context) {
	s.once.Do(func() {
		stop(ctx, s.socketPath, s.logger, s.podBytes)
	})
}

func stop(ctx context.Context, socketPath string, log logging.Logger, podBytes []byte) {
	waitChan := make(chan error)
	go func() {
		// attempt graceful shutdown
		errs := wrappers.Errs{}
		ctx, err := bindings.NewConnection(context.Background(), socketPath)
		errs.Add(err)
		_, err = kube.DownWithBody(ctx, bytes.NewReader(podBytes), kube.DownOptions{})
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
		// force kill TODO whats the heavy hammer here?
	}
}
