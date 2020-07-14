/*
 * Copyright (c) 2019 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package cluster

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/SUSE/skuba/internal/pkg/skuba/kubernetes"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kubectlget "k8s.io/kubectl/pkg/cmd/get"
)

// Status prints the status of the cluster on the standard output by reading the
// admin configuration file from the current folder
func Status(client clientset.Interface) error {
	nodeList, err := client.CoreV1().Nodes().List(
		context.TODO(),
		metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "could not retrieve node list")
	}

	for _, node := range nodeList.Items {
		status := node.Status.Conditions[len(node.Status.Conditions)-1].Status
		if status == "True" {
			node.Labels["node-status.kubernetes.io"] = "Ready"
		} else {
			node.Labels["node-status.kubernetes.io"] = "NotReady"
		}

		if ok := node.Spec.Unschedulable; ok {
			node.Labels["node-status.kubernetes.io"] = node.Labels["node-status.kubernetes.io"] + ",SchedulingDisabled"
		}

		for label := range node.Labels {
			if strings.Contains(label, "node-role.kubernetes.io") && len(strings.Split(label, "/")) > 0 {
				node.Labels["caasp-role.kubernetes.io"] = strings.Split(label, "/")[1]
			}
		}
		currentAnnotations := node.GetAnnotations()
		// TODO: Fine tune what means is supported or not. Is it only because you are at latest release?
		// This would mark the state as unsupported after `zypper migration`, because skuba would get
		// updated, and the node would be outdated.
		if node.Status.NodeInfo.KubeletVersion == fmt.Sprintf("v%s", kubernetes.LatestVersion().String()) {
			currentAnnotations["caasp.suse.com/supported"] = "✅"
		} else {
			currentAnnotations["caasp.suse.com/supported"] = "❌"
		}
	}

	outputFormat := "custom-columns=" +
		"NAME:.metadata.name," +
		"STATUS:.metadata.labels.node-status\\.kubernetes\\.io," +
		"SUPPORTED:.metadata.annotations.caasp\\.suse\\.com/supported," +
		"ROLE:.metadata.labels.caasp-role\\.kubernetes\\.io," +
		"OS-IMAGE:.status.nodeInfo.osImage," +
		"KERNEL:.status.nodeInfo.kernelVersion," +
		"KUBELET:.status.nodeInfo.kubeletVersion," +
		"CONTAINER-RUNTIME:.status.nodeInfo.containerRuntimeVersion," +
		"HAS-UPDATES:.metadata.annotations.caasp\\.suse\\.com/has-updates," +
		"HAS-DISRUPTIVE-UPDATES:.metadata.annotations.caasp\\.suse\\.com/has-disruptive-updates," +
		"RELEASE:.metadata.annotations.caasp\\.suse\\.com/caasp-release-version"

	printFlags := kubectlget.NewGetPrintFlags()
	printFlags.OutputFormat = &outputFormat

	printer, err := printFlags.ToPrinter()
	if err != nil {
		return errors.Wrap(err, "could not create printer")
	}
	if err := printer.PrintObj(nodeList, os.Stdout); err != nil {
		return errors.Wrap(err, "could not print to stdout")
	}
	return nil
}
