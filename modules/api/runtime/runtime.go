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
	cyberotypes "cybero/types"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type runtimeModule struct {
	cyberotypes.RestAPIModule
}

// RuntimeInfo runtime information
type RuntimeInfo struct {
	Name         string
	Runtime      string
	Env          []string
	Devices      []specs.LinuxDevice
	Caps         []string
	Priviliged   bool
	GIDs         string
	SnapshotName string
}

// TaskInfo Task information
type TaskInfo struct {
	Name string
	Args []string
	Env  []string
	Cwd  string
	PID  uint32
}

// defaultLogger default logger
var defaultLogger *log.Logger = nil
var client *containerd.Client = nil

var (
	defaultNamespace = "cybero"
)

func (mod runtimeModule) Initialize(logger *log.Logger, config map[string]interface{}) error {
	var err error

	defaultLogger = logger

	// Connect to the containerd socket
	client, err = containerd.New("/run/containerd/containerd.sock")

	if err != nil {
		defaultLogger.Printf("Runtime: Error initializing runtime plugin\n")
	}

	defaultLogger.Printf("Runtime: Initializing module\n")
	return nil
}

func (mod runtimeModule) Name() string {
	return "Runtime Module"
}

func (mod runtimeModule) Version() string {
	return "0.0.1"
}

func (mod runtimeModule) Info() string {
	return "Controls the creation of execution runtimes"
}

func (mod runtimeModule) Help(action string) string {
	return "Not Implemented yet"
}

func (mod runtimeModule) HandleRequest(w http.ResponseWriter, r *http.Request) error {

	return nil
}

func (mod runtimeModule) ImageExists(name string) bool {

	if client == nil {
		return false
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)
	_, err := client.ImageService().Get(ctx, name)

	return err == nil
}

