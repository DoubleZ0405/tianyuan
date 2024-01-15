package mod

import (
	"strconv"
	"time"
)

const (
	ImplementState = "implement"
)

// Physical 是物理属性通用
type Physical struct {
	CreationTime     time.Time          `json:"creation-time"`
	ImplementState   string             `json:"implement-state"`
	Port             int                `json:"port"`
	LoginName        string             `json:"login-name"`
	LoginPasswd      string             `json:"login-passwd"`
	VendorType       string             `json:"vendor-type"`
	AdminState       string             `json:"admin-state"`
	Properties       Properties         `json:"properties"`
	NodeType         string             `json:"node-type"`
	OrderID          []string           `json:"order-id"`
	DomainName       string             `json:"domain-name"`
	VendorName       string             `json:"vendor-name"`
	System           System             `json:"system"`
	FriendlyName     string             `json:"friendly-name"`
	CrossConnections []CrossConnections `json:"cross-connections"`
	UsedGripGroupID  int                `json:"used-grip-group-id"`
	OcmGripGroups    []OcmGripGroups    `json:"OCM-grip-groups"`
	InternalLinks    []InternalLinks    `json:"internal-links"`
	RiskGroupName    string             `json:"risk-group-name"`
	OperationalState string             `json:"operational-state"`
	IP               string             `json:"ip"`
	AlarmState       string             `json:"alarm-state"`
	PlaneName        string             `json:"plane-name"`
	NodeRole         string             `json:"node-role"`
	AlignmentStatus  string             `json:"alignment-status"`
	Equipments       []Equipments       `json:"equipments"`
}

// PhySrc 是source terminal point
type PhySrc struct {
	SrcNode string `json:"source-node"`
	SrcTp   string `json:"source-tp"`
	NodeIp  string `json:"-"`
}

// PhyDst 是destination terminal point
type PhyDst struct {
	DstNode string `json:"dest-node"`
	DstTp   string `json:"dest-tp"`
	NodeIp  string `json:"-"`
}

// PhyNode 是node
type PhyNode struct {
	NodeId   string   `json:"node-id"`
	Physical Physical `json:"otn-phy-topology:physical"`
}

// PhyLink 是Link
type PhyLink struct {
	LinkId   string   `json:"link-id"`
	Physical Physical `json:"otn-phy-topology:physical"`
	Src      PhySrc   `json:"source"`
	Dst      PhyDst   `json:"destination"`
}

// PhyTopology 是物理topology
type PhyTopology struct {
	Node []PhyNode `json:"node"`
}

// AdaptPhyRep 是南向网元
type AdaptPhyRep struct {
	Nes []AdapterNe `json:"ne"`
}

// AdapterNe  是南向网元
type AdapterNe struct {
	NodeId           string             `json:"node-id"`
	Physical         Physical           `json:"physical"`
	TerminationPoint []TerminationPoint `json:"termination-point"`
}

// NesNode 是南向
type NesNode struct {
	Ada AdaptPhyRep `json:"nes"`
}

// SouthNe 是南向处理
type SouthNe struct {
	Nes []AdapterNe `json:"ne"`
}

// PhyTopoRep 是reply
type PhyTopoRep struct {
	Topology []PhyTopology `json:"topology"`
}

// InstanceDetails 是zk InstanceDetails
type InstanceDetails struct {
	ModuleName string `json:"moduleName"`
	MyIp       string `json:"myIp"`
	User       string `json:"user"`
	Pwd        string `json:"passwd"`
	Namespace  string `json:"namespace"`
}

const (
	TT_BIDIRECT = "bidirection"
	TT_SRC      = "source"
	TT_SINK     = "sink"
)

// LinkTask 是link threshold task
type LinkTask struct {
	LinkId   string  `json:"link-id"`
	LinkName string  `json:"link-name"`
	Positive float32 `json:"positive-threshold"`
	Reverse  float32 `json:"reverse-threshold"`
	TaskType string  `json:"task-type"`
}

// TaskData 是threshold tasks
type TaskData struct {
	Tasks []LinkTask `json:"threshold-schedule:links"`
}

// TaskInput 是task input
type TaskInput struct {
	Data     TaskData `json:"task-data"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Interval int      `json:"repeat-interval"`
	Oper     string   `json:"operation"`
	Start    int64    `json:"start"`
	End      int64    `json:"end"`
}

// TaskReq 是task request
type TaskReq struct {
	Input TaskInput `json:"input"`
}

// NewTaskReq 生成Task请求
func NewTaskReq(tasks []LinkTask) *TaskReq {
	now := time.Now().Unix() * 1000
	return &TaskReq{
		Input: TaskInput{
			Data: TaskData{
				Tasks: tasks,
			},
			Name:     "Modify-OtsLink-Attenuation-Schedule-" + strconv.FormatInt(now, 10),
			Status:   "running",
			Interval: -1,
			Oper:     "threshold-schedule:threshold-modify",
			Start:    now,
			End:      now + 24*60*60*1000,
		},
	}
}

// ComInput 是ctrlinput
type ComInput struct {
	Input CtrParam `json:"input"`
}

// CtrParam 是ctl参数
type CtrParam struct {
	TopologyRef string        `json:"topology-ref"`
	StartPos    int           `json:"start-pos"`
	HowMany     int           `json:"how-many"`
	SortInfos   []interface{} `json:"sort-infos"`
}

// ComOutput 是返回
type ComOutput struct {
	Output []Node `json:"output"`
}

// Node 是网元节点
type Node struct {
	NodeId   string   `json:"node-id"`
	Physical Physical `json:"physical"`
}

// CtrlNe 是控制器网元
type CtrlNe struct {
	Output []CtrlNode `json:"node"`
}

// CtrlNode 是node
type CtrlNode struct {
	NodeId           string             `json:"node-id"`
	Physical         Physical           `json:"physical"`
	TerminationPoint []TerminationPoint `json:"termination-point"`
}

// CtrlNesNode 是output
type CtrlNesNode struct {
	CtrlTopology CtrlNe `json:"output"`
}
