package builtin

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type orchestratorModule string

// ModuleLogger default logger
var ModuleLogger *log.Logger

var (
	defaultRegistry = "cybero.com"
)

func (g *orchestratorModule) Name() string {
	return "Orchestrator Module"
}

func (g *orchestratorModule) Version() string {
	return "0.0.1"
}

func (g *orchestratorModule) Info() string {
	return "Controls the creation of container on the host machine"
}

func (g *orchestratorModule) Help(action string) string {
	return "Not Implemented yet"
}

func (g *orchestratorModule) HandleRequest(w http.ResponseWriter, r *http.Request) error {

	return nil
}

func createNetwork(name string, labels map[string]string) error {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	options := types.NetworkCreate{
		Labels:         labels,
		Attachable:     true,
		CheckDuplicate: true,
	}

	_, err = dockerClient.NetworkCreate(context.Background(), name, options)

	ModuleLogger.Printf("Orchestrator: Created a new network %s\n", name)
	return err
}

func createVolume(name string, labels map[string]string) error {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	options := volume.VolumeCreateBody{
		Labels: labels,
		Name:   name,
	}
	_, err = dockerClient.VolumeCreate(context.Background(), options)

	ModuleLogger.Printf("Orchestrator: Created a new volume %s\n", name)
	return err
}

func attachNetwork(containerID string, networkID string, aliases []string) error {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	options := network.EndpointSettings{
		Aliases: aliases,
	}

	err = dockerClient.NetworkConnect(context.Background(), networkID, containerID, &options)

	return err
}

func executeCommand(containerID string, cmd []string, env []string) (string, int, error) {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
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
	execID, err := dockerClient.ContainerExecCreate(context.Background(), containerID, execConfig)

	if err != nil {
		return "", 0, err
	}

	startCheck := types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	}

	// We execute and hijack the stdout
	attach, err := dockerClient.ContainerExecAttach(context.Background(), execID.ID, startCheck)

	if err != nil {
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
				ModuleLogger.Printf("Error reading from process stdout:%v", err)
				break
			}
		}
	}()

	// We monitor for the execution to finish
	go func() {
		for {
			execInspect, err := dockerClient.ContainerExecInspect(context.Background(), execID.ID)
			if execInspect.Running == false || err != nil {
				exitCode = execInspect.ExitCode
				close(done)
				break
			}
		}
	}()

	<-done
	return stdout, exitCode, nil
}

func putDirectory(name string, path string, directory string) {

}

func putFile(name string, path string, file string) {

}

func putArchive(name string, path string, archive string) {

}

func snapshot(container string, restart bool, latest string, snapshot string) {

}

func stop(container string, remove bool) {

}

func rollback(container string, tag string, restart bool, rollback string) {

}

func stopAll(labels map[string]interface{}) {

}

func launch() {

}

func get(name string) {

}

func wait(name string, timeout int, condition string) {

}

func commitImage(container string, tag string) {

}

func removeImage(repo string, tag string) {

}

func tagImage(repo string, tag string, newTag string) {

}

func pruneImages() {

}

// OrchestratorModule the exported plugin
var OrchestratorModule orchestratorModule
