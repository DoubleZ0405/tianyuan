package worker

import (
	"encoding/json"
	"rcc/common/websocket"
	"rcc/db"
	"rcc/mod"
	"rcc/proxy"
	"runtime"
	"strconv"
	"sync"
	"time"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
)

// NeInfoStream 是工厂
type NeInfoStream interface {
	getQueueObj(flag string) map[string]db.CircleQueue
	getNeList(ver uint64) ([]string, uint64)
	CompareNe(conf config.Config, taskId string, neList []string, mgo db.MdbProxy, sql db.MysqlDbProxy)
	Run()
	webRun(taskId string, neList []string)
}

// NeBaseInfo 是Ne
type NeBaseInfo struct {
	lock                                                                               sync.RWMutex
	NeBase                                                                             map[string]mod.NeBase
	ver                                                                                uint64
	syncInterval                                                                       time.Duration
	taskId                                                                             string
	sql                                                                                db.MysqlDbProxy
	Ws                                                                                 *websocket.WsServer
	neList                                                                             []string
	baseQue, propertyQue, tpIdListQue                                                  map[string]db.CircleQueue
	apiFlag                                                                            bool
	nodeList                                                                           []string
	baseHandler                                                                        BaseHandler
	propertyHandler                                                                    PropertyHandler
	tpOpcHandler                                                                       TpOpcHandler
	crossConHandler                                                                    CrossHandler
	otuClientHandler                                                                   OtuClientHandler
	otuLineHandler                                                                     OtuLineHandler
	ocmHandler                                                                         OcmHandler
	nodes                                                                              map[string]mod.CtrlNode
	amplifierQue, ocmQue, wssChanQue, apsQue, crossPropertyQue, tpClientQue, tpLineQue map[string]map[string]db.CircleQueue
}

func (t *NeBaseInfo) webRun(taskId string, neList []string) {
	log.Infof("Ne BaseInfo diff start, sync interval: %d", t.syncInterval)
	t.apiFlag = true
	log.Infof("init is %#v", neList)
	t.nodeList = neList
	log.Infof("task is running %s ", taskId)
	t.taskId = taskId
	go t.doCrl()
}

// CompareNe 是对比服务
func (t *NeBaseInfo) CompareNe(conf config.Config, taskId string, neList []string,
	mgo db.MdbProxy, sql db.MysqlDbProxy) {
	//stream := NewBaseCommon(conf, mgo, sql)
	//stream.webRun(taskId, neList)
	t.webRun(taskId, neList)
}

// NewBaseCommon 是初始化
func NewBaseCommon(conf config.Config, mgo db.MdbProxy, sql db.MysqlDbProxy, ws *websocket.WsServer) NeInfoStream {
	interval := conf.GetInt("custom.topo.interval", 3600)
	return &NeBaseInfo{
		syncInterval:     time.Duration(interval),
		ver:              0,
		sql:              sql,
		Ws:               ws,
		baseHandler:      NewBaseHandler(mgo, sql),
		propertyHandler:  NewPropertyHandler(mgo, sql),
		crossConHandler:  NewCrossHandler(mgo, sql),
		tpOpcHandler:     NewTpOpcHandler(mgo, sql),
		otuClientHandler: NewOtuClientHandler(mgo, sql),
		otuLineHandler:   NewOtuLineHandler(mgo, sql),
		ocmHandler:       NewOcmHandler(mgo, sql),
	}
}

// Run 是同步
func (t *NeBaseInfo) Run() {
	log.Infof("Ne BaseInfo diff start, sync interval: %d", t.syncInterval)
	go t.run()
}

func (t *NeBaseInfo) run() {
	ticker := time.NewTicker(t.syncInterval * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.doCrl()
		}
	}
}

// NeBase 是网元对比
type NeBase struct {
	port            int
	loginName       string
	loginPasswd     string
	vendorType      string
	adapterId       string
	usedGripGroupId int
}

// NeProperty 是property
type NeProperty struct {
	WssMinDestSourceVoa float64
	WssMinSourceDestVoa float64
	WssMaxDestSourceVoa float64
	WssMaxSourceDestVoa float64
	CurrentSoftware     string
	HostName            string
	AdapterId           string
}

// NeAmplifier 是网元放大器
type NeAmplifier struct {
	TargetGain        float64
	TargetGainTilt    float64
	TargetAttenuation float64
	AdapterId         string
}

// NeAps 是网元APS
type NeAps struct {
	Revertive   bool
	ForceToPort string
	HoldOffTime int
	Name        string
	ActivePath  string
	AdapterId   string
}

