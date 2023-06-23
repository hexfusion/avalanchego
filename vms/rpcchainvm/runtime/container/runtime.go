// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package container

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/grpcutils"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/gruntime"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime/podman"

	pb "github.com/ava-labs/avalanchego/proto/pb/vm/runtime"
)

type Config struct {
	// Stderr of the VM process written to this writer.
	Stderr io.Writer
	// Stdout of the VM process written to this writer.
	Stdout io.Writer
	// Duration engine server will wait for handshake success.
	HandshakeTimeout time.Duration
	Log              logging.Logger
}

type Status struct {
	// Id of the process.
	Pid int
	// Address of the VM gRPC service.
	Addr string
}

// Bootstrap starts a VM as a subprocess after initialization completes and
// pipes the IO to the appropriate writers.
//
// The subprocess is expected to be stopped by the caller if a non-nil error is
// returned. If piping the IO fails then the subprocess will be stopped.
//
// TODO: create the listener inside this method once we refactor the tests
func Bootstrap(
	ctx context.Context,
	listener net.Listener,
	config *Config,
) (*Status, runtime.Stopper, error) {
	defer listener.Close()

	intitializer := newInitializer()

	server := grpcutils.NewServer()
	defer server.GracefulStop()
	pb.RegisterRuntimeServer(server, gruntime.NewServer(intitializer))

	go grpcutils.Serve(listener, server)

	serverAddr := listener.Addr()

	podman.NewClient()

	// set pod ENV
	// cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", runtime.EngineAddressKey, serverAddr.String()))
	// pass golang debug env to subprocess
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GRPC_") || strings.HasPrefix(env, "GODEBUG") {
		}
	}

	// start container


	// fix stopper
	// stopper := NewStopper(log, cmd)

	// wait for handshake success
	timeout := time.NewTimer(config.HandshakeTimeout)
	defer timeout.Stop()

	select {
	case <-intitializer.initialized:
	case <-timeout.C:
		stopper.Stop(ctx)
		return nil, nil, fmt.Errorf("%w: %v", runtime.ErrHandshakeFailed, runtime.ErrProcessNotFound)
	}

	if intitializer.err != nil {
		stopper.Stop(ctx)
		return nil, nil, fmt.Errorf("%w: %v", runtime.ErrHandshakeFailed, intitializer.err)
	}

	log.Info("plugin handshake succeeded",
		zap.String("addr", intitializer.vmAddr),
	)

	status := &Status{
		Pid:  cmd.Process.Pid,
		Addr: intitializer.vmAddr,
	}
	return status, stopper, nil
}
