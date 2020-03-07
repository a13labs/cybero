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

package nodemanager

import (
	"context"
	"errors"
	"log"
	"sync"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// NodeManager the node manager implementation
type NodeManager struct {
	Logger *log.Logger
	Client *containerd.Client
}

// logger default logger

var (
	defaultNamespace = "cybero"
	nodeManagerSync  sync.Once
	nodeManager      *NodeManager
)

// RuntimeExists check if runtime exits
func (manager *NodeManager) RuntimeExists(name string) bool {

	if manager.Client == nil {
		return false
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)
	_, err := manager.Client.ImageService().Get(ctx, name)

	return err == nil
}

// RuntimePull pull a runtime from a store
func (manager *NodeManager) RuntimePull(name string) error {

	if manager.Client == nil {
		manager.Logger.Println("NodeManager: Containerd not available")
		return errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	_, err := manager.Client.Pull(ctx, name, containerd.WithPullUnpack)

	if err != nil {
		manager.Logger.Printf("NodeManager: Error on runtime pull, runtime: %q, err: %v\n", name, err)
		return err
	}

	manager.Logger.Printf("NodeManager: Runtime pulled, runtime: %q\n", name)
	return nil
}

// RuntimeList list the available runtimes
func (manager *NodeManager) RuntimeList() (*[]string, error) {

	if manager.Client == nil {
		manager.Logger.Println("NodeManager: Containerd not available")
		return nil, errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)
	runtimes, err := manager.Client.ImageService().List(ctx)

	if err != nil {
		manager.Logger.Printf("NodeManager: Error listing runtimes, err: %v\n", err)
		return nil, err
	}

	result := make([]string, len(runtimes))

	for i, runtime := range runtimes {
		result[i] = runtime.Name
	}

	return &result, nil
}

// NodeCreate create a node
func (manager *NodeManager) NodeCreate(node *Node) error {

	if manager.Client == nil {
		manager.Logger.Println("NodeManager: Containerd not available")
		return errors.New("Containerd available")
	}

	var runtime containerd.Image
	var err error

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	if manager.NodeExists(node.Name) {
		manager.Logger.Printf("NodeManager: Node runtime exists, node:%q\n", node.Name)
		return errors.New("Node already exists")
	}

	if !manager.RuntimeExists(node.Runtime) {

		runtime, err = manager.Client.Pull(ctx, node.Runtime, containerd.WithPullUnpack)

		if err != nil {
			manager.Logger.Printf("NodeManager: Error pulling runtime, runtime: %q, err: %v\n", node.Runtime, err)
			return err
		}

		manager.Logger.Printf("NodeManager: Runtime pulled, runtime: %q\n", node.Runtime)

	} else {

		runtime, err = manager.Client.GetImage(ctx, node.Runtime)

		if err != nil {
			manager.Logger.Printf("NodeManager: Error getting runtime, runtime: %q, err: %v\n", node.Runtime, err)
			return err
		}
	}

	// TODO: add more specs like devices, env, etc
	newOpts := containerd.WithNewSpec(
		oci.WithImageConfig(runtime),
	)

	snapShotname := node.Name + "-snapshot"

	_, err = manager.Client.NewContainer(
		ctx,
		node.Name,
		containerd.WithNewSnapshot(snapShotname, runtime),
		newOpts,
	)

	if err != nil {
		manager.Logger.Printf("NodeManager: Failed creating execution runtime, node %q, err: %v\n", node.Name, err)
		return err
	}

	manager.Logger.Printf("NodeManager: Execution runtime created, node: %q\n", node.Name)
	return nil

}

// NodeDestroy destroy node
func (manager *NodeManager) NodeDestroy(node *Node) error {

	if manager.Client == nil {
		manager.Logger.Println("NodeManager: Containerd not available")
		return errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := manager.Client.LoadContainer(ctx, node.Name)

	if err != nil {
		manager.Logger.Printf("NodeManager: Error loading node runtime, node: %q, err: %v\n", node.Name, err)
		return err
	}

	// Force a kill request of the associated task
	_, err = manager.Client.TaskService().Kill(ctx, &tasks.KillRequest{
		ContainerID: container.ID(),
		Signal:      9,
		All:         true,
	})

	// Force delete the associated task
	_, err = manager.Client.TaskService().Delete(ctx, &tasks.DeleteTaskRequest{
		ContainerID: container.ID(),
	})

	err = container.Delete(ctx, containerd.WithSnapshotCleanup)

	if err != nil {
		manager.Logger.Printf("NodeManager: Error destroying execution runtime, node: %q, err: %v\n", node.Name, err)
		return err
	}

	manager.Logger.Printf("NodeManager: Execution runtime destroyed, node: %q\n", node.Name)
	return nil
}

// NodeExists check if node exists
func (manager *NodeManager) NodeExists(name string) bool {

	if manager.Client == nil {
		manager.Logger.Println("NodeManager: Containerd not available")
		return false
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	_, err := manager.Client.ContainerService().Get(ctx, name)

	return err == nil
}

// NodeLoad load an existing node
func (manager *NodeManager) NodeLoad(name string) (*Node, error) {

	if manager.Client == nil {
		manager.Logger.Println("NodeManager: Containerd not available")
		return nil, errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := manager.Client.ContainerService().Get(ctx, name)

	if err != nil {
		manager.Logger.Printf("NodeManager: Error loading execution runtime, node %q, err: %v\n", name, err)
		return nil, err
	}

	runtime := &Node{
		Name:    container.ID,
		Runtime: container.Image,
		// TODO: get missing properties
	}

	manager.Logger.Printf("NodeManager: Execution runtime loaded, node: %q\n", name)
	return runtime, nil
}

// NodeExec execute a task inside a node
func (manager *NodeManager) NodeExec(node *Node, task *NodeTask) error {

	if manager.Client == nil {
		manager.Logger.Println("NodeManager: Containerd not available")
		return errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := manager.Client.LoadContainer(ctx, node.Name)

	if err != nil {
		manager.Logger.Printf("NodeManager: Execution runtime does not exits, node: %q, err: %v\n", node.Name, err)
		return err
	}

	ciOptions := cio.NewCreator(
		cio.WithStreams(nil, manager.Logger.Writer(), manager.Logger.Writer()),
	)

	taskEnv, err := container.NewTask(ctx, ciOptions)

	if err != nil {
		manager.Logger.Printf("NodeManager: Error creating task, task: %q, node %q, err: %v\n", task.Name, node.Name, err)
		return err
	}

	_, err = taskEnv.Wait(ctx)
	if err != nil {
		manager.Logger.Printf("NodeManager: Error waiting task, task: %q, node %q, err: %v\n", task.Name, node.Name, err)
		return err
	}

	execOpts := &specs.Process{
		Args: task.Args,
		Env:  task.Env,
		Cwd:  task.Cwd,
	}

	process, err := taskEnv.Exec(ctx, task.Name, execOpts, ciOptions)

	if err != nil {
		manager.Logger.Printf("NodeManager: Error creating process, task: %q, node %q, args: %q, err: %v\n", task.Name, node.Name, task.Args, err)
		return err
	}

	err = process.Start(ctx)

	if err != nil {
		manager.Logger.Printf("NodeManager: Error starting process, task: %q, node %q, args: %q, err: %v\n", task.Name, node.Name, task.Args, err)
		return err
	}

	_, err = process.Wait(ctx)
	if err != nil {
		manager.Logger.Printf("NodeManager: Error waiting process, task: %q, node %q, args: %q, err: %v\n", task.Name, node.Name, task.Args, err)
		return err
	}

	task.PID = process.Pid()
	manager.Logger.Printf("NodeManager: Task process started, pid: %d, task: %q, node %q, args: %q\n", task.PID, task.Name, node.Name, task.Args)
	return nil
}

// NodeSignal send a signal to a node
func (manager *NodeManager) NodeSignal(node *Node, task *NodeTask, signal syscall.Signal) error {

	if manager.Client == nil {
		manager.Logger.Println("NodeManager: Containerd not available")
		return errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := manager.Client.LoadContainer(ctx, node.Name)

	if err != nil {
		manager.Logger.Printf("NodeManager: Execution runtime does not exits, node: %q, err: %v\n", node.Name, err)
		return err
	}

	ciAttach := cio.NewAttach(
		cio.WithStreams(nil, manager.Logger.Writer(), manager.Logger.Writer()),
	)

	taskEnv, err := container.Task(ctx, ciAttach)
	if err != nil {
		manager.Logger.Printf("NodeManager: Execution runtime is not started, node: %q, err: %v\n", node.Name, err)
		return err
	}

	process, err := taskEnv.LoadProcess(ctx, task.Name, ciAttach)
	if err != nil {
		manager.Logger.Printf("NodeManager: Execution runtime task does not exits, task: %q, node: %q, err: %v\n", task.Name, node.Name, err)
		return err
	}

	err = process.Kill(ctx, signal)
	if err != nil {
		manager.Logger.Printf("NodeManager: Error sending signal %d to task, task: %q, node: %q, err: %v\n", signal, task.Name, node.Name, err)
		return err
	}

	manager.Logger.Printf("NodeManager: Signal %d sent, task: %q, node: %q, args:%q, pid: %d\n", signal, task.Name, node.Name, task.Args, task.PID)
	return nil
}

// GetNodeManager Initialize modules compoment
func GetNodeManager(defaultLogger *log.Logger) *NodeManager {

	var err error

	nodeManagerSync.Do(func() {

		nodeManager = &NodeManager{}
		nodeManager.Logger = defaultLogger
		defaultLogger.Printf("NodeManager: Initializing containerd\n")

		// Connect to the containerd socket
		nodeManager.Client, err = containerd.New("/run/containerd/containerd.sock")

		if err != nil {
			defaultLogger.Printf("NodeManager: Error initializing runtime plugin\n")
			return
		}
	})

	if nodeManager == nil {
		defaultLogger.Println("NodeManager: Something wen't wrong when starting Node manager!")
	}

	return nodeManager
}