// NeCrossProperty 是Cross
type NeCrossProperty struct {
	MaxIngressVoa float64
}

// NeOtuLine tpc4设备 L口
type NeOtuLine struct {
	TpId              string
	PortType          string
	SignalRate        string
	TargetOutputPower float64
	AdapterId         string
}

// NeOtuClient NeTerminalPoint tpc4设备 C口
type NeOtuClient struct {
	TpId       string
	PortType   string
	SignalRate string
	FecMod     string
	AdapterId  string
}

// NeOcmGripChannel 是ocm
type NeOcmGripChannel struct {
	Index          int
	LowerFrequency int
	UpperFrequency int
	AdapterId      string
}

func (t *NeBaseInfo) doCrl() {
	log.Infof("entering doCrl****")
	log.Infof("flag is %v", t.apiFlag)

	topology, err := proxy.GetNeInfoFromController()
	if err != nil {
		log.Errorf("get phy topology failed(%v)", err)
		return
	}

	//step0 缓存一个当前管理网neId拓扑
	var newNeList []string
	t.nodes = make(map[string]mod.CtrlNode)
	apiNodes := make(map[string]mod.CtrlNode)
	for _, v := range topology.Output {
		if v.Physical.ImplementState != mod.ImplementState {
			log.Debugf("that ne node (%s) is not ready,pass ", v.NodeId)
			continue
		}
		if t.apiFlag && in(v.NodeId, t.nodeList) {
			log.Infof("entering %s", t.apiFlag)
			apiNodes[v.NodeId] = v
		}

		t.nodes[v.NodeId] = v
		newNeList = append(newNeList, v.NodeId)
	}

	if !t.apiFlag {
		curTopo, _ := t.getNeList(0)
		if !compareTopo(newNeList, curTopo) {
			t.setNeList(newNeList)
		}
	} else {
		t.nodes = apiNodes
		log.Infof("t.nodeList is %#v", t.nodeList)
		t.setNeList(t.nodeList)
	}

	//log.Infof("nodes is %#v",t.nodes)

	//并发处理多喝init后继续执行主线程下游的diff处理
	handlers := []func() error{t.BaseInit, t.PropertyInit, t.CrossConnectionInit, t.TerminalPointInit, t.OcmGroupsInit}
	var wg sync.WaitGroup
	var once sync.Once
	for _, f := range handlers {
		wg.Add(1)
		go func(handler func() error) {
			defer func() {
				if e := recover(); e != nil {
					buf := make([]byte, 1024)
					buf = buf[:runtime.Stack(buf, false)]
					once.Do(func() {
						err = errs.New(errs.RetServerSystemErr, "panic found in call handlers")
					})
				}
				wg.Done()
			}()
			if e := handler(); e != nil {
				once.Do(func() {
					err = e
				})
			}
		}(f)
	}
	wg.Wait()

	if t.apiFlag {
		log.Info("entering handler ******")
		t.baseHandler.CompareNe(t.baseQue, t.nodeList, t.taskId)
		t.propertyHandler.CompareNe(t.propertyQue, t.nodeList, t.taskId)
		t.crossConHandler.CompareNe(t.amplifierQue, t.apsQue, t.nodeList, t.taskId)
		t.tpOpcHandler.CompareNe(t.tpIdListQue, t.nodeList, t.taskId)
		t.otuClientHandler.CompareNe(t.tpClientQue, t.nodeList, t.taskId)
		t.otuLineHandler.CompareNe(t.tpLineQue, t.nodeList, t.taskId)
		t.ocmHandler.CompareNe(t.ocmQue, t.nodeList, t.taskId)

		//对比完成后通过websocket通知前端，内容为本次task id
		msg, _ := json.Marshal(&websocket.TaskNotification{Body: t.taskId})
		err = t.Ws.Handle(msg)
		if err != nil {
			log.Errorf("task %s failed to send websocket message to website", t.taskId)
		}
	}
}

// BaseInit 初始化
func (t *NeBaseInfo) BaseInit() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	log.Info("base init ************")
	t.baseQue = make(map[string]db.CircleQueue)
	for _, v := range t.nodes {
		var aId string
		for _, item := range v.Physical.Properties.Property {
			if item.Name == "adapter-id" {
				aId = item.Value
			}
		}
		baseQueue := db.NewCircleQueue(0)
		baseQueue.Push(NeBase{port: v.Physical.Port, loginName: v.Physical.LoginName,
			loginPasswd: v.Physical.LoginPasswd, vendorType: v.Physical.VendorType, adapterId: aId,
			usedGripGroupId: v.Physical.UsedGripGroupID})
		t.baseQue[v.NodeId] = baseQueue
	}

	return nil
}

