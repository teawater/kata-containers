//
// Copyright (c) 2021 Ant Group
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	persistapi "github.com/kata-containers/kata-containers/src/runtime/virtcontainers/persist/api"
	pbTypes "github.com/kata-containers/kata-containers/src/runtime/virtcontainers/pkg/agent/protocols"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers/pkg/agent/protocols/grpc"
	vcTypes "github.com/kata-containers/kata-containers/src/runtime/virtcontainers/pkg/types"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers/types"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const unikernelUrl = "127.0.0.1:18082"

// unikernelAgent is an Agent implementation for unikernel
type unikernelAgent struct {
	containers map[string]unikernelContainer
}

type unikernelContainer struct {
	conn net.Conn
}

func (u *unikernelAgent) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "unikernel_agent").WithField("subsystem", "libhermit")
}

// nolint:golint
func NewUnikernelAgent() agent {
	return &unikernelAgent{containers: make(map[string]unikernelContainer)}
}

// init initializes the Noop agent, i.e. it does nothing.
func (n *unikernelAgent) init(ctx context.Context, sandbox *Sandbox, config KataAgentConfig) (bool, error) {
	return false, nil
}

func (n *unikernelAgent) longLiveConn() bool {
	return false
}

// createSandbox is the Noop agent sandbox creation implementation. It does nothing.
func (n *unikernelAgent) createSandbox(ctx context.Context, sandbox *Sandbox) error {
	return nil
}

// capabilities returns empty capabilities, i.e no capabilties are supported.
func (n *unikernelAgent) capabilities() types.Capabilities {
	return types.Capabilities{}
}

// disconnect is the Noop agent connection closer. It does nothing.
func (n *unikernelAgent) disconnect(ctx context.Context) error {
	return nil
}

// exec is the Noop agent command execution implementation. It does nothing.
func (n *unikernelAgent) exec(ctx context.Context, sandbox *Sandbox, c Container, cmd types.Cmd) (*Process, error) {
	return nil, nil
}

