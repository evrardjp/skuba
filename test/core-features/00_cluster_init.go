package corefeatures

import (
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"os"
	"os/exec"
	"time"
)

var _ = ginkgo.Describe("Create Caaspctl Cluster", func() {
	// ENV. parameters , for convenience here but they can be global parameter, configurable and passed to testsuite
	controlPlaneIP := os.Getenv("CONTROLPLANE") // ENV variable IP of controlplane
	master00IP := os.Getenv("MASTER00")         // IP of master 00
	worker00IP := os.Getenv("WORKER00")         // IP of worker 00

	// constants used by caaspctl
	clusterName := "e2e-cluster"
	master00Name := "master00"
	worker00Name := "worker00"

	// configuration of OS
	username := "sles"

	// Use an RPM binary provided by env variable otherwise use devel mode
	var caaspctl string
	caaspctl = os.Getenv("CAASPCTL_BIN_PATH")
	if len(caaspctl) == 0 {
		// use devel binary from gopath
		fmt.Println("taking caaspctl from GOPATH")
		caaspctl = os.Getenv("GOPATH") + "/bin/caaspctl"
	}

	// wait 10 minutes max as timeout for completing command
	// the default timeout provided by ginkgo is 1 sec which is to low for us.
	gomega.SetDefaultEventuallyTimeout(600 * time.Second)
	gomega.SetDefaultEventuallyPollingInterval(5 * time.Second)
	gomega.SetDefaultConsistentlyDuration(600 * time.Second)
	gomega.SetDefaultConsistentlyPollingInterval(5 * time.Second)

	ginkgo.BeforeEach(func() {
		os.RemoveAll(clusterName)
	})

	ginkgo.It("00: Initialize cluster", func() {
		ginkgo.By("create configuration files")
		command := exec.Command(caaspctl, "cluster", "init", "--control-plane", controlPlaneIP, clusterName)
		session, err := gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
		gomega.Eventually(session.Out).Should(gbytes.Say(".*configuration files written to"))
		gomega.Expect(session).Should(gexec.Exit(), "configuration was not created")
		gomega.Expect(err).To(gomega.BeNil(), "configuration was not created")

		// change to created caaspctl directory
		err = os.Chdir(clusterName)
		if err != nil {
			panic(err)
		}

		ginkgo.By("add master00 to the cluster")
		command = exec.Command(caaspctl, "node", "bootstrap", "-v3", "--user", username, "--sudo", "--target", master00IP, master00Name)
		session, err = gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)

		gomega.Expect(session.Wait().Out.Contents()).Should(gomega.ContainSubstring("kubeadm.init applied successfully"))
		gomega.Expect(session).Should(gexec.Exit(), "caaspctl adding master00 failed")
		gomega.Expect(err).To(gomega.BeNil(), "caaspctl adding master00 failed")

		ginkgo.By("verify master00 with caaspctl status")
		command = exec.Command(caaspctl, "cluster", "status")
		session, err = gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)

		gomega.Eventually(session.Out).Should(gbytes.Say(".*" + master00Name))
		gomega.Expect(session).Should(gexec.Exit(), "caaspctl status verify master00 failed")
		gomega.Expect(err).To(gomega.BeNil(), "caaspctl status verify master00 failed")

		ginkgo.By("add a worker00 to the cluster")
		command = exec.Command(caaspctl, "node", "join", "-v3", "--role", "worker", "--user", username, "--sudo", "--target", worker00IP, worker00Name)
		session, err = gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)

		gomega.Eventually(session.Out).Should(gbytes.Say(".*state kubeadm.join applied successfully"))
		gomega.Expect(session).Should(gexec.Exit(), "caaspctl adding worker00 failed")
		gomega.Expect(err).To(gomega.BeNil(), "caaspctl adding worker00 failed")

		ginkgo.By("verify worker00 with caaspctl status")
		command = exec.Command(caaspctl, "cluster", "status")
		session, err = gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)

		gomega.Eventually(session.Out).Should(gbytes.Say(".*" + worker00Name))
		gomega.Expect(session).Should(gexec.Exit(), "caaspctl status verify worker00 failed")
		gomega.Expect(err).To(gomega.BeNil(), "caaspctl status verify worker00 failed")

	})

})
