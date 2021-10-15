//
// Copyright (c) 2021 Ant Group
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"path"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
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
	conn       net.Conn
	containers map[string]*unikernelContainer
}

type unikernelContainer struct {
	ch     chan string
	exitch chan struct{}
	closed bool
}

func (u *unikernelAgent) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "unikernel_agent").WithField("subsystem", "libhermit")
}

// nolint:golint
func NewUnikernelAgent() agent {
	return &unikernelAgent{containers: make(map[string]*unikernelContainer)}
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

func (u *unikernelAgent) startSandbox(ctx context.Context, sandbox *Sandbox) error {
	//u.Logger().Infof("startSandbox %s", sandbox.id, string(debug.Stack()))
	targetNS, err := ns.GetNS(sandbox.networkNS.NetNsPath)
	if err != nil {
		return err
	}
	if err := targetNS.Set(); err != nil {
		return err
	}
	u.conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", unikernelUrl)
	if err != nil {
		err = fmt.Errorf("startSandbox %s failed with %s\n", unikernelUrl, err)
		u.Logger().Error(err)
		return err
	}

	return nil
}

// stopSandbox is the Noop agent Sandbox stopping implementation. It does nothing.
func (u *unikernelAgent) stopSandbox(ctx context.Context, sandbox *Sandbox) error {
	u.conn.Close()
	return nil
}

// createContainer is the Noop agent Container creation implementation. It does nothing.
func (u *unikernelAgent) createContainer(ctx context.Context, sandbox *Sandbox, c *Container) (*Process, error) {
	return &Process{}, nil
}

// startContainer is the Noop agent Container starting implementation. It does nothing.
func (u *unikernelAgent) startContainer(ctx context.Context, sandbox *Sandbox, c *Container) error {
	//u.Logger().Infof("startContainer sid %s cid %s %+v", sandbox.id, c.id, c)
	//u.Logger().Infof("startContainer sid %s cid %s %+v", sandbox.id, c.id, c.config)
	if c.id == sandbox.id {
		return nil
	}

	if len(c.config.Cmd.Args) != 1 {
		err := fmt.Errorf("startContainer cid %s command is not set", c.id)
		u.Logger().Error(err)
		return err
	}
	cmd := ""
	cmd_dir := filepath.Dir(c.config.Cmd.Args[0])
	for _, mount := range c.mounts {
		if mount.Destination == cmd_dir {
			cmd = path.Join(mount.Source, filepath.Base(c.config.Cmd.Args[0]))
			break
		}
	}
	if cmd == "" {
		cmd = path.Join(c.rootFs.Target, c.config.Cmd.Args[0])
	}
	u.Logger().Infof("startContainer cmd %s", cmd)

	err := u.writeString("c" + c.id + "," + cmd)
	if err != nil {
		err = fmt.Errorf("startContainer writeString failed with %s", err)
		u.Logger().Error(err)
		return err
	}

	u.containers[c.id] = &unikernelContainer{ch: make(chan string, 100), exitch: make(chan struct{}, 0), closed: false}

	return nil
}

// stopContainer is the Noop agent Container stopping implementation. It does nothing.
func (u *unikernelAgent) stopContainer(ctx context.Context, sandbox *Sandbox, c Container) error {
	u.Logger().Infof("stopContainer sid %s cid %s", sandbox.id, c.id)
	uc, ok := u.containers[c.id]
	if ok {
		if !uc.closed {
			close(uc.ch)
			uc.closed = true
		}
	} else {
		u.conn.Close()
	}
	return nil
}

// signalProcess is the Noop agent Container signaling implementation. It does nothing.
func (u *unikernelAgent) signalProcess(ctx context.Context, c *Container, processID string, signal syscall.Signal, all bool) error {
	u.Logger().Infof("signalProcess cid %s signal %+v", c.id, signal)
	uc, ok := u.containers[c.id]
	if ok {
		if !uc.closed {
			close(uc.ch)
			uc.closed = true
		}
	} else {
		u.writeString("k")
		u.conn.Close()
	}
	return nil
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

func (u *unikernelAgent) writeString(content string) error {
	writer := bufio.NewWriter(u.conn)
	_, err := writer.WriteString(content)
	if err == nil {
		err = writer.Flush()
	}
	return err
}

func (u *unikernelAgent) readString() (string, error) {
	var buf [512]byte
	n, err := u.conn.Read(buf[0:])
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

// waitProcess is the Noop agent process waiter. It does nothing.
func (u *unikernelAgent) waitProcess(ctx context.Context, c *Container, processID string) (int32, error) {
	u.Logger().Infof("waitProcess cid %s pid %s %s", c.id, processID, string(debug.Stack()))

	uc, ok := u.containers[c.id]
	if ok {
		<-uc.exitch
	} else {
		buf := ""
		data_size := 0
		need_read := true
		for {
			u.Logger().Infof("waitProcess loop buf %s data_size %d need_read %v", buf, data_size, need_read)
			if need_read {
				gots, err := u.readString()
				if err != nil {
					if err == io.EOF {
						u.Logger().Infof("waitProcess conn closed")
						break
					}
					err = fmt.Errorf("waitProcess readString failed with %s", err)
					u.Logger().Error(err)
					return 0, err
				}
				u.Logger().Infof("waitProcess readString %s", gots)
				buf += gots
			} else {
				// Reopen need_read to make true is the default value of need_read.
				// if don't need read in following, set need_read to false.
				need_read = true
			}

			if data_size == 0 {
				s := strings.SplitN(buf, ":", 2)
				if len(s) != 2 {
					continue
				}
				u64, err := strconv.ParseUint(s[0], 10, 64)
				if err != nil {
					err = fmt.Errorf("waitProcess format of %s is not right %s", s[0], err)
					u.Logger().Error(err)
					return 0, err
				}
				data_size = int(u64)
				buf = s[1]
			}

			// setup data
			if len(buf) < data_size {
				continue
			}
			data := buf[:data_size]

			buf = buf[data_size:]
			data_size = 0
			if len(buf) > 0 {
				need_read = false
			}

			// handle data
			s := strings.SplitN(data, ":", 2)
			if len(s) != 2 {
				u.Logger().Errorf("waitProcess data %s format is not right", data)
			}
			u.Logger().Infof("waitProcess send %s to %s", s[1], s[0])
			u.containers[s[0]].ch <- s[1]
		}
	}
	u.Logger().Infof("waitProcess cid %s exit", c.id)

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
func (u *unikernelAgent) readProcessStdout(ctx context.Context, c *Container, processID string, data []byte) (int, error) {
	//u.Logger().Infof("readProcessStdout cid %s", c.id)
	v, ok := <-u.containers[c.id].ch
	if !ok {
		u.Logger().Infof("readProcessStdout cid %s exitch", c.id)
		close(u.containers[c.id].exitch)
		return 0, io.EOF
	}
	copy(data, v)

	//u.Logger().Infof("readProcessStdout cid %s %s %d", c.id, v, len(v))
	//u.Logger().Infof("readProcessStdout cid %s %s %s", c.id, string(debug.Stack()))

	return len(v), nil
}

// readProcessStderr is the Noop agent process stderr reader. It does nothing.
func (u *unikernelAgent) readProcessStderr(ctx context.Context, c *Container, processID string, data []byte) (int, error) {
	//u.Logger().Infof("readProcessStderr cid %s", c.id)
	<-u.containers[c.id].exitch
	return 0, io.EOF
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
