/*
Copyright 2016 The Kubernetes Authors.

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

package manager

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/frakti/pkg/runtime"
	kubeapi "k8s.io/kubernetes/pkg/kubelet/api/v1alpha1/runtime"
)

const (
	runtimeAPIVersion = "0.1.0"
)

// FraktiManager serves the kubelet runtime gRPC api which will be
// consumed by kubelet
type FraktiManager struct {
	// The grpc server.
	server *grpc.Server

	runtimeService runtime.RuntimeService
	imageService   runtime.ImageService
}

// NewFraktiManager creates a new FraktiManager
func NewFraktiManager(runtimeService runtime.RuntimeService, imageService runtime.ImageService) (*FraktiManager, error) {
	s := &FraktiManager{
		server:         grpc.NewServer(),
		runtimeService: runtimeService,
		imageService:   imageService,
	}
	s.registerServer()

	return s, nil
}

// Serve starts gRPC server at unix://addr
func (s *FraktiManager) Serve(addr string) error {
	glog.V(1).Infof("Start frakti at %s", addr)

	if err := syscall.Unlink(addr); err != nil && !os.IsNotExist(err) {
		return err
	}

	lis, err := net.Listen("unix", addr)
	if err != nil {
		glog.Fatalf("Failed to listen %s: %v", addr, err)
		return err
	}

	defer lis.Close()
	return s.server.Serve(lis)
}

func (s *FraktiManager) registerServer() {
	kubeapi.RegisterRuntimeServiceServer(s.server, s)
	kubeapi.RegisterImageServiceServer(s.server, s)
}

// Version returns the runtime name, runtime version and runtime API version
func (s *FraktiManager) Version(ctx context.Context, req *kubeapi.VersionRequest) (*kubeapi.VersionResponse, error) {
	runtimeName, version, apiVersion, err := s.runtimeService.Version()
	if err != nil {
		glog.Errorf("Get version from runtime service failed: %v", err)
		return nil, err
	}

	kubeletAPIVersion := runtimeAPIVersion
	return &kubeapi.VersionResponse{
		Version:           &kubeletAPIVersion,
		RuntimeName:       &runtimeName,
		RuntimeVersion:    &version,
		RuntimeApiVersion: &apiVersion,
	}, nil
}

// CreatePodSandbox creates a hyper Pod
func (s *FraktiManager) CreatePodSandbox(ctx context.Context, req *kubeapi.CreatePodSandboxRequest) (*kubeapi.CreatePodSandboxResponse, error) {
	glog.V(3).Infof("CreatePodSandbox with request %s", req.String())

	podID, err := s.runtimeService.CreatePodSandbox(req.Config)
	if err != nil {
		glog.Errorf("CreatePodSandbox from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.CreatePodSandboxResponse{PodSandboxId: &podID}, nil
}

// StopPodSandbox stops the sandbox.
func (s *FraktiManager) StopPodSandbox(ctx context.Context, req *kubeapi.StopPodSandboxRequest) (*kubeapi.StopPodSandboxResponse, error) {
	glog.V(3).Infof("StopPodSandbox with request %s", req.String())

	err := s.runtimeService.StopPodSandbox(req.GetPodSandboxId())
	if err != nil {
		glog.Errorf("StopPodSandbox from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.StopPodSandboxResponse{}, nil
}

// DeletePodSandbox deletes the sandbox.
func (s *FraktiManager) DeletePodSandbox(ctx context.Context, req *kubeapi.DeletePodSandboxRequest) (*kubeapi.DeletePodSandboxResponse, error) {
	glog.V(3).Infof("DeletePodSandbox with request %s", req.String())

	err := s.runtimeService.DeletePodSandbox(req.GetPodSandboxId())
	if err != nil {
		glog.Errorf("DeletePodSandbox from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.DeletePodSandboxResponse{}, nil
}

// PodSandboxStatus returns the Status of the PodSandbox.
func (s *FraktiManager) PodSandboxStatus(ctx context.Context, req *kubeapi.PodSandboxStatusRequest) (*kubeapi.PodSandboxStatusResponse, error) {
	glog.V(3).Infof("PodSandboxStatus with request %s", req.String())

	podStatus, err := s.runtimeService.PodSandboxStatus(req.GetPodSandboxId())
	if err != nil {
		glog.Errorf("PodSandboxStatus from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.PodSandboxStatusResponse{Status: podStatus}, nil
}

// ListPodSandbox returns a list of SandBox.
func (s *FraktiManager) ListPodSandbox(ctx context.Context, req *kubeapi.ListPodSandboxRequest) (*kubeapi.ListPodSandboxResponse, error) {
	glog.V(3).Infof("ListPodSandbox with request %s", req.String())

	items, err := s.runtimeService.ListPodSandbox(req.GetFilter())
	if err != nil {
		glog.Errorf("ListPodSandbox from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.ListPodSandboxResponse{Items: items}, nil
}

// CreateContainer creates a new container in specified PodSandbox
func (s *FraktiManager) CreateContainer(ctx context.Context, req *kubeapi.CreateContainerRequest) (*kubeapi.CreateContainerResponse, error) {
	glog.V(3).Infof("CreateContainer with request %s", req.String())

	containerID, err := s.runtimeService.CreateContainer(req.GetPodSandboxId(), req.Config, req.SandboxConfig)
	if err != nil {
		glog.Errorf("CreateContainer from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.CreateContainerResponse{ContainerId: &containerID}, nil
}

// StartContainer starts the container.
func (s *FraktiManager) StartContainer(ctx context.Context, req *kubeapi.StartContainerRequest) (*kubeapi.StartContainerResponse, error) {
	glog.V(3).Infof("StartContainer with request %s", req.String())

	err := s.runtimeService.StartContainer(req.GetContainerId())
	if err != nil {
		glog.Errorf("StartContainer from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.StartContainerResponse{}, nil
}

// StopContainer stops a running container with a grace period (i.e. timeout).
func (s *FraktiManager) StopContainer(ctx context.Context, req *kubeapi.StopContainerRequest) (*kubeapi.StopContainerResponse, error) {
	glog.V(3).Infof("StopContainer with request %s", req.String())

	err := s.runtimeService.StopContainer(req.GetContainerId(), req.GetTimeout())
	if err != nil {
		glog.Errorf("StopContainer from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.StopContainerResponse{}, nil
}

// RemoveContainer removes the container.
func (s *FraktiManager) RemoveContainer(ctx context.Context, req *kubeapi.RemoveContainerRequest) (*kubeapi.RemoveContainerResponse, error) {
	glog.V(3).Infof("RemoveContainer with request %s", req.String())

	err := s.runtimeService.RemoveContainer(req.GetContainerId())
	if err != nil {
		glog.Errorf("RemoveContainer from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.RemoveContainerResponse{}, nil
}

// ListContainers lists all containers by filters.
func (s *FraktiManager) ListContainers(ctx context.Context, req *kubeapi.ListContainersRequest) (*kubeapi.ListContainersResponse, error) {
	glog.V(3).Infof("ListContainers with request %s", req.String())

	containers, err := s.runtimeService.ListContainers(req.GetFilter())
	if err != nil {
		glog.Errorf("ListContainers from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.ListContainersResponse{
		Containers: containers,
	}, nil
}

// ContainerStatus returns the container status.
func (s *FraktiManager) ContainerStatus(ctx context.Context, req *kubeapi.ContainerStatusRequest) (*kubeapi.ContainerStatusResponse, error) {
	glog.V(3).Infof("ContainerStatus with request %s", req.String())

	kubeStatus, err := s.runtimeService.ContainerStatus(req.GetContainerId())
	if err != nil {
		glog.Errorf("ContainerStatus from runtime service failed: %v", err)
		return nil, err
	}

	return &kubeapi.ContainerStatusResponse{
		Status: kubeStatus,
	}, nil
}

// Exec execute a command in the container.
func (s *FraktiManager) Exec(stream kubeapi.RuntimeService_ExecServer) error {
	// TODO: implement exec in container
	return fmt.Errorf("Not implemented")
}

// ListImages lists existing images.
func (s *FraktiManager) ListImages(ctx context.Context, req *kubeapi.ListImagesRequest) (*kubeapi.ListImagesResponse, error) {
	glog.V(3).Infof("ListImages with request %s", req.String())

	images, err := s.imageService.ListImages(req.GetFilter())
	if err != nil {
		glog.Errorf("ListImages from image service failed: %v", err)
		return nil, err
	}

	return &kubeapi.ListImagesResponse{
		Images: images,
	}, nil
}

// ImageStatus returns the status of the image.
func (s *FraktiManager) ImageStatus(ctx context.Context, req *kubeapi.ImageStatusRequest) (*kubeapi.ImageStatusResponse, error) {
	glog.V(3).Infof("ImageStatus with request %s", req.String())

	status, err := s.imageService.ImageStatus(req.Image)
	if err != nil {
		glog.Infof("ImageStatus from image service failed: %v", err)
		return nil, err
	}
	return &kubeapi.ImageStatusResponse{Image: status}, nil
}

// PullImage pulls a image with authentication config.
func (s *FraktiManager) PullImage(ctx context.Context, req *kubeapi.PullImageRequest) (*kubeapi.PullImageResponse, error) {
	glog.V(3).Infof("PullImage with request %s", req.String())

	err := s.imageService.PullImage(req.Image, req.Auth)
	if err != nil {
		glog.Errorf("PullImage from image service failed: %v", err)
		return nil, err
	}

	return &kubeapi.PullImageResponse{}, nil
}

// RemoveImage removes the image.
func (s *FraktiManager) RemoveImage(ctx context.Context, req *kubeapi.RemoveImageRequest) (*kubeapi.RemoveImageResponse, error) {
	glog.V(3).Infof("RemoveImage with request %s", req.String())

	err := s.imageService.RemoveImage(req.Image)
	if err != nil {
		glog.Errorf("RemoveImage from image service failed: %v", err)
		return nil, err
	}

	return &kubeapi.RemoveImageResponse{}, nil
}