// PropertyInit 是Init
func (t *NeBaseInfo) PropertyInit() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	log.Info("property init ************")
	t.propertyQue = make(map[string]db.CircleQueue)
	for _, v := range t.nodes {
		var aId, cSoft, hName string
		var wssMind2s, wssMins2d, wssMaxd2s, wssMaxs2d float64
		for _, item := range v.Physical.Properties.Property {
			switch item.Name {
			case "wss_min-dest-to-source-voa":
				wssMind2s, _ = strconv.ParseFloat(item.Value, 64)
			case "wss_min-source-to-dest-voa":
				wssMins2d, _ = strconv.ParseFloat(item.Value, 64)
			case "wss_max-source-to-dest-voa":
				wssMaxs2d, _ = strconv.ParseFloat(item.Value, 64)
			case "wss_max-dest-to-source-voa":
				wssMaxd2s, _ = strconv.ParseFloat(item.Value, 64)
			case "current-software":
				cSoft = item.Value
			case "hostName":
				hName = item.Value
			case "adapter-id":
				aId = item.Value
			default:
				break
			}
		}
		propertyQue := db.NewCircleQueue(0)
		propertyQue.Push(NeProperty{WssMinDestSourceVoa: wssMind2s, WssMinSourceDestVoa: wssMins2d,
			WssMaxDestSourceVoa: wssMaxd2s, WssMaxSourceDestVoa: wssMaxs2d, CurrentSoftware: cSoft,
			HostName: hName, AdapterId: aId})
		t.propertyQue[v.NodeId] = propertyQue
	}

	return nil
}

// OcmGroupsInit 是OcmInit
func (t *NeBaseInfo) OcmGroupsInit() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.ocmQue = make(map[string]map[string]db.CircleQueue)

	for _, node := range t.nodes {
		var aId string
		for _, item := range node.Physical.Properties.Property {
			if item.Name == "adapter-id" {
				aId = item.Value
			}
		}

		nodeType := node.Physical.NodeType
		usedGroupId := node.Physical.UsedGripGroupID
		crlOcmGrips := make(map[string]db.CircleQueue)

		if nodeType == "OPC4" {
			for _, ocm := range node.Physical.OcmGripGroups {
				//只获取OPC当前正在用的波道系列
				if ocm.Index == usedGroupId {
					for _, ch := range ocm.Channels {
						channelQue := db.NewCircleQueue(0)
						channelQue.Push(NeOcmGripChannel{
							Index:          ch.Index,
							LowerFrequency: ch.LowerFrequency,
							UpperFrequency: ch.UpperFrequency,
							AdapterId:      aId})
						crlOcmGrips[strconv.Itoa(ch.Index)] = channelQue
					}
				}
			}
		}

		if len(crlOcmGrips) != 0 {
			t.ocmQue[node.NodeId] = crlOcmGrips
		}
	}

	return nil
}

// CrossConnectionInit 是CrossCon的初始化
func (t *NeBaseInfo) CrossConnectionInit() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.amplifierQue = make(map[string]map[string]db.CircleQueue)
	t.apsQue = make(map[string]map[string]db.CircleQueue)
	t.crossPropertyQue = make(map[string]map[string]db.CircleQueue)

	for _, v := range t.nodes {
		var aId string
		for _, item := range v.Physical.Properties.Property {
			if item.Name == "adapter-id" {
				aId = item.Value
			}
		}
		crossAmplifier := make(map[string]db.CircleQueue)
		crossAps := make(map[string]db.CircleQueue)
		for _, info := range v.Physical.CrossConnections {
			amplifierQue := db.NewCircleQueue(0)
			apsQue := db.NewCircleQueue(0)
			neAmplifier := NeAmplifier{TargetGain: info.Amplifier.TargetGain,
				TargetGainTilt: info.Amplifier.TargetGainTilt, TargetAttenuation: info.Amplifier.TargetAttenuation,
				AdapterId: aId}
			neAps := NeAps{Name: info.Aps.Name, ForceToPort: info.Aps.ForceToPort, HoldOffTime: info.Aps.HoldOffTime,
				Revertive: info.Aps.Revertive, ActivePath: info.Aps.ActivePath, AdapterId: aId}

			if neAmplifier != (NeAmplifier{}) || neAps != (NeAps{}) {
				amplifierQue.Push(neAmplifier)
				apsQue.Push(neAps)
				crossAmplifier[info.CrossConnectionID] = amplifierQue
				crossAps[info.CrossConnectionID] = apsQue
			}
		}
		t.amplifierQue[v.NodeId] = crossAmplifier
		t.apsQue[v.NodeId] = crossAps
	}

	return nil
}

