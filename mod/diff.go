package mod

import (
	"fmt"
	"trpc.group/trpc-go/trpc-go/log"
)

// NeBase 是基础
type NeBase struct {
	Port            int
	LoginName       string
	LoginPasswd     string
	VendorType      string
	UsedGripGroupId int
}

// NeProperty 是分类对比
type NeProperty struct {
	WssMinDestSourceVoa float64
	WssMinSourceDestVoa float64
	WssMaxDestSourceVoa float64
	WssMaxSourceDestVoa float64
	CurrentSoftware     string
	HostName            string
}

// NeAmplifier 是NE 放大器
type NeAmplifier struct {
	TargetGain        float64
	TargetGainTilt    float64
	TargetAttenuation float64
}

// NeAps 是aps组件
type NeAps struct {
	Revertive   bool
	ForceToPort string
	HoldOffTime int
	Name        string
	ActivePath  string
}

// NeOtuLine tpc4设备 L口
type NeOtuLine struct {
	PortType          string
	SignalRate        string
	TargetOutputPower float64
}

// NeOtuClient NeTerminalPoint tpc4设备 C口
type NeOtuClient struct {
	PortType   string
	SignalRate string
	FecMod     string
}

// Changelog stores a list of changed items
type Changelog []Change

// Change stores information about a changed item
type Change struct {
	Type string      `json:"type"`
	Path []string    `json:"path"`
	From interface{} `json:"from"`
	To   interface{} `json:"to"`
}

// DiffRet 是diff返回
type DiffRet struct {
	Type       string      // The type of change detected; can be one of create, update or delete
	Path       []string    // The path of the detected change;
	Controller interface{} // The original value that was present in the "from" structure
	Adapter    interface{} // The new value that was detected as a change in the "to" structure
}

// DiffMgo 是mgo的底层
type DiffMgo struct {
	Id            string            `json:"id,omitempty"`
	NeId          string            `json:"node-id,omitempty"`
	BaseDiffs     []DiffRet         `json:"base-diffs,omitempty"`
	OcmDiffs      []OcmDiffRet      `json:"ocm-diffs,omitempty"`
	PropertyDiffs []DiffRet         `json:"property-diffs,omitempty"`
	CrossConDiffs []CrossConRet     `json:"cross-con-diffs,omitempty"`
	TpClientDiffs []TpClientDiffRet `json:"tp-otu-client-diffs,omitempty"`
	TpLineDiffs   []TpLineDiffRet   `json:"tp-otu-line-diffs,omitempty"`
	Opc4Diffs     []DiffRet         `json:"tp-opc4-diffs,omitempty"`
}

// CrossConRet 是底层
type CrossConRet struct {
	CrossConnectionID string    `json:"cross-connection-id"`
	AmplifierDiffs    []DiffRet `json:"amplifier-diffs,omitempty"`
	ApsDiffs          []DiffRet `json:"aps-diffs,omitempty"`
}

// TpClientDiffRet 是tp口
type TpClientDiffRet struct {
	TpId        string    `json:"tp-id"`
	ClientDiffs []DiffRet `json:"client-diffs,omitempty"`
}

// TpLineDiffRet 是LINE口diff
type TpLineDiffRet struct {
	TpId      string    `json:"tp-id"`
	LineDiffs []DiffRet `json:"line-diffs,omitempty"`
}

// TpOpcDiffRet 是OpcDiff返回
type TpOpcDiffRet struct {
	OpcDiffRet []DiffRet `json:"opc-diff-ret,omitempty"`
}

// OcmDiffRet 是返回
type OcmDiffRet struct {
	Index        string    `json:"index"`
	ChannelDiffs []DiffRet `json:"channel-diffs,omitempty"`
}

