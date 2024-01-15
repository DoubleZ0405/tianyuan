package mod

import "time"

// OtnPhysicalNodes 是Node
type OtnPhysicalNodes struct {
	Node []OtnPhysicalNode `json:"node"`
}

// OtnPhysicalNode 是物理网元
type OtnPhysicalNode struct {
	NodeID                 string                 `json:"node-id"`
	OtnPhyTopologyPhysical OtnPhyTopologyPhysical `json:"otn-phy-topology:physical"`
	TerminationPoint       []TerminationPoint     `json:"termination-point"`
}

// Property 是Property
type Property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Properties 是Pro
type Properties struct {
	Property []Property `json:"property"`
}

// Ntp 是Ntp
type Ntp struct {
	IP string `json:"ip"`
}

// Syslog 对比Log
type Syslog struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Severity string `json:"severity"`
	Selector string `json:"selector"`
}

// Radius 是Radius
type Radius struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	SecretKey  string `json:"secret-key"`
	RetryTimes int    `json:"retry-times"`
	Timeout    int    `json:"timeout"`
}

// Telemetry 采集
type Telemetry struct {
	IP                string   `json:"ip"`
	SensorGroupName   string   `json:"sensor-group-name"`
	Port              int      `json:"port"`
	SensorGroupPath   []string `json:"sensor-group-path"`
	SampleInterval    int      `json:"sample-interval"`
	HeartbeatInterval int      `json:"heartbeat-interval"`
	SuppressRedundant bool     `json:"suppress-redundant"`
}

// System 配置
type System struct {
	Ntp        []Ntp       `json:"ntp"`
	Syslog     []Syslog    `json:"syslog"`
	Radius     []Radius    `json:"radius"`
	Telemetry  []Telemetry `json:"telemetry"`
	Properties Properties  `json:"properties"`
}

// DestinationTp 是Tp
type DestinationTp struct {
	TpRef string `json:"tp-ref"`
}

// Aps 是Aps
type Aps struct {
	Revertive         bool       `json:"revertive"`
	ForceToPort       string     `json:"force-to-port"`
	HoldOffTime       int        `json:"hold-off-time"`
	WaitToRestoreTime int        `json:"wait-to-restore-time"`
	ApsMode           string     `json:"aps-mode"`
	Name              string     `json:"name"`
	Properties        Properties `json:"properties"`
	ActivePath        string     `json:"active-path"`
}

// SourceTp 是
type SourceTp struct {
	TpRef string `json:"tp-ref"`
}

// Amplifier A
type Amplifier struct {
	TargetGain         float64 `json:"target-gain"`
	AmpMode            string  `json:"amp-mode"`
	TargetAttenuation  float64 `json:"target-attenuation"`
	GainRange          string  `json:"gain-range"`
	TargetGainTilt     float64 `json:"target-gain-tilt"`
	Enable             bool    `json:"enable"`
	TargetOutputPower  float64 `json:"target-output-power"`
	AutoPowerReduction bool    `json:"auto-power-reduction"`
}

// PsdValue 频率
type PsdValue struct {
	LowerFrequency  int     `json:"lower-frequency"`
	UpperFrequency  int     `json:"upper-frequency"`
	SliceType       string  `json:"slice-type"`
	SourceToDestVoa float64 `json:"source-to-dest-voa"`
	DestToSourceVoa int     `json:"dest-to-source-voa"`
}

// PsdDistribution 路径
type PsdDistribution struct {
	PsdValue []PsdValue `json:"psd-value"`
}

// WssChannel Wss
type WssChannel struct {
	PowerControlMode string          `json:"power-control-mode"`
	UpperFrequency   int             `json:"upper-frequency"`
	LowerFrequency   int             `json:"lower-frequency"`
	PsdDistribution  PsdDistribution `json:"psd-distribution"`
}

// CrossConnections CrossCon
type CrossConnections struct {
	CrossConnectionID string          `json:"cross-connection-id"`
	ImplementState    string          `json:"implement-state"`
	DestinationTp     []DestinationTp `json:"destination-tp"`
	NodeRef           string          `json:"node-ref"`
	Fixed             bool            `json:"fixed"`
	Aps               Aps             `json:"aps,omitempty"`
	Description       string          `json:"description"`
	OperationalState  string          `json:"operational-state"`
	Direction         string          `json:"direction"`
	SourceTp          []SourceTp      `json:"source-tp"`
	AdminState        string          `json:"admin-state"`
	Properties        Properties      `json:"properties,omitempty"`
	Amplifier         Amplifier       `json:"amplifier,omitempty"`
	WssChannel        WssChannel      `json:"wss-channel,omitempty"`
}

