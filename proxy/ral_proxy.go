package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/imdario/mergo"
	"net/url"
	"rcc/mod"
	"trpc.group/trpc-go/trpc-go/log"
)

const (
	GET_PHY_TOPO_URL = "http://%s:8181/restconf/operational/network-topology:network-topology/topology/otn-phy-topology"
	GET_NE_FROM_CTL  = "http://%s:8181/restconf/operations/nms:get-phy-node-paged"
)

// CtrlGetPhyTopo 从controller获取物理拓扑
func CtrlGetPhyTopo() (*mod.PhyTopology, error) {
	host, err := getServiceAddr(conf.ctrlName)
	if err != nil {
		return nil, err
	}
	rep := mod.PhyTopoRep{}

	data, err := doGet(fmt.Sprintf(GET_PHY_TOPO_URL, host), conf.ctrlUserName, conf.ctrlPwd)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &rep)
	if err != nil {
		return nil, err
	}

	if len(rep.Topology) == 0 {
		return nil, errors.New("nil topology")
	}
	//log.Debugf("topo: %v", rep.Topology[0])
	return &rep.Topology[0], nil
}

// GetNeInfoFromMount 从adapter获取物理拓扑
func GetNeInfoFromMount() (*mod.AdaptPhyRep, error) {
	url := "http://106.55.81.224/restconf/operational/" +
		"opendaylight-inventory:nodes/node/adapter_10.255.1.164/yang-ext:mount/eml:nes"

	//host, err := getServiceAddr(conf.ctrlName)
	//if err != nil {
	//	return nil, err
	//}
	rep := new(mod.NesNode)

	//data, err := doGet(fmt.Sprintf(GET_PHY_TOPO_URL, host), conf.ctrlUserName, conf.ctrlPwd)
	data, err := doGet(url, "admin", "yijingdaizaiWANSHIXIAOXIN@2019")
	if err != nil {
		return nil, err
	}
	//fmt.Printf("ret is %v",string(data))
	err = json.Unmarshal(data, rep)
	if err != nil {
		return nil, err
	}

	if len(rep.Ada.Nes) == 0 {
		return nil, errors.New("nil topology")
	}
	return &rep.Ada, nil
}

// GetNeInfoFromController 是获取网元
func GetNeInfoFromController() (*mod.CtrlNe, error) {
	host, err := getServiceAddr(conf.ctrlName)
	if err != nil {
		return nil, err
	}
	//url := "http://106.55.81.224/restconf/operations/nms:get-phy-node-paged"

	req := mod.ComInput{
		Input: mod.CtrParam{
			TopologyRef: "otn-phy-topology",
			StartPos:    0,
			HowMany:     1000,
			SortInfos:   make([]interface{}, 0), //构造一个空的slice
		}}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	rep := new(mod.CtrlNesNode)

	repBody, err := doPost(fmt.Sprintf(GET_NE_FROM_CTL, host), conf.ctrlUserName, conf.ctrlPwd, reqBody)
	//repBody, err := doPost(url,"admin", "yijingdaizaiWANSHIXIAOXIN@2019",reqBody)

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(repBody, rep)

	if len(rep.CtrlTopology.Output) == 0 {
		return nil, errors.New("nil topology")
	}

	//获取到otn-phy-topo的cross connection  两个数据流merge
	otnPhy, err := CtrlGetPhyTopo()
	if err != nil {
		log.Errorf("get otn phy topology failed(%v)", err)
		return nil, nil
	}
	for _, v := range otnPhy.Node {
		for i, info := range rep.CtrlTopology.Output {
			if v.NodeId == info.NodeId {
				if err := mergo.Merge(&info.Physical, v.Physical); err != nil {
					log.Errorf("fatal is %s", err)
				}
				rep.CtrlTopology.Output[i].Physical.CrossConnections = info.Physical.CrossConnections
			}
		}
	}

	return &rep.CtrlTopology, nil
}

// GetNeInfoFromAdapter 是从Adapter获取
func GetNeInfoFromAdapter(aId, nodeId string) (*mod.AdapterNe, error) {
	log.Info("entering in adapter ne get *****")
	nodeId = url.QueryEscape(nodeId)
	log.Infof("adapter Id is %v", aId)
	aId = aId[8:len(aId)]
	adpUrl := "http://%s:8181/restconf/operational/eml:nes/ne/%s" //podIp 只能在集群内访问4层

	rep := new(mod.SouthNe)

	data, err := doGet(fmt.Sprintf(adpUrl, aId, nodeId), conf.ctrlUserName, "admin")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, rep)
	if err != nil {
		return nil, err
	}

	if len(rep.Nes) == 0 {
		return nil, errors.New("nil topology")
	}
	return &rep.Nes[0], nil
}
