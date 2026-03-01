//go:build integration

package podintegration

import (
	"fmt"
	"os"
	"testing"

	"k8s.io/client-go/kubernetes"

	"github.com/michaelkoro/kubewhy/internal/analyzer/pod"
	"github.com/michaelkoro/kubewhy/tests/integration/framework"
)

// env holds the shared namespace and clients for this test package.
var env *framework.Env

// clientset is a convenience alias into env so helpers can reference it directly.
var clientset *kubernetes.Clientset

// analyzer is the pod analyzer under test.
var analyzer *pod.PodAnalyzer

// TestMain creates the shared Env (namespace + clients) before any test runs
// and tears it down afterwards.
func TestMain(m *testing.M) {
	var err error
	env, err = framework.NewEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration setup failed: %v\n", err)
		os.Exit(1)
	}

	clientset = env.Clientset
	analyzer = pod.NewAnalyzer(env.K8sClient)

	code := m.Run()

	env.Teardown()
	os.Exit(code)
}