func (mod runtimeModule) ImagePull(name string) error {

	if client == nil {
		defaultLogger.Println("Runtime: Client not available")
		return errors.New("No client available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	_, err := client.Pull(ctx, name, containerd.WithPullUnpack)

	if err != nil {
		defaultLogger.Println("Runtime: Client not available")
		return err
	}

	defaultLogger.Printf("Runtime: Image %q sucessfully pulled\n", name)
	return nil
}

func (mod runtimeModule) ImageList() (*[]string, error) {

	if client == nil {
		defaultLogger.Println("Runtime: Client not available")
		return nil, errors.New("No client available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)
	images, err := client.ImageService().List(ctx)

	if err != nil {
		return nil, err
	}

	result := make([]string, len(images))

	for i, image := range images {
		result[i] = image.Name
	}

	return &result, nil
}

func (mod runtimeModule) RuntimeCreate(runtimeInfo *RuntimeInfo) (string, error) {

	var image containerd.Image
	var err error

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	if !mod.ImageExists(runtimeInfo.Runtime) {

		image, err = client.Pull(ctx, runtimeInfo.Runtime, containerd.WithPullUnpack)

		if err != nil {
			defaultLogger.Println("Runtime: Error pulling image")
			return "", err
		}

		defaultLogger.Printf("Runtime: Image %q sucessfully pulled\n", runtimeInfo.Runtime)
	} else {

		image, err = client.GetImage(ctx, runtimeInfo.Runtime)

		if err != nil {
			defaultLogger.Printf("Runtime: Error pulling image: %v\n", err)
			return "", err
		}
	}

	// TODO: add more specs like devices, env, etc
	newOpts := containerd.WithNewSpec(
		oci.WithImageConfig(image),
	)

	container, err := client.NewContainer(
		ctx,
		runtimeInfo.Name,
		containerd.WithNewSnapshot(runtimeInfo.SnapshotName, image),
		newOpts,
	)

	if err != nil {
		defaultLogger.Printf("Runtime: Error creating runtime: %v\n", err)
		return "", err
	}

	defaultLogger.Printf("Runtime: Runtime %q sucessfully created\n", runtimeInfo.Name)
	return container.ID(), nil
}

func (mod runtimeModule) RuntimeDestroy(runtimeID string) error {

	if client == nil {
		defaultLogger.Println("Runtime: Client not available")
		return errors.New("No client available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := client.LoadContainer(ctx, runtimeID)

	if err != nil {
		defaultLogger.Println("Runtime: client.LoadContainer")
		return err
	}

	// Force a kill request of the associated
	_, err = client.TaskService().Kill(ctx, &tasks.KillRequest{
		ContainerID: container.ID(),
		Signal:      9,
		All:         true,
	})

	// Force delete the associated task
	_, err = client.TaskService().Delete(ctx, &tasks.DeleteTaskRequest{
		ContainerID: container.ID(),
	})

	err = container.Delete(ctx, containerd.WithSnapshotCleanup)

	if err != nil {
		defaultLogger.Printf("Runtime: Error container.Delete: %v\n", err)
		return err
	}

	defaultLogger.Printf("Runtime: Runtime %q sucessfully destroyed\n", runtimeID)
	return nil
}

func (mod runtimeModule) RuntimeExists(runtimeID string) bool {

	if client == nil {
		defaultLogger.Println("Runtime: Client not available")
		return false
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	_, err := client.ContainerService().Get(ctx, runtimeID)

	return err == nil
}

func (mod runtimeModule) RuntimeExec(runtimeID string, taskInfo *TaskInfo) error {

	if client == nil {
		defaultLogger.Println("Runtime: Client not available")
		return errors.New("Client not available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := client.LoadContainer(ctx, runtimeID)

	if err != nil {
		defaultLogger.Printf("Runtime: Runtime does not exits: %v\n", err)
		return err
	}

	ciOptions := cio.NewCreator(
		cio.WithStreams(nil, defaultLogger.Writer(), defaultLogger.Writer()),
	)

	task, err := container.NewTask(ctx, ciOptions)

	if err != nil {
		defaultLogger.Printf("Runtime: Error preparing task execution environment: %v\n", err)
		return err
	}

	_, err = task.Wait(ctx)
	if err != nil {
		defaultLogger.Printf("Runtime: Error preparing task execution environment: %v\n", err)
		return err
	}

	execOpts := &specs.Process{
		Args: taskInfo.Args,
		Env:  taskInfo.Env,
		Cwd:  taskInfo.Cwd,
	}

	process, err := task.Exec(ctx, taskInfo.Name, execOpts, ciOptions)

	if err != nil {
		defaultLogger.Printf("Runtime: Error task.Exec: %v\n", err)
		return err
	}

	err = process.Start(ctx)

	if err != nil {
		defaultLogger.Printf("Runtime: Error process.Start: %v\n", err)
		return err
	}

	_, err = process.Wait(ctx)
	if err != nil {
		defaultLogger.Printf("Runtime: Error process.Wait: %v\n", err)
		return err
	}

	taskInfo.PID = process.Pid()

	return nil
}

func (mod runtimeModule) RuntimeKill(runtimeID string, taskInfo *TaskInfo, signal syscall.Signal) error {

	if client == nil {
		defaultLogger.Println("Runtime: Client not available")
		return errors.New("Client not available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := client.LoadContainer(ctx, runtimeID)

	if err != nil {
		defaultLogger.Printf("Runtime: Runtime does not exits: %v\n", err)
		return err
	}

	ciAttach := cio.NewAttach(
		cio.WithStreams(nil, defaultLogger.Writer(), defaultLogger.Writer()),
	)

	task, err := container.Task(ctx, ciAttach)
	if err != nil {
		defaultLogger.Printf("Runtime: Error container.Task: %v\n", err)
		return err
	}

	process, err := task.LoadProcess(ctx, taskInfo.Name, ciAttach)
	if err != nil {
		defaultLogger.Printf("Runtime: Error task.LoadProcess: %v\n", err)
		return err
	}

	err = process.Kill(ctx, signal)
	if err != nil {
		defaultLogger.Printf("Runtime: Error process.Kill: %v\n", err)
		return err
	}

	return nil
}

func main() {

	logFile, err := os.OpenFile("/tmp/runtime.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	defer logFile.Close()

	// Initialize logging to a file
	if err != nil {
		log.Println(err)
		return
	}

	logger := log.New(logFile, "", log.LstdFlags)

	test := runtimeModule{}
	test.Initialize(logger, nil)
	test.ImagePull("docker.io/library/redis:alpine")
	a, _ := test.ImageList()

	fmt.Println(a)

	rInfo := &RuntimeInfo{
		Name:         "redis-manager-test",
		Runtime:      "docker.io/library/redis:alpine",
		SnapshotName: "redis-manager-test-snapshot",
	}

	rID, err := test.RuntimeCreate(rInfo)

	if err != nil {
		fmt.Println("oops, something went wrong creating runtime!!")
		return
	}

	fmt.Println(rID)
	time.Sleep(5 * time.Second)

	task := TaskInfo{
		Name: "command",
		Args: []string{"/bin/ls"},
		Cwd:  "/",
	}

	err = test.RuntimeExec(rID, &task)

	if err == nil {
		fmt.Println(task.PID)
		time.Sleep(5 * time.Second)
	} else {
		fmt.Println("oops, something went wrong executing command!!")
	}

	err = test.RuntimeDestroy(rID)

	if err != nil {
		fmt.Println("oops, something went wrong destroying runtime!!")
		return
	}

	// Nothing here, we are a module
}

// CyberoRestHandler the exported plugin
var CyberoRestHandler runtimeModule
