// Copyright 2020 Alexandre Pires (c.alexandre.pires@gmail.com)

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"cybero/types"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/system"
)

type orchestratorModule struct {
	types.RestAPIModule
}

// ContainerInfo container information
type ContainerInfo struct {
	Name         string
	Hostname     string
	Image        string
	Network      string
	Cmd          []string
	Ports        []string
	AutoRemove   bool
	Labels       map[string]string
	Mounts       map[string]string
	Env          []string
	Volumes      []string
	Privileged   bool
	Devices      []string
	Groups       []string
	UID          int
	Capabilities []string
}

// defaultLogger default logger
var defaultLogger *log.Logger

var (
	defaultRegistry   = "cybero.com"
	moduleInitialized = false
)

func resolveLocalPath(localPath string) (absPath string, err error) {
	if absPath, err = filepath.Abs(localPath); err != nil {
		return
	}

	return archive.PreserveTrailingDotOrSeparator(absPath, localPath, filepath.Separator), nil
}

func (mod orchestratorModule) Initialize(logger *log.Logger, config map[string]interface{}) error {
	defaultLogger = logger
	moduleInitialized = true
	defaultLogger.Printf("Orchestrator: Initializing module\n")
	return nil
}

func (mod orchestratorModule) IsInitialized() bool {
	return moduleInitialized
}

func (mod orchestratorModule) Name() string {
	return "Orchestrator Module"
}

func (mod orchestratorModule) Version() string {
	return "0.0.1"
}

func (mod orchestratorModule) Info() string {
	return "Controls the creation of container on the host machine"
}

func (mod orchestratorModule) Help(action string) string {
	return "Not Implemented yet"
}

func (mod orchestratorModule) HandleRequest(w http.ResponseWriter, r *http.Request) error {

	return nil
}

func (mod orchestratorModule) CreateNetwork(name string, labels map[string]string) error {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	ctx := context.Background()

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error creating network: %v\n", err)
		return err
	}

	options := types.NetworkCreate{
		Labels:         labels,
		Attachable:     true,
		CheckDuplicate: true,
	}

	_, err = dockerClient.NetworkCreate(ctx, name, options)

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error creating network: %v\n", err)
		return err
	}

	defaultLogger.Printf("Orchestrator: Network %q created\n", name)
	return nil
}

func (mod orchestratorModule) CreateVolume(name string, labels map[string]string) error {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	ctx := context.Background()

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error creating volume: %v\n", err)
		return err
	}

	options := volume.VolumeCreateBody{
		Labels: labels,
		Name:   name,
	}
	_, err = dockerClient.VolumeCreate(ctx, options)

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error creating volume: %v\n", err)
		return err
	}

	defaultLogger.Printf("Orchestrator: Volume %q created\n", name)
	return nil
}

func (mod orchestratorModule) AttachNetwork(containerID string, networkID string, aliases []string) error {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	ctx := context.Background()

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error attaching container to network: %v\n", err)
		return err
	}

	options := network.EndpointSettings{
		Aliases: aliases,
	}

	err = dockerClient.NetworkConnect(ctx, networkID, containerID, &options)

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error attaching container to network: %v\n", err)
		return err
	}

	defaultLogger.Printf("Orchestrator: Container %q attached to network %q\n", containerID, networkID)
	return nil
}

func (mod orchestratorModule) RunCommand(containerID string, cmd []string, env []string) (string, int, error) {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	ctx := context.Background()

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error executing command: %v\n", err)
		return "", 0, err
	}

	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStderr: true,
		AttachStdin:  false,
		AttachStdout: true,
		Env:          env,
		Tty:          false,
	}

	// We prepare the execution of the command
	execID, err := dockerClient.ContainerExecCreate(ctx, containerID, execConfig)

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error executing command: %v\n", err)
		return "", 0, err
	}

	startCheck := types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	}

	// We execute and hijack the stdout
	attach, err := dockerClient.ContainerExecAttach(ctx, execID.ID, startCheck)

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error executing command: %v\n", err)
		return "", 0, err
	}

	exitCode := -1
	stdout := ""

	done := make(chan bool, 1)

	// Create a goroutine to read execution stdout
	go func() {
		for {
			buf := make([]byte, 4096)
			_, err := attach.Reader.Read(buf)
			stdout += fmt.Sprint(buf)
			if err != nil {
				defaultLogger.Printf("Orchestrator: Error reading from process stdout: %v\n", err)
				break
			}
		}
	}()

	// We monitor for the execution to finish
	go func() {
		for {
			execInspect, err := dockerClient.ContainerExecInspect(ctx, execID.ID)
			if execInspect.Running == false || err != nil {
				exitCode = execInspect.ExitCode
				close(done)
				break
			}
		}
	}()

	<-done
	defaultLogger.Printf("Orchestrator: Command %q executed on container %q with exot code %d\n", cmd, containerID, exitCode)
	return stdout, exitCode, nil
}