func diffToPb(diff []DiffRet) []*DiffRet {
	var cValue, aValue string
	var rs []*DiffRet

	for _, v := range diff {
		if v.Controller != nil {
			switch v.Controller.(type) {
			case int8:
				tmp := v.Controller.(int8)
				cValue = fmt.Sprintf("%d", tmp)
			case int32:
				tmp := v.Controller.(int32)
				cValue = fmt.Sprintf("%d", tmp)
			case int64:
				tmp := v.Controller.(int64)
				cValue = fmt.Sprintf("%d", tmp)
			case float32:
				tmp := v.Controller.(float32)
				cValue = fmt.Sprintf("%f", tmp)
			case float64:
				tmp := v.Controller.(float64)
				cValue = fmt.Sprintf("%f", tmp)
			default:
				cValue = v.Controller.(string)
			}
		}

		if v.Adapter != nil {
			switch v.Adapter.(type) {
			case int8:
				tmp := v.Adapter.(int8)
				aValue = fmt.Sprintf("%d", tmp)
			case int32:
				tmp := v.Adapter.(int32)
				aValue = fmt.Sprintf("%d", tmp)
			case int64:
				tmp := v.Adapter.(int64)
				aValue = fmt.Sprintf("%d", tmp)
			case float32:
				tmp := v.Adapter.(float32)
				aValue = fmt.Sprintf("%f", tmp)
			case float64:
				tmp := v.Adapter.(float64)
				aValue = fmt.Sprintf("%f", tmp)
			default:
				aValue = v.Adapter.(string)
			}
		}

		rs = append(rs, &DiffRet{
			Type:       v.Type,
			Path:       v.Path,
			Controller: cValue,
			Adapter:    aValue})
	}

	return rs
}

func crossDiffToPb(diff []CrossConRet) []*CrossDiff {
	var rs []*rcc.CrossDiff

	for _, v := range diff {
		rs = append(rs, &rcc.CrossDiff{
			CrossId:       v.CrossConnectionID,
			AmplifierDiff: diffToPb(v.AmplifierDiffs),
			ApsDiff:       diffToPb(v.ApsDiffs)})
	}

	return rs
}

func tpClientDiffToPb(diff []TpClientDiffRet) []*rcc.TpClientDiff {
	var rs []*rcc.TpClientDiff
	for _, v := range diff {
		rs = append(rs, &rcc.TpClientDiff{
			TpId:       v.TpId,
			ClientDiff: diffToPb(v.ClientDiffs)})
	}

	return rs
}
func tpLineDiffToPb(diff []TpLineDiffRet) []*rcc.TpLineDiff {
	var rs []*rcc.TpLineDiff

	for _, v := range diff {
		rs = append(rs, &rcc.TpLineDiff{TpId: v.TpId, LineDiff: diffToPb(v.LineDiffs)})
	}

	return rs
}

func ocmDiffToPb(diff []OcmDiffRet) []*rcc.OcmDiff {
	var rs []*rcc.OcmDiff

	for _, v := range diff {
		rs = append(rs, &rcc.OcmDiff{Index: v.Index, ChannelDiff: diffToPb(v.ChannelDiffs)})
	}

	return rs
}

// DiffMongoToPb 是入库
func DiffMongoToPb(diff []*DiffMgo) []*rcc.NeDifferences {
	var rs []*rcc.NeDifferences

	for _, v := range diff {
		rs = append(rs, &rcc.NeDifferences{
			NeId:         v.NeId,
			BaseDiff:     diffToPb(v.BaseDiffs),
			OcmDiff:      ocmDiffToPb(v.OcmDiffs),
			PropertyDiff: diffToPb(v.PropertyDiffs),
			CrossDiff:    crossDiffToPb(v.CrossConDiffs),
			TpOpcDiff:    diffToPb(v.Opc4Diffs),
			TpClientDiff: tpClientDiffToPb(v.TpClientDiffs),
			TpLineDiff:   tpLineDiffToPb(v.TpLineDiffs)})
	}
	log.Infof("diff ret is %#v", rs)
	return rs
}
