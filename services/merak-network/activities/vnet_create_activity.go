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

package activities

import (
	"context"
	"encoding/json"
	pb "github.com/futurewei-cloud/merak/api/proto/v1/merak"
	"github.com/futurewei-cloud/merak/services/merak-network/database"
	"github.com/futurewei-cloud/merak/services/merak-network/entities"
	"github.com/futurewei-cloud/merak/services/merak-network/http"
	"github.com/futurewei-cloud/merak/services/merak-network/utils"
	"log"
	"sync"
)

var (
	returnNetworkMessage = pb.ReturnNetworkMessage{
		ReturnCode:       pb.ReturnCode_OK,
		ReturnMessage:    "returnNetworkMessage Finished",
		Vpcs:             nil,
		SecurityGroupIds: nil,
	}
)

func doVPC(vpc *pb.InternalVpcInfo, projectId string) (vpcId string, err error) {
	log.Println("doVPC")
	vpcBody := entities.VpcStruct{Network: entities.VpcBody{
		AdminStateUp:        true,
		RevisionNumber:      0,
		Cidr:                vpc.VpcCidr,
		ByDefault:           true,
		Description:         "vpc",
		DnsDomain:           "domain",
		IsDefault:           true,
		Mtu:                 1400,
		Name:                "YM_sample_vpc",
		PortSecurityEnabled: true,
		ProjectId:           vpc.ProjectId,
	}}
	returnMessage, returnErr := http.RequestCall("http://"+utils.ALCORURL+":30001/project/"+projectId+"/vpcs", "POST", vpcBody, nil)
	if returnErr != nil {
		log.Printf("returnErr %s", returnErr)
		return "", returnErr
	}
	log.Printf("returnMessage %s", returnMessage)
	var returnJson entities.VpcReturn
	json.Unmarshal([]byte(returnMessage), &returnJson)
	database.Set(utils.VPC+returnJson.Network.ID, returnJson.Network)
	log.Printf("returnJson : %+v", returnJson)
	log.Println("doVPC done")
	return returnJson.Network.ID, nil
}
func doSubnet(subnet *pb.InternalSubnetInfo, vpcId string, projectId string) (subnetId string, err error) {
	log.Println("doSubnet")
	subnetBody := entities.SubnetStruct{Subnet: entities.SubnetBody{
		Cider:     subnet.SubnetCidr,
		Name:      "YM_sample_subnet",
		IpVersion: 4,
		NetworkId: vpcId,
	}}
	returnMessage, returnErr := http.RequestCall("http://"+utils.ALCORURL+":30002/project/"+projectId+"/subnets", "POST", subnetBody, nil)
	if returnErr != nil {
		log.Printf("returnErr %s", returnErr)
		return "", returnErr
	}
	log.Printf("returnMessage %s", returnMessage)
	var returnJson entities.SubnetReturn
	json.Unmarshal([]byte(returnMessage), &returnJson)
	database.Set(utils.SUBNET+returnJson.Subnet.ID, returnJson.Subnet)
	log.Printf("doSubnet returnJson : %+v", returnJson)
	log.Println("doSubnet done")
	return returnJson.Subnet.ID, nil
}
func doRouter(vpcId string, projectId string) (routerId string, err error) {
	log.Println("doRouter")
	routerBody := entities.RouterStruct{Router: entities.RouterBody{
		AdminStateUp: true,
		Description:  "router description",
		Distributed:  true,
		ExternalGatewayInfo: entities.RouterExternalGatewayInfo{
			EnableSnat:       true,
			ExternalFixedIps: nil,
			NetworkId:        vpcId,
		},
		FlavorId:       "",
		GatewayPorts:   nil,
		Ha:             true,
		Name:           "YM_simple_router",
		ProjectId:      "123456789",
		RevisionNumber: 0,
		Status:         "BUILD",
		TenantId:       "123456789",
	}}
	returnMessage, returnErr := http.RequestCall("http://"+utils.ALCORURL+":30003/project/"+projectId+"/routers", "POST", routerBody, nil)
	if returnErr != nil {
		log.Printf("returnErr %s", returnErr)
		return "", returnErr
	}
	log.Printf("returnMessage %s", returnMessage)
	var returnJson entities.RouterReturn
	json.Unmarshal([]byte(returnMessage), &returnJson)
	database.Set(utils.Router+returnJson.Router.ID, returnJson.Router)
	log.Printf("returnJson : %+v", returnJson)
	log.Println("doRouter done")
	return returnJson.Router.ID, nil
}
func doAttachRouter(routerId string, subnetId string, projectId string) error {
	log.Println("doAttachRouter")
	attachRouterBody := entities.AttachRouterStruct{SubnetId: subnetId}
	url := "http://" + utils.ALCORURL + ":30003/project/" + projectId + "/routers/" + routerId + "/add_router_interface"
	returnMessage, returnErr := http.RequestCall(url, "PUT", attachRouterBody, nil)
	if returnErr != nil {
		log.Printf("returnErr %s", returnErr)
		return returnErr
	}
	log.Printf("returnMessage %s", returnMessage)
	var returnJson entities.AttachRouterReturn
	json.Unmarshal([]byte(returnMessage), &returnJson)
	log.Printf("returnJson : %+v", returnJson)
	log.Println("doAttachRouter done")
	return nil
}
func doSg(sg *pb.InternalSecurityGroupInfo, sgID string, projectId string) (string, error) {
	log.Println("doSg")
	sgBody := entities.SgStruct{Sg: entities.SgBody{
		Id:                 sgID,
		Description:        "sg Description",
		Name:               "YM_sample_sg",
		ProjectId:          sg.ProjectId,
		SecurityGroupRules: nil,
		TenantId:           sg.TenantId,
	}}
	returnMessage, returnErr := http.RequestCall("http://"+utils.ALCORURL+":30008/project/"+projectId+"/security-groups", "POST", sgBody, nil)
	if returnErr != nil {
		log.Printf("returnErr %s", returnErr)
		return "", returnErr
	}
	log.Printf("returnMessage %s", returnMessage)
	var returnJson entities.SgReturn
	json.Unmarshal([]byte(returnMessage), &returnJson)
	database.Set(utils.SECURITYGROUP+returnJson.SecurityGroup.ID, returnJson.SecurityGroup)
	log.Printf("returnJson : %+v", returnJson)
	log.Println("doSg done")
	return returnJson.SecurityGroup.ID, nil
}

