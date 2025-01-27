/*
MIT License
Copyright(c) 2022 Futurewei Cloud

	Permission is hereby granted,
	free of charge, to any person obtaining a copy of this software and associated documentation files(the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and / or sell copies of the Software, and to permit persons
	to whom the Software is furnished to do so, subject to the following conditions:
	The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
	WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package test

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	common_pb "github.com/futurewei-cloud/merak/api/proto/v1/common"
	pb "github.com/futurewei-cloud/merak/api/proto/v1/compute"
	constants "github.com/futurewei-cloud/merak/services/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestGrpcClient(t *testing.T) {
	totalPods := 5
	numVms := 5
	var compute_address strings.Builder
	compute_address.WriteString(constants.COMPUTE_GRPC_SERVER_ADDRESS)
	compute_address.WriteString(":")
	compute_address.WriteString(strconv.Itoa(constants.COMPUTE_GRPC_SERVER_PORT))
	ctx := context.Background()
	conn, err := grpc.Dial(compute_address.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial gRPC server address!: %v\n", err)
	}
	client := pb.NewMerakComputeServiceClient(conn)

	config, err := rest.InClusterConfig()
	if err != nil {
		t.Fatalf("Failed to get in cluster config!: %v\n", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create kube client!: %v\n", err.Error())
	}
	var sc corev1.SecurityContext
	pri := true
	sc.Privileged = &pri
	allow_pri := true
	sc.AllowPrivilegeEscalation = &allow_pri
	var capab corev1.Capabilities

	capab.Add = append(capab.Add, "NET_ADMIN")
	capab.Add = append(capab.Add, "SYS_TIME")
	sc.Capabilities = &capab

	podConfigs := []*common_pb.InternalComputeInfo{}
	pods := []*pb.InternalVMPod{}
	for i := 0; i < totalPods; i++ {
		newPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "vhost-" + strconv.Itoa(i),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "vhost-" + strconv.Itoa(i),
						Image:           "meraksim/merak-agent:test",
						ImagePullPolicy: constants.POD_PULL_POLICY_ALWAYS,
						SecurityContext: &sc,
						Ports: []corev1.ContainerPort{
							{ContainerPort: constants.AGENT_GRPC_SERVER_PORT},
							{ContainerPort: constants.PROMETHEUS_PORT},
						},
						Env: []corev1.EnvVar{
							{
								Name: constants.MODE_ENV, Value: constants.MODE_STANDALONE,
							},
						},
					},
				},
			},
		}
		_, err := clientset.CoreV1().Pods("default").Create(context.Background(), newPod, metav1.CreateOptions{})
		if err != nil {
			log.Fatal(err)
		}
	}
	for i := 0; i < totalPods; i++ {
		ip := ""
		hostname := ""
		for net.ParseIP(ip) == nil || hostname == "" {
			time.Sleep(2 * time.Second)
			kubePod, err := clientset.CoreV1().Pods("default").Get(ctx, "vhost-"+strconv.Itoa(i), metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Failed to get pod!: %v\n", err)
			}
			ip = kubePod.Status.PodIP
			hostname = kubePod.Spec.NodeName
		}

		t.Log("Found merak agent pod at " + ip)
		podConfig := common_pb.InternalComputeInfo{
			OperationType: common_pb.OperationType_CREATE,
			Id:            "1",
			Name:          "vhost-" + strconv.Itoa(i),
			DatapathIp:    ip,
			ContainerIp:   ip,
			Mac:           "aa:bb:cc:dd:ee",
			Veth:          "test",
			Hostname:      hostname,
		}
		pod := pb.InternalVMPod{
			OperationType: common_pb.OperationType_CREATE,
			PodIp:         ip,
			Subnets:       []string{"subnet1"},
			NumOfVm:       1,
		}
		podConfigs = append(podConfigs, &podConfig)
		pods = append(pods, &pod)
	}

	subnets := common_pb.InternalSubnetInfo{
		SubnetId:   "8182a4d4-ffff-4ece-b3f0-8d36e3d88000",
		SubnetCidr: "10.0.1.0/24",
		SubnetGw:   "10.0.1.1",
		NumberVms:  uint32(numVms),
	}
	vpc := common_pb.InternalVpcInfo{
		VpcId:     "9192a4d4-ffff-4ece-b3f0-8d36e3d88001",
		Subnets:   []*common_pb.InternalSubnetInfo{&subnets},
		ProjectId: "123456789",
		TenantId:  "123456789",
	}
	deploy := pb.InternalVMDeployInfo{
		OperationType: common_pb.OperationType_CREATE,
		DeployType:    pb.VMDeployType_UNIFORM,
		Vpcs:          []*common_pb.InternalVpcInfo{&vpc},
		Secgroups:     []string{"3dda2801-d675-4688-a63f-dcda8d111111"},
		Scheduler:     pb.VMScheduleType_SEQUENTIAL,
		DeployMethod:  pods,
	}

	service := common_pb.InternalServiceInfo{
		OperationType: common_pb.OperationType_CREATE,
		Id:            "2",
		Name:          "test",
		Cmd:           "10.224",
		Url:           "project/123456789/ports",
		Parameters:    []string{"test1", "test2"},
		ReturnCode:    []uint32{0},
		ReturnString:  []string{"success"},
		WhenToRun:     "now",
		WhereToRun:    "here",
	}

	computeConfig := pb.InternalComputeConfiguration{
		FormatVersion:   1,
		RevisionNumber:  1,
		RequestId:       "test",
		ComputeConfigId: "test",
		MessageType:     common_pb.MessageType_FULL,
		Pods:            podConfigs,
		VmDeploy:        &deploy,
		Services:        []*common_pb.InternalServiceInfo{&service},
		ExtraInfo:       &pb.InternalComputeExtraInfo{Info: "test"},
	}

	compute_info := pb.InternalComputeConfigInfo{
		OperationType: common_pb.OperationType_CREATE,
		Config:        &computeConfig,
	}

	// Test Create
	resp, err := client.ComputeHandler(ctx, &compute_info)
	if err != nil {
		t.Fatalf("Compute Handler Create failed: %v", err)
	}
	t.Log("Response: ", resp.ReturnMessage)

	// // Test Info
	// compute_info.OperationType = common_pb.OperationType_INFO
	// resp, err = client.ComputeHandler(ctx, &compute_info)
	// if err != nil {
	// 	t.Fatalf("Compute Handler Info failed: %v", err)
	// }
	// t.Log("Response: ", resp.ReturnMessage)

	// // Test Delete
	// compute_info.OperationType = common_pb.OperationType_DELETE
	// resp, err = client.ComputeHandler(ctx, &compute_info)
	// if err != nil {
	// 	t.Fatalf("Compute Handler Delete failed: %v", err)
	// }
	t.Log("Response: ", resp)

	defer conn.Close()
}
