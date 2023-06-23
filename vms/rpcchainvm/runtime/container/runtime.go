// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package container

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ghodss/yaml"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/grpcutils"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/gruntime"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/kube"

	pb "github.com/ava-labs/avalanchego/proto/pb/vm/runtime"
)

type Config struct {
	PodBytes []byte
	// Duration engine server will wait for handshake success.
	HandshakeTimeout time.Duration
	Log              logging.Logger
}

type Status struct {
	// Pod bytes used to shutdown
	PodBytes []byte
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

	log := config.Log
	serverAddr := listener.Addr()

	socket, err := getSocketPath()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find socket path: %w", err)
	}

	pctx, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start new podman connection: %w", err)
	}

	// all this should go into factory
	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(config.PodBytes, nil, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derialize pod yaml: %w", err)
	}

	// ensure valid pod spec from bytes
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return nil, nil, fmt.Errorf("not a valid v1.Pod: %w", err)
	}

	//now we can inject stuff we want to enforce
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env,
		v1.EnvVar{
			Name:  runtime.EngineAddressKey,
			Value: serverAddr.String(),
		},
	)

	podBytes, err := yaml.Marshal(&pod)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshall pod: %w", err)
	}

	_, err = kube.PlayWithBody(ctx, bytes.NewReader(podBytes), &kube.PlayOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start pod: %w", err)
	}

	// fix stopper
	stopper := NewStopper(log, cmd)

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
		PodBytes: podBytes,
		Addr:     intitializer.vmAddr,
	}
	return status, stopper, nil
}
