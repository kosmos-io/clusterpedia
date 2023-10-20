package e2e

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var (
	// host集群的kubeconfig
	//kubeconfig = os.Getenv("KUBECONFIG")
	// apiserver部署地址 eg http://localhost:443
	//apiserverAddress = os.Getenv("APISERVER_HOST")
	kubeconfig       = "D:\\.kube\\config-38\\config"
	apiserverAddress = "https://localhost:6443"
)

// e2e test start
func TestE2e(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2e Suite")
}
