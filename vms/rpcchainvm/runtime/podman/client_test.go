package podman

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/kube"
	"github.com/ghodss/yaml"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var podYamlBytes = `
apiVersion: v1
kind: Pod
metadata:
  name: subnet-evm
spec:
  hostNetwork: true
  containers:
  - name: subnet-evm
    image: dansoveravax/subnet-evm-container:v0.0.1-amd64
    resources:
      requests:
        memory: 600Mi
        cpu: 300m`

// go test -v -timeout 30s -run ^TestSchedulePod$ github.com/ava-labs/avalanchego/vms/rpcchainvm/runtime/podman
func TestSchedulePod(t *testing.T) {
	require := require.New(t)
	obj, kind, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(podYamlBytes), nil, nil)
	require.NoError(err)

	// ensure valid pod spec from bytes
	pod, ok := obj.(*v1.Pod)
	require.NotNil(ok)

	//now we can inject stuff we want to enforce
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env,
		v1.EnvVar{
			Name:  "INJECTED_IP",
			Value: "IP",
		},
	)

	require.Equal(kind.Kind, "Pod")

	podBytes, err := yaml.Marshal(&pod)
	require.NoError(err)

	fmt.Printf("%#v\n", fmt.Sprint(string(podBytes)))

	socket, err := getSocketPath()
	require.NoError(err)

	fmt.Printf("%#v\n", socket)
	ctx, err := bindings.NewConnection(context.Background(), socket)
	require.NoError(err)

	report, err := kube.PlayWithBody(ctx, bytes.NewReader(podBytes), &kube.PlayOptions{})
	require.NoError(err)

	fmt.Printf("%#v\n", report)


}
