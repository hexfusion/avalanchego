// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpcchainvm

import (
	"context"
	"fmt"
	"os"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/resource"
	"github.com/ava-labs/avalanchego/vms"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/grpcutils"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime/container"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime/subprocess"

	vmpb "github.com/ava-labs/avalanchego/proto/pb/vm"
)

var _ vms.Factory = (*factory)(nil)

type RuntimeType string

const (
	RuntimeContainer  RuntimeType = "Container"
	RuntimeSubprocess RuntimeType = "Subprocess"
)

type factory struct {
	runtime        RuntimeType
	path           string
	processTracker resource.ProcessTracker
	runtimeTracker runtime.Tracker
}

func NewFactory(path string, runtimeType RuntimeType, processTracker resource.ProcessTracker, runtimeTracker runtime.Tracker) vms.Factory {
	return &factory{
		runtime:        runtimeType,
		path:           path,
		processTracker: processTracker,
		runtimeTracker: runtimeTracker,
	}
}

func (f *factory) New(log logging.Logger) (interface{}, error) {

	listener, err := grpcutils.NewListener()
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	var vm *VMClient

	// TODO dedupe O:)
	switch f.runtime {
	case RuntimeSubprocess:
		config := &subprocess.Config{
			Stderr:           log,
			Stdout:           log,
			HandshakeTimeout: runtime.DefaultHandshakeTimeout,
			Log:              log,
		}

		status, stopper, err := subprocess.Bootstrap(
			context.TODO(),
			listener,
			subprocess.NewCmd(f.path),
			config,
		)
		if err != nil {
			return nil, err
		}

		clientConn, err := grpcutils.Dial(status.Addr)
		if err != nil {
			return nil, err
		}

		vm = NewClient(vmpb.NewVMClient(clientConn))
		vm.SetProcess(stopper, status.Pid, f.processTracker)
		f.runtimeTracker.TrackRuntime(stopper)

	case RuntimeContainer:
		podBytes, err := os.ReadFile(f.path)
		if err != nil {
			return nil, fmt.Errorf("failed to read pod yaml: %q", f.path)
		}
		config := &container.Config{
			PodBytes:         podBytes,
			HandshakeTimeout: runtime.DefaultHandshakeTimeout,
			Log:              log,
		}

		status, stopper, err := container.Bootstrap(
			context.TODO(),
			listener,
			config,
		)
		if err != nil {
			return nil, err
		}
		clientConn, err := grpcutils.Dial(status.Addr)
		if err != nil {
			return nil, err
		}

		vm = NewClient(vmpb.NewVMClient(clientConn))
		// IDK if this does anything crazy with pid 0
		vm.SetProcess(stopper, 0, f.processTracker)
		f.runtimeTracker.TrackRuntime(stopper)
	}

	return vm, nil
}
