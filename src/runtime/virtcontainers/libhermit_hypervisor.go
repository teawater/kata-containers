//
// Copyright (c) 2021 Ant Group
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"

	persistapi "github.com/kata-containers/kata-containers/src/runtime/virtcontainers/persist/api"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers/types"
	"github.com/sirupsen/logrus"
)

//var MockHybridVSockPath = "/tmp/kata-mock-hybrid-vsock.socket"
const helloKataString = "Hello Kata libHermit WAMR!"

type libhermitHypervisor struct {
	store       persistapi.PersistDriver
	cmd         *exec.Cmd
	hermit_path string
	agent_path  string
	stdout      *bufio.Scanner
}

func (h *libhermitHypervisor) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "libhermit")
}

func (h *libhermitHypervisor) capabilities(ctx context.Context) types.Capabilities {
	caps := types.Capabilities{}
	caps.SetFsSharingSupport()
	return caps
}

func (h *libhermitHypervisor) hypervisorConfig() HypervisorConfig {
	return HypervisorConfig{}
}

func (h *libhermitHypervisor) createSandbox(ctx context.Context, id string, networkNS NetworkNamespace, hypervisorConfig *HypervisorConfig) error {
	h.hermit_path = hypervisorConfig.HypervisorPath
	h.agent_path = hypervisorConfig.KernelPath
	return nil
}

func (h *libhermitHypervisor) startSandbox(ctx context.Context, timeout int) error {
	cmdEnv := []string{}
	cmdEnv = append(cmdEnv, "HERMIT_ISLE=qemu")
	cmd := exec.CommandContext(ctx, h.hermit_path, h.agent_path)
	cmd.Env = cmdEnv

	var err error
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		err = fmt.Errorf("startSandbox cmd.StderrPipe() %s %s failed with %s\n", h.hermit_path, h.agent_path, err)
		h.Logger().Error(err)
		return err
	}
	h.stdout = bufio.NewScanner(stdout)
	/*h.stderr, err = cmd.StderrPipe()
	if err != nil {
		err = fmt.Errorf("startSandbox cmd.StderrPipe() %s %s failed with %s\n", h.hermit_path, h.agent_path, err)
		h.Logger().Error(err)
		return err
	}*/
	err = cmd.Start()
	if err != nil {
		err = fmt.Errorf("startSandbox cmd.Start() %s %s failed with %s\n", h.hermit_path, h.agent_path, err)
		h.Logger().Error(err)
		return err
	}

	h.Logger().Infof("startSandbox pid %d", cmd.Process.Pid)
	h.cmd = cmd

	/*reader := bufio.NewReader(h.stdout)*/
	if !h.stdout.Scan() {
		err = fmt.Errorf("startSandbox read stdout %s %s failed with %s\n", h.hermit_path, h.agent_path, h.stdout.Err())
		h.Logger().Error(err)
		return err
	}
	line := h.stdout.Text()
	if line != helloKataString {
		err = fmt.Errorf("startSandbox hello string is %s that it is not right", line)
		h.Logger().Error(err)
		return err
	}

	go func() {
		h.Logger().WithFields(logrus.Fields{}).Info("reading guest console begin")

		for h.stdout.Scan() {
			h.Logger().WithFields(logrus.Fields{
				"vmconsole": h.stdout.Text(),
			}).Info("reading guest console")
		}

		if err := h.stdout.Err(); err != nil {
			if err == io.EOF {
				h.Logger().Info("console watcher quits")
			} else {
				h.Logger().WithError(err).Error("Failed to read guest console logs")
			}
		}
	}()

	//time.Sleep(time.Second * 60)

	return nil
}

func (h *libhermitHypervisor) stopSandbox(ctx context.Context, waitOnly bool) error {
	h.Logger().Info("stopSandbox %s", string(debug.Stack()))
	if !waitOnly {
		h.cmd.Process.Kill()
	}
	h.cmd.Wait()

	return nil
}

func (m *libhermitHypervisor) pauseSandbox(ctx context.Context) error {
	return nil
}

func (m *libhermitHypervisor) resumeSandbox(ctx context.Context) error {
	return nil
}

func (m *libhermitHypervisor) saveSandbox() error {
	return nil
}

func (m *libhermitHypervisor) addDevice(ctx context.Context, devInfo interface{}, devType deviceType) error {
	return nil
}

func (m *libhermitHypervisor) hotplugAddDevice(ctx context.Context, devInfo interface{}, devType deviceType) (interface{}, error) {
	switch devType {
	case cpuDev:
		return devInfo.(uint32), nil
	case memoryDev:
		memdev := devInfo.(*memoryDevice)
		return memdev.sizeMB, nil
	}
	return nil, nil
}

func (m *libhermitHypervisor) hotplugRemoveDevice(ctx context.Context, devInfo interface{}, devType deviceType) (interface{}, error) {
	switch devType {
	case cpuDev:
		return devInfo.(uint32), nil
	case memoryDev:
		return 0, nil
	}
	return nil, nil
}

func (h *libhermitHypervisor) getSandboxConsole(ctx context.Context, sandboxID string) (string, string, error) {
	return consoleProtoScanner, "", nil
}

func (h *libhermitHypervisor) getSandboxConsoleScanner() (*bufio.Scanner, error) {
	return h.stdout, nil
}

func (m *libhermitHypervisor) resizeMemory(ctx context.Context, memMB uint32, memorySectionSizeMB uint32, probe bool) (uint32, memoryDevice, error) {
	return 0, memoryDevice{}, nil
}
func (m *libhermitHypervisor) resizeVCPUs(ctx context.Context, cpus uint32) (uint32, uint32, error) {
	return 0, 0, nil
}

func (m *libhermitHypervisor) disconnect(ctx context.Context) {
}

func (m *libhermitHypervisor) getThreadIDs(ctx context.Context) (vcpuThreadIDs, error) {
	vcpus := map[int]int{0: os.Getpid()}
	return vcpuThreadIDs{vcpus}, nil
}

func (m *libhermitHypervisor) cleanup(ctx context.Context) error {
	return nil
}

func (h *libhermitHypervisor) getPids() []int {
	return []int{h.cmd.Process.Pid}
}

func (m *libhermitHypervisor) getVirtioFsPid() *int {
	return nil
}

func (m *libhermitHypervisor) fromGrpc(ctx context.Context, hypervisorConfig *HypervisorConfig, j []byte) error {
	return errors.New("libhermitHypervisor is not supported by VM cache")
}

func (m *libhermitHypervisor) toGrpc(ctx context.Context) ([]byte, error) {
	return nil, errors.New("libhermitHypervisor is not supported by VM cache")
}

func (m *libhermitHypervisor) save() (s persistapi.HypervisorState) {
	return
}

func (m *libhermitHypervisor) load(s persistapi.HypervisorState) {}

func (m *libhermitHypervisor) check() error {
	return nil
}

func (m *libhermitHypervisor) generateSocket(id string) (interface{}, error) {
	return types.MockHybridVSock{
		//	UdsPath: MockHybridVSockPath,
	}, nil
}

func (m *libhermitHypervisor) isRateLimiterBuiltin() bool {
	return false
}

func (m *libhermitHypervisor) setSandbox(sandbox *Sandbox) {
}