// TerminalPointInit 是Tp初始化
func (t *NeBaseInfo) TerminalPointInit() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.tpClientQue = make(map[string]map[string]db.CircleQueue)
	t.tpLineQue = make(map[string]map[string]db.CircleQueue)
	t.tpIdListQue = make(map[string]db.CircleQueue)

	for _, v := range t.nodes {
		var aId string
		for _, item := range v.Physical.Properties.Property {
			if item.Name == "adapter-id" {
				aId = item.Value
			}
		}
		tpClient := make(map[string]db.CircleQueue)
		tpLine := make(map[string]db.CircleQueue)
		nodeType := v.Physical.NodeType
		tpIdQue := db.NewCircleQueue(0)
		var tpList []string
		for _, tpInfo := range v.TerminationPoint {
			otuLineQue := db.NewCircleQueue(0)
			otuClientQue := db.NewCircleQueue(0)
			if nodeType == "OPC4" {
				log.Infof("entering opc init node id is (%s) and nodeType is (%s)", v.NodeId, nodeType)
				tpList = append(tpList, tpInfo.TpID)
			} else if nodeType == "TPC4" {
				if tpInfo.Physical.PortType == "OTU-Line" && tpInfo.Physical.OtuLine != (mod.OtuLine{}) {
					otuLineQue.Push(NeOtuLine{TpId: tpInfo.TpID,
						PortType:          tpInfo.Physical.PortType,
						SignalRate:        tpInfo.Physical.OtuLine.SignalRate,
						TargetOutputPower: tpInfo.Physical.OtuLine.TargetOutputPowerLower,
						AdapterId:         aId})
					tpLine[tpInfo.TpID] = otuLineQue
				} else if tpInfo.Physical.PortType == "OTU-Client" && tpInfo.Physical.OtuClient != (mod.OtuClient{}) {
					otuClientQue.Push(NeOtuClient{TpId: tpInfo.TpID,
						PortType:   tpInfo.Physical.PortType,
						FecMod:     tpInfo.Physical.OtuClient.Client.FecMode,
						SignalRate: tpInfo.Physical.OtuClient.SignalRate,
						AdapterId:  aId})
					tpClient[tpInfo.TpID] = otuClientQue
				}
			}
		}
		if len(tpList) != 0 && nodeType == "OPC4" {
			log.Infof("en %s", tpList)
			tpIdQue.Push(tpList)
			t.tpIdListQue[v.NodeId+","+aId] = tpIdQue
		} else if nodeType == "TPC4" {
			t.tpClientQue[v.NodeId] = tpClient
			t.tpLineQue[v.NodeId] = tpLine
		}
	}

	return nil
}

func (t *NeBaseInfo) setNeBase(neBaseInfo map[string]mod.NeBase) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.NeBase = neBaseInfo
}

// 返回一个新的ne表实例
func (t *NeBaseInfo) getNeList(ver uint64) ([]string, uint64) {
	var neList []string
	t.lock.RLock()
	defer t.lock.RUnlock()
	if ver == t.ver {
		return nil, 0
	}
	for _, v := range t.neList {
		neList = append(neList, v)
	}
	return neList, t.ver
}

// 更新topolink
func (t *NeBaseInfo) setNeList(neList []string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.neList = neList
	t.ver++
}

// 比较ne topo，一致返回true否则返回false
func compareTopo(ta, tb []string) bool {
	if len(ta) != len(tb) {
		return false
	}
	if (ta == nil) != (tb == nil) {
		return false
	}
	for k, v := range ta {
		if v != ta[k] {
			return false
		}
	}
	return true
}

func (t *NeBaseInfo) getQueueObj(flag string) map[string]db.CircleQueue {
	t.lock.RLock()
	defer t.lock.RUnlock()
	switch flag {
	case "base":
		return t.baseQue
	case "property":
		return t.propertyQue
	default:
		break
	}
	return nil
}

func in(target string, str_array []string) bool {
	for _, element := range str_array {
		if target == element {
			return true
		}
	}
	return false
}
