/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"fmt"

	"github.com/containerd/nerdctl/pkg/netutil"

	"github.com/spf13/cobra"
)

func newNetworkRmCommand() *cobra.Command {
	networkRmCommand := &cobra.Command{
		Use:               "rm [flags] NETWORK [NETWORK, ...]",
		Aliases:           []string{"remove"},
		Short:             "Remove one or more networks",
		Long:              "NOTE: network in use is deleted without caution",
		Args:              cobra.MinimumNArgs(1),
		RunE:              networkRmAction,
		ValidArgsFunction: networkRmShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
	return networkRmCommand
}

func networkRmAction(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := newClient(cmd)
	if err != nil {
		return err
	}
	defer cancel()
	cniPath, err := cmd.Flags().GetString("cni-path")
	if err != nil {
		return err
	}
	cniNetconfpath, err := cmd.Flags().GetString("cni-netconfpath")
	if err != nil {
		return err
	}
	e, err := netutil.NewCNIEnv(cniPath, cniNetconfpath)
	if err != nil {
		return err
	}
	containers, err := getContainersInAllNamespaces(ctx, client)
	if err != nil {
		return err
	}
	usedNetworkInfo, err := netutil.UsedNetworks(ctx, containers)
	if err != nil {
		return err
	}
	netMap := e.NetworkMap()

	for _, name := range args {
		if name == "host" || name == "none" {
			return fmt.Errorf("pseudo network %q cannot be removed", name)
		}
		net, ok := netMap[name]
		if !ok {
			return fmt.Errorf("no such network: %s", name)
		}
		if value, ok := usedNetworkInfo[name]; ok {
			return fmt.Errorf("network %q is in use by container %q", name, value)
		}
		if net.NerdctlID == nil {
			return fmt.Errorf("%s is managed outside nerdctl and cannot be removed", name)
		}
		if net.File == "" {
			return fmt.Errorf("%s is a pre-defined network and cannot be removed", name)
		}
		if err := e.RemoveNetwork(net); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), name)
	}
	return nil
}

func networkRmShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show network names, including "bridge"
	exclude := []string{netutil.DefaultNetworkName, "host", "none"}
	return shellCompleteNetworkNames(cmd, exclude)
}