func (mod orchestratorModule) CopyIntoContainer(containerID string, src string, dest string, followLink bool) error {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	ctx := context.Background()

	if err != nil {
		defaultLogger.Printf("Orchestrator: Error on putting directory: %v\n", err)
		return err
	}

	src, err = resolveLocalPath(src)
	if err != nil {
		defaultLogger.Printf("Orchestrator: Error on putting directory: %v\n", err)
		return err
	}

	dstInfo := archive.CopyInfo{Path: dest}
	dstStat, err := dockerClient.ContainerStatPath(ctx, containerID, dest)

	// If the destination is a symbolic link, we should evaluate it.
	if err == nil && dstStat.Mode&os.ModeSymlink != 0 {
		linkTarget := dstStat.LinkTarget
		if !system.IsAbs(linkTarget) {
			// Join with the parent directory.
			dstParent, _ := archive.SplitPathDirEntry(dest)
			linkTarget = filepath.Join(dstParent, linkTarget)
		}

		dstInfo.Path = linkTarget
		dstStat, err = dockerClient.ContainerStatPath(ctx, containerID, linkTarget)
	}

	// Ignore any error and assume that the parent directory of the destination
	// path exists, in which case the copy may still succeed. If there is any
	// type of conflict (e.g., non-directory overwriting an existing directory
	// or vice versa) the extraction will fail. If the destination simply did
	// not exist, but the parent directory does, the extraction will still
	// succeed.
	if err == nil {
		dstInfo.Exists, dstInfo.IsDir = true, dstStat.Mode.IsDir()
	}

	var (
		content         io.Reader
		resolvedDstPath string
	)

	srcInfo, err := archive.CopyInfoSourcePath(src, followLink)

	if err != nil {
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		defaultLogger.Printf("Orchestrator: Error on putting directory: %v\n", err)
		return err
	}
	defer srcArchive.Close()

	// With the stat info about the local source as well as the
	// destination, we have enough information to know whether we need to
	// alter the archive that we upload so that when the server extracts
	// it to the specified directory in the container we get the desired
	// copy behavior.

	// See comments in the implementation of `archive.PrepareArchiveCopy`
	// for exactly what goes into deciding how and whether the source
	// archive needs to be altered for the correct copy behavior when it is
	// extracted. This function also infers from the source and destination
	// info which directory to extract to, which may be the parent of the
	// destination that the user specified.
	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		defaultLogger.Printf("Orchestrator: Error on putting directory: %v\n", err)
		return err
	}
	defer preparedArchive.Close()

	resolvedDstPath = dstDir
	content = preparedArchive

	copyOption := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
	}

	err = dockerClient.CopyToContainer(ctx, containerID, resolvedDstPath, content, copyOption)
	if err != nil {
		defaultLogger.Printf("Orchestrator: Error putting directory: %v\n", err)
		return err
	}

	defaultLogger.Printf("Orchestrator: Sucessfully put %q inside container %q on destination %q\n", src, containerID, dest)
	return nil
}

func (mod orchestratorModule) Launch(containerInfo *ContainerInfo) (string, error) {
	return "", errors.New("Not implemented")
}

func (mod orchestratorModule) Stop(containerID string, remove bool) error {
	return errors.New("Not implemented")
}

func (mod orchestratorModule) StopAll(labels map[string]interface{}) error {
	return errors.New("Not implemented")
}

func (mod orchestratorModule) Get(name string) (string, error) {
	return "", errors.New("Not implemented yet")
}

func (mod orchestratorModule) Wait(name string, timeout int, condition string) error {
	return errors.New("Not implemented yet")
}

func (mod orchestratorModule) CommitImage(container string, tag string) error {
	return errors.New("Not implemented yet")
}

func (mod orchestratorModule) RemoveImage(repo string, tag string) error {
	return errors.New("Not implemented yet")
}

func (mod orchestratorModule) TagImage(repo string, tag string, newTag string) error {
	return errors.New("Not implemented yet")
}

func (mod orchestratorModule) PruneImages() error {
	return errors.New("Not implemented yet")
}

func (mod orchestratorModule) Snapshot(container string, restart bool, latest string, snapshot string) error {
	return errors.New("Not implemented yet")
}

func (mod orchestratorModule) Rollback(container string, tag string, restart bool, rollback string) error {
	return errors.New("Not implemented yet")
}

func main() {
	// Nothing here, we are a module
}

// CyberoModule the exported plugin
var CyberoModule orchestratorModule
