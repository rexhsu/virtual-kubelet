package openstack

import (
	"fmt"
	"os"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/container/v1/capsules"
	"github.com/virtual-kubelet/virtual-kubelet/manager"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ZunProvider implements the virtual-kubelet provider interface and communicates with OpenStack's Zun APIs.
type ZunProvider struct {
	ZunClient          *gophercloud.ServiceClient
	resourceManager    *manager.ResourceManager
	region             string
	nodeName           string
	operatingSystem    string
	cpu                string
	memory             string
	pods               string
}

// NewZunProvider creates a new ZunProvider.
func NewZunProvider(config string, rm *manager.ResourceManager, nodeName, operatingSystem string) (*ZunProvider, error) {
	var p ZunProvider
	var err error

	p.resourceManager = rm

	AuthOptions, err := openstack.AuthOptionsFromEnv()
	if err != nil{
		fmt.Errorf("Unable to get the Auth options from environment variables: %s", err)
		return nil, err
	}

	Provider, err := openstack.AuthenticatedClient(AuthOptions)
	if err != nil {
		fmt.Errorf("Unable to get provider: %s", err)
		return nil, err
	}

	p.ZunClient, err = openstack.NewContainerV1(Provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		fmt.Errorf("Unable to get zun client")
		return nil, err
	}

	// Set sane defaults for Capacity in case config is not supplied
	p.cpu = "20"
	p.memory = "100Gi"
	p.pods = "20"

	p.operatingSystem = operatingSystem
	p.nodeName = nodeName

	return &p, err
}

// GetPod returns a pod by name that is running inside ACI
// returns nil if a pod by that name is not found.
func (p *ZunProvider) GetPod(namespace, name string) (*v1.Pod, error) {
	capsule, err := capsules.Get(p.ZunClient, fmt.Sprintf("%s-%s", namespace, name)).Extract()
	if err != nil {
		return nil, err
	}

	//if cg.Tags["NodeName"] != p.nodeName {
	//	return nil, nil
	//}

	return capsuleToPod(capsule)
}

func capsuleToPod(capsule *capsules.Capsule) (*v1.Pod, error) {
	var podCreationTimestamp metav1.Time

	podCreationTimestamp = metav1.NewTime(capsule.CreatedAt)
	//Zun don't record capsule start time, use update time instead
	//containerStartTime := metav1.NewTime(time.Time(cg.Containers[0].ContainerProperties.InstanceView.CurrentState.StartTime))
	containerStartTime := metav1.NewTime(capsule.UpdatedAt)

	// Deal with container inside capsule
	containers := make([]v1.Container, 0, len(capsule.Containers))
	containerStatuses := make([]v1.ContainerStatus, 0, len(capsule.Containers))
	for _, c := range capsule.Containers {
		container_command := []string{c.Command}
		container := v1.Container{
			Name:    c.Name,
			Image:   c.Image,
			Command: container_command,
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%d", int64(c.CPU))),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%gG", c.Memory)),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%d", int64(c.CPU*1024/100))),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("")),
				},
			},
		}
		containers = append(containers, container)
		containerStatus := v1.ContainerStatus{
			Name:                 c.Name,
			State:                zunContainerStausToContainerStatus(&c),
			//Zun doesn't record termination state.
			LastTerminationState: zunContainerStausToContainerStatus(&c),
			Ready:                zunStatusToPodPhase(c.Status) == v1.PodRunning,
			//Zun doesn't record restartCount.
			RestartCount:         int32(0),
			Image:                c.Image,
			ImageID:              "",
			ContainerID:          c.ContainerID,
		}

		// Add to containerStatuses
		containerStatuses = append(containerStatuses, containerStatus)
	}

	ip := ""
	if capsule.Addresses != nil {
		for _, v := range capsule.Addresses {
			for _, addr := range v {
				if addr.Version == float64(4) {
					ip = addr.Addr
				}
			}
		}
	}

	p := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              capsule.MetaLabels["PodName"],
			Namespace:         capsule.MetaLabels["Namespace"],
			ClusterName:       capsule.MetaLabels["ClusterName"],
			UID:               types.UID(capsule.UUID),
			CreationTimestamp: podCreationTimestamp,
		},
		Spec: v1.PodSpec{
			NodeName:   capsule.MetaLabels["NodeName"],
			Volumes:    []v1.Volume{},
			Containers: containers,
		},

		Status: v1.PodStatus{
			Phase:             zunCapStatusToPodPhase(capsule.Status),
			Conditions:        []v1.PodCondition{},
			Message:           "",
			Reason:            "",
			HostIP:            "",
			PodIP:             ip,
			StartTime:         &containerStartTime,
			ContainerStatuses: containerStatuses,
		},
	}

	return &p, nil
}

func zunContainerStausToContainerStatus(cs *capsules.Container) v1.ContainerState {
	// Zun already container start time but not add support at gophercloud
	//startTime := metav1.NewTime(time.Time(cs.StartTime))

	// Zun container status:
	//'Error', 'Running', 'Stopped', 'Paused', 'Unknown', 'Creating', 'Created',
	//'Deleted', 'Deleting', 'Rebuilding', 'Dead', 'Restarting'

	// Handle the case where the container is running.
	if cs.Status == "Running" || cs.Status == "Stopped"{
		return v1.ContainerState{
			Running: &v1.ContainerStateRunning{
				StartedAt: metav1.NewTime(time.Time(cs.CreatedAt)),
			},
		}
	}

	// Handle the case where the container failed.
	if cs.Status == "Error" || cs.Status == "Dead" {
		return v1.ContainerState{
			Terminated: &v1.ContainerStateTerminated{
				//ExitCode:   cs.ExitCode,
				ExitCode:   int32(0),
				Reason:     cs.Status,
				Message:    cs.StatusDetail,
				//StartedAt:  startTime,
				StartedAt:  metav1.NewTime(time.Time(cs.CreatedAt)),
				//Zun doesn't have FinishAt
				FinishedAt: metav1.NewTime(time.Time(cs.UpdatedAt)),
			},
		}
	}

	// Handle the case where the container is pending.
	// Which should be all other aci states.
	return v1.ContainerState{
		Waiting: &v1.ContainerStateWaiting{
			Reason:  cs.Status,
			Message: cs.StatusDetail,
		},
	}
}

func zunStatusToPodPhase(status string) v1.PodPhase {
	switch status {
	case "Running":
		return v1.PodRunning
	case "Stopped":
		return v1.PodSucceeded
	case "Error":
		return v1.PodFailed
	case "Dead":
		return v1.PodFailed
	case "Creating":
		return v1.PodPending
	case "Created":
		return v1.PodPending
	case "Restarting":
		return v1.PodPending
	case "Rebuilding":
		return v1.PodPending
	case "Paused":
		return v1.PodPending
	case "Deleting":
		return v1.PodPending
	case "Deleted":
		return v1.PodPending
	}

	return v1.PodUnknown
}

func zunCapStatusToPodPhase(status string) v1.PodPhase {
	switch status {
	case "Running":
		return v1.PodRunning
	case "Succeeded":
		return v1.PodSucceeded
	case "Failed":
		return v1.PodFailed
	case "Pending":
		return v1.PodPending
	}

	return v1.PodUnknown
}

// Capacity returns a resource list containing the capacity limits set for ACI.
func (p *ZunProvider) Capacity() v1.ResourceList {
	return v1.ResourceList{
		"cpu":    resource.MustParse(p.cpu),
		"memory": resource.MustParse(p.memory),
		"pods":   resource.MustParse(p.pods),
	}
}