// Channels channel
type Channels struct {
	Index          int `json:"index"`
	LowerFrequency int `json:"lower-frequency"`
	UpperFrequency int `json:"upper-frequency"`
}

// OcmGripGroups Ocm
type OcmGripGroups struct {
	Index    int        `json:"index"`
	Channels []Channels `json:"channels"`
}

// InternalLinks InLink
type InternalLinks struct {
	LinkName       string     `json:"link-name"`
	LinkRef        string     `json:"link-ref"`
	Dsttp          string     `json:"dstTp"`
	ImplementState string     `json:"implement-state"`
	Srctp          string     `json:"srcTp"`
	LinkType       string     `json:"link-type"`
	Properties     Properties `json:"properties,omitempty"`
	AdminState     string     `json:"admin-state,omitempty"`
}

// Equipments 设备
type Equipments struct {
	EquipmentID             string     `json:"equipment-id"`
	EquipType               string     `json:"equip-type"`
	CreationTime            time.Time  `json:"creation-time,omitempty"`
	ImplementState          string     `json:"implement-state"`
	EquipTypeConfiged       string     `json:"equip-type-configed,omitempty"`
	RiskGroupName           string     `json:"risk-group-name,omitempty"`
	OperationalState        string     `json:"operational-state"`
	AdminState              string     `json:"admin-state"`
	OrderID                 []string   `json:"order-id,omitempty"`
	NodeRef                 string     `json:"node-ref"`
	AlarmState              string     `json:"alarm-state"`
	PlaneName               string     `json:"plane-name,omitempty"`
	EquipTypeVendorSpecific string     `json:"equip-type-vendor-specific,omitempty"`
	AlignmentStatus         string     `json:"alignment-status"`
	FriendlyName            string     `json:"friendly-name"`
	Empty                   bool       `json:"empty,omitempty"`
	Properties              Properties `json:"properties,omitempty"`
	EquipTypeInstalled      string     `json:"equip-type-installed,omitempty"`
	SerialNo                string     `json:"serial-no,omitempty"`
	Removeable              bool       `json:"removeable,omitempty"`
	HardwareVersion         string     `json:"hardware-version,omitempty"`
	SoftwareVersion         string     `json:"software-version,omitempty"`
	VendorName              string     `json:"vendor-name,omitempty"`
}

// OtnPhyTopologyPhysical 物理
type OtnPhyTopologyPhysical struct {
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

// Osc O
type Osc struct {
	AutoAttenuationMode bool `json:"auto-attenuation-mode"`
}

// Wdm wdm
type Wdm struct {
	Osc Osc `json:"osc"`
}

// TpPhysical 是Tp的
type TpPhysical struct {
	OtuLine          OtuLine    `json:"otu-line,omitempty"`
	NodeRef          string     `json:"node-ref"`
	PortType         string     `json:"port-type"`
	ConnectionStatus string     `json:"connection-status"`
	FriendlyName     string     `json:"friendly-name"`
	ImplementState   string     `json:"implement-state"`
	AlignmentStatus  string     `json:"alignment-status"`
	Direction        string     `json:"direction"`
	AlarmState       string     `json:"alarm-state"`
	EquipmentRef     string     `json:"equipment-ref"`
	AdminState       string     `json:"admin-state"`
	Properties       Properties `json:"properties"`
	OperationalState string     `json:"operational-state"`
	OtuClient        OtuClient  `json:"otu-client,omitempty"`
}

// OtuClient 是OTUClient
type OtuClient struct {
	SignalRate string `json:"signal-rate"`
	Client     Client `json:"client"`
}

// Client 是Client
type Client struct {
	EthComplianceCode string `json:"ethComplianceCode"`
	FecMode           string `json:"fec-mode"`
}

// OtuLine 是Line口
type OtuLine struct {
	SignalRate              string  `json:"signal-rate"`
	TargetOutputPower       float64 `json:"target-output-power"`
	TargetOutputPowerHigher float64 `json:"target-output-power-higher"`
	TargetOutputPowerLower  float64 `json:"target-output-power-lower"`
	CentralFrequency        int     `json:"central-frequency"`
}

// TerminationPoint 是tp
type TerminationPoint struct {
	TpID     string     `json:"tp-id"`
	Physical TpPhysical `json:"physical,omitempty"`
}