func (u *unikernelAgent) start(ctx context.Context, sandbox *Sandbox, id string) error {
	targetNS, err := ns.GetNS(sandbox.networkNS.NetNsPath)
	if err != nil {
		return err
	}
	if err := targetNS.Set(); err != nil {
		return err
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", unikernelUrl)
	if err != nil {
		err = fmt.Errorf("startSandbox %s failed with %s\n", unikernelUrl, err)
		u.Logger().Error(err)
		return err
	}
	u.containers[id] = unikernelContainer{conn: conn}

	return nil
}

// startSandbox is the Noop agent Sandbox starting implementation. It does nothing.
func (u *unikernelAgent) startSandbox(ctx context.Context, sandbox *Sandbox) error {
	return u.start(ctx, sandbox, sandbox.id)
}

// stopSandbox is the Noop agent Sandbox stopping implementation. It does nothing.
func (u *unikernelAgent) stopSandbox(ctx context.Context, sandbox *Sandbox) error {
	u.containers[sandbox.id].conn.Close()
	return nil
}

// createContainer is the Noop agent Container creation implementation. It does nothing.
func (u *unikernelAgent) createContainer(ctx context.Context, sandbox *Sandbox, c *Container) (*Process, error) {
	return &Process{}, nil
}

// startContainer is the Noop agent Container starting implementation. It does nothing.
func (u *unikernelAgent) startContainer(ctx context.Context, sandbox *Sandbox, c *Container) error {
	if c.id == sandbox.id {
		return nil
	}

	return u.start(ctx, sandbox, c.id)
}

// stopContainer is the Noop agent Container stopping implementation. It does nothing.
func (u *unikernelAgent) stopContainer(ctx context.Context, sandbox *Sandbox, c Container) error {
	u.containers[c.id].conn.Close()
	return nil
}

// signalProcess is the Noop agent Container signaling implementation. It does nothing.
func (u *unikernelAgent) signalProcess(ctx context.Context, c *Container, processID string, signal syscall.Signal, all bool) error {
	_, err := u.containers[c.id].conn.Write([]byte("k"))
	return err
}

// updateContainer is the Noop agent Container update implementation. It does nothing.
func (n *unikernelAgent) updateContainer(ctx context.Context, sandbox *Sandbox, c Container, resources specs.LinuxResources) error {
	return nil
}

// memHotplugByProbe is the Noop agent notify meomory hotplug event via probe interface implementation. It does nothing.
func (n *unikernelAgent) memHotplugByProbe(ctx context.Context, addr uint64, sizeMB uint32, memorySectionSizeMB uint32) error {
	return nil
}

// onlineCPUMem is the Noop agent Container online CPU and Memory implementation. It does nothing.
func (n *unikernelAgent) onlineCPUMem(ctx context.Context, cpus uint32, cpuOnly bool) error {
	return nil
}

// updateInterface is the Noop agent Interface update implementation. It does nothing.
func (n *unikernelAgent) updateInterface(ctx context.Context, inf *pbTypes.Interface) (*pbTypes.Interface, error) {
	return nil, nil
}

// listInterfaces is the Noop agent Interfaces list implementation. It does nothing.
func (n *unikernelAgent) listInterfaces(ctx context.Context) ([]*pbTypes.Interface, error) {
	return nil, nil
}

// updateRoutes is the Noop agent Routes update implementation. It does nothing.
func (n *unikernelAgent) updateRoutes(ctx context.Context, routes []*pbTypes.Route) ([]*pbTypes.Route, error) {
	return nil, nil
}

// listRoutes is the Noop agent Routes list implementation. It does nothing.
func (n *unikernelAgent) listRoutes(ctx context.Context) ([]*pbTypes.Route, error) {
	return nil, nil
}

// check is the Noop agent health checker. It does nothing.
func (n *unikernelAgent) check(ctx context.Context) error {
	return nil
}

// statsContainer is the Noop agent Container stats implementation. It does nothing.
func (u *unikernelAgent) statsContainer(ctx context.Context, sandbox *Sandbox, c Container) (*ContainerStats, error) {
	u.Logger().Infof("statsContainer cid %s %s %s", c.id, string(debug.Stack()))
	return &ContainerStats{}, nil
}

// waitProcess is the Noop agent process waiter. It does nothing.
func (u *unikernelAgent) waitProcess(ctx context.Context, c *Container, processID string) (int32, error) {
	//u.Logger().Infof("waitProcess cid %s pid %s %s", c.id, processID, string(debug.Stack()))
	var buf [512]byte
	for {
		n, err := u.containers[c.id].conn.Read(buf[0:])
		if err != nil {
			if err == io.EOF {
				break
			}
			err = fmt.Errorf("waitProcess %s failed with %s\n", unikernelUrl, err)
			u.Logger().Error(err)
			return 0, err
		}
		u.Logger().Infof("waitProcess %s", string(buf[:n]))
	}

	return 0, nil
}

// winsizeProcess is the Noop agent process tty resizer. It does nothing.
func (n *unikernelAgent) winsizeProcess(ctx context.Context, c *Container, processID string, height, width uint32) error {
	return nil
}

// writeProcessStdin is the Noop agent process stdin writer. It does nothing.
func (n *unikernelAgent) writeProcessStdin(ctx context.Context, c *Container, ProcessID string, data []byte) (int, error) {
	return 0, nil
}

// closeProcessStdin is the Noop agent process stdin closer. It does nothing.
func (n *unikernelAgent) closeProcessStdin(ctx context.Context, c *Container, ProcessID string) error {
	return nil
}

// readProcessStdout is the Noop agent process stdout reader. It does nothing.
func (n *unikernelAgent) readProcessStdout(ctx context.Context, c *Container, processID string, data []byte) (int, error) {
	return 0, nil
}

// readProcessStderr is the Noop agent process stderr reader. It does nothing.
func (n *unikernelAgent) readProcessStderr(ctx context.Context, c *Container, processID string, data []byte) (int, error) {
	return 0, nil
}

// pauseContainer is the Noop agent Container pause implementation. It does nothing.
func (n *unikernelAgent) pauseContainer(ctx context.Context, sandbox *Sandbox, c Container) error {
	return nil
}

// resumeContainer is the Noop agent Container resume implementation. It does nothing.
func (n *unikernelAgent) resumeContainer(ctx context.Context, sandbox *Sandbox, c Container) error {
	return nil
}

// configure is the Noop agent configuration implementation. It does nothing.
func (n *unikernelAgent) configure(ctx context.Context, h hypervisor, id, sharePath string, config KataAgentConfig) error {
	return nil
}

func (n *unikernelAgent) configureFromGrpc(ctx context.Context, h hypervisor, id string, config KataAgentConfig) error {
	return nil
}

// reseedRNG is the Noop agent RND reseeder. It does nothing.
func (n *unikernelAgent) reseedRNG(ctx context.Context, data []byte) error {
	return nil
}

// reuseAgent is the Noop agent reuser. It does nothing.
func (n *unikernelAgent) reuseAgent(agent agent) error {
	return nil
}

// getAgentURL is the Noop agent url getter. It returns nothing.
func (n *unikernelAgent) getAgentURL() (string, error) {
	return "", nil
}

// setAgentURL is the Noop agent url setter. It does nothing.
func (n *unikernelAgent) setAgentURL() error {
	return nil
}

// getGuestDetails is the Noop agent GuestDetails queryer. It does nothing.
func (n *unikernelAgent) getGuestDetails(context.Context, *grpc.GuestDetailsRequest) (*grpc.GuestDetailsResponse, error) {
	return nil, nil
}

// setGuestDateTime is the Noop agent guest time setter. It does nothing.
func (n *unikernelAgent) setGuestDateTime(context.Context, time.Time) error {
	return nil
}

// copyFile is the Noop agent copy file. It does nothing.
func (n *unikernelAgent) copyFile(ctx context.Context, src, dst string) error {
	return nil
}

// addSwap is the Noop agent setup swap. It does nothing.
func (n *unikernelAgent) addSwap(ctx context.Context, PCIPath vcTypes.PciPath) error {
	return nil
}

func (n *unikernelAgent) markDead(ctx context.Context) {
}

func (n *unikernelAgent) cleanup(ctx context.Context, s *Sandbox) {
}

// save is the Noop agent state saver. It does nothing.
func (n *unikernelAgent) save() (s persistapi.AgentState) {
	return
}

// load is the Noop agent state loader. It does nothing.
func (n *unikernelAgent) load(s persistapi.AgentState) {}

func (n *unikernelAgent) getOOMEvent(ctx context.Context) (string, error) {
	return "", nil
}

func (n *unikernelAgent) getAgentMetrics(ctx context.Context, req *grpc.GetMetricsRequest) (*grpc.Metrics, error) {
	return nil, nil
}
