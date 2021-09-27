//
// Copyright (c) 2021 Ant Group
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"context"
	"errors"
	"os"

	persistapi "github.com/kata-containers/kata-containers/src/runtime/virtcontainers/persist/api"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers/types"
)

//var MockHybridVSockPath = "/tmp/kata-mock-hybrid-vsock.socket"

type libhermitHypervisor struct {
	store        persistapi.PersistDriver
	libhermitPid int
}

func (m *libhermitHypervisor) capabilities(ctx context.Context) types.Capabilities {
	caps := types.Capabilities{}
	caps.SetFsSharingSupport()
	return caps
}

func (m *libhermitHypervisor) hypervisorConfig() HypervisorConfig {
	return HypervisorConfig{}
}

func (m *libhermitHypervisor) createSandbox(ctx context.Context, id string, networkNS NetworkNamespace, hypervisorConfig *HypervisorConfig) error {
	return nil
}

func (m *libhermitHypervisor) startSandbox(ctx context.Context, timeout int) error {
	return nil
}

func (m *libhermitHypervisor) stopSandbox(ctx context.Context, waitOnly bool) error {
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

func (m *libhermitHypervisor) getSandboxConsole(ctx context.Context, sandboxID string) (string, string, error) {
	return "", "", nil
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

func (m *libhermitHypervisor) getPids() []int {
	return []int{m.libhermitPid}
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