func VnetCreate(ctx context.Context, netConfigId string, network *pb.InternalNetworkInfo, wg *sync.WaitGroup, returnMessage chan *pb.ReturnNetworkMessage, projectId string) (*pb.ReturnNetworkMessage, error) {
	log.Println("VnetCreate")
	//defer wg.Done()
	// TODO may want to separate bellow sections to different function, and use `go` and `wg` to improve overall speed
	// TODO when do concurrent, need to keep in mind on how to control the number of concurrency
	// Doing vpc and subnet

	var vpcId string
	var vpcIds []string
	subnetCiderIdMap := make(map[string]string)
	for _, vpc := range network.Vpcs {
		vpcId, err := doVPC(vpc, projectId)
		if err != nil {
			return nil, err
		}
		vpcIds = append(vpcIds, vpcId)
		var returnInfo []*pb.InternalVpcInfo

		var subnetInfo []*pb.InternalSubnetInfo
		for _, subnet := range vpc.Subnets {
			subnetId, err := doSubnet(subnet, vpcId, projectId)
			if err != nil {
				return nil, err
			}
			subnetCiderIdMap[subnet.SubnetCidr] = subnetId
			log.Printf("subnetCiderIdMap %s", subnetCiderIdMap)
			currentSubnet := pb.InternalSubnetInfo{
				SubnetId:   subnetId,
				SubnetCidr: subnet.SubnetCidr,
				SubnetGw:   subnet.SubnetGw,
				NumberVms:  subnet.NumberVms,
			}
			subnetInfo = append(subnetInfo, &currentSubnet)
		}
		currentVPC := pb.InternalVpcInfo{
			VpcId:     vpcId,
			TenantId:  vpc.TenantId,
			ProjectId: vpc.ProjectId,
			Subnets:   subnetInfo,
		}
		returnNetworkMessage.Vpcs = append(returnNetworkMessage.Vpcs, &currentVPC)
		log.Printf("VnetCreate End %s", returnInfo)
	}

	//doing security group
	for _, sg := range network.SecurityGroups {
		sgId := utils.GenUUID()
		_, err := doSg(sg, sgId, projectId)
		if err != nil {
			return nil, err
		}
		returnNetworkMessage.SecurityGroupIds = append(returnNetworkMessage.SecurityGroupIds, sgId)
		log.Printf("sgId: %s", sgId)
	}

	//doing router: create and attach subnet
	for _, router := range network.Routers {
		routerId, err := doRouter(vpcId, projectId)
		if err != nil {
			return nil, err
		}
		for _, subnet := range router.Subnets {
			err := doAttachRouter(routerId, subnetCiderIdMap[subnet], projectId)
			if err != nil {
				return nil, err
			}
		}
	}
	database.Set(utils.NETCONFIG+netConfigId, &returnNetworkMessage)
	log.Printf("&returnNetworkMessage %s", &returnNetworkMessage)
	return &returnNetworkMessage, nil
}