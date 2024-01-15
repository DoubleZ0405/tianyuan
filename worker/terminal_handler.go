package worker

import (
	"context"
	"github.com/r3labs/diff"
	"rcc/db"
	"rcc/mod"
	"rcc/proxy"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/log"
)

// TerminalHandler 是handler interface
type TerminalHandler interface {
	CompareNe(clientQue, lineQue map[string]map[string]db.CircleQueue,
		tpIdListQue map[string]db.CircleQueue, neList []string)
	TerminalHandler()
	Run()
}

type terminalInfo struct {
	neId          string
	TerminalQueue db.CircleQueue
}

type terminalHandle struct {
	neTopo                     NeInfoStream
	streamData                 map[string]terminalInfo
	lock                       sync.Mutex
	mgo                        db.DbProxy
	handleDeadLock             sync.Mutex
	neInit                     NeBaseInfo
	streamOld                  map[string]interface{}
	minInterval                time.Duration
	neList                     []string
	clientQueInfo, lineQueInfo map[string]map[string]db.CircleQueue
	flag                       bool
	adpaterId                  string
	tpIdList                   map[string]db.CircleQueue
}

func (h *terminalHandle) CompareNe(clientQue, lineQue map[string]map[string]db.CircleQueue,
	tpIdListQue map[string]db.CircleQueue, neList []string) {
	log.Infof("entering terminal Compare ****")
	log.Infof("neList is %v", neList)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = neList
	h.clientQueInfo = make(map[string]map[string]db.CircleQueue)
	h.clientQueInfo = clientQue
	h.lineQueInfo = make(map[string]map[string]db.CircleQueue)
	h.lineQueInfo = lineQue
	h.tpIdList = make(map[string]db.CircleQueue)
	h.tpIdList = tpIdListQue
	h.flag = true
	h.do()
}

// NewTerminalHandler 是TpHandle
func NewTerminalHandler(mgo db.DbProxy) TerminalHandler {
	bH := &terminalHandle{
		mgo:           mgo,
		lineQueInfo:   make(map[string]map[string]db.CircleQueue),
		clientQueInfo: make(map[string]map[string]db.CircleQueue),
		tpIdList:      make(map[string]db.CircleQueue),
	}
	return bH
}

// TerminalHandler 从队列缓存数据 处理
func (h *terminalHandle) TerminalHandler() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = h.neInit.neList
}

func (h *terminalHandle) Run() {
	log.Infof("handler start")
	ticker := time.NewTicker(h.minInterval * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go h.do()
		}
	}
}

// routine执行方法
func (h *terminalHandle) run() {
	go h.do()
}

func (h *terminalHandle) do() {
	log.Infof("entering terminal-point diff do ****")
	var wg sync.WaitGroup
	var once sync.Once
	for _, v := range h.neList {
		wg.Add(1)
		log.Infof("this node is %s", v)
		go func(neId string) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("Recovered in f", err)
					once.Do(func() {
						log.Errorf("panic found in call handlers %v", err)
					})
				}
				wg.Done()
			}()
			// step1 处理TPC4 C口 Diff
			var tpClientRet []mod.TpClientDiffRet
			for tpId, clientInfo := range h.clientQueInfo[neId] {
				ret, err := clientInfo.Pop()
				log.Infof("que is and tpId is %v %s", ret, tpId)
				if err != nil {
					log.Errorf("no data in client queue and ne is %s and err%v", neId, err)
					continue
				}
				streamData := ret.(NeOtuClient)
				h.adpaterId = streamData.AdapterId
				var ctrNeFroClient, adpNeFroClient mod.NeOtuClient
				ctrNeFroClient = mod.NeOtuClient{PortType: streamData.PortType, SignalRate: streamData.SignalRate,
					FecMod: streamData.FecMod}

				result, err := proxy.GetNeInfoFromAdapter(h.adpaterId, neId)
				if err != nil {
					log.Errorf("get ne from adapter failed(%v)", err)
					return
				}

				for _, v := range result.TerminationPoint {
					if tpId == v.TpID {
						adpNeFroClient = mod.NeOtuClient{PortType: v.Physical.PortType,
							SignalRate: v.Physical.OtuClient.SignalRate,
							FecMod:     v.Physical.OtuClient.Client.FecMode}
						if ctrNeFroClient == (mod.NeOtuClient{}) && adpNeFroClient == (mod.NeOtuClient{}) {
							continue
						}
						clientDiff := h.terminalPointDiff(neId, ctrNeFroClient, adpNeFroClient)
						if len(clientDiff) == 0 {
							continue
						}
						tpClientRet = append(tpClientRet, mod.TpClientDiffRet{
							TpId: tpId, ClientDiffs: clientDiff})
					}
				}
			}

			//step2 tpc4 L口 diff 与OPC4处理
			var nodeType string
			var tpLineRet []mod.TpLineDiffRet
			var tpIdListFromAdapter []string
			for tpId, lineInfo := range h.lineQueInfo[neId] {
				retLine, err := lineInfo.Pop()
				log.Infof("que is and tpId is %v  and %s", retLine, tpId)
				if err != nil {
					log.Errorf("no data in line queue and ne is %s and err%v", neId, err)
					continue
				}
				streamDataLine := retLine.(NeOtuLine)
				h.adpaterId = streamDataLine.AdapterId
				var ctlNeFromLine, adpNeFromLine mod.NeOtuLine
				ctlNeFromLine = mod.NeOtuLine{PortType: streamDataLine.PortType, SignalRate: streamDataLine.SignalRate,
					TargetOutputPower: streamDataLine.TargetOutputPower}

				resultLine, err := proxy.GetNeInfoFromAdapter(h.adpaterId, neId)
				log.Infof("adapter is %s", resultLine)
				if err != nil {
					log.Errorf("get ne from adapter failed(%v)", err)
					return
				}
				for _, lineInfo := range resultLine.TerminationPoint {
					if tpId == lineInfo.TpID {
						adpNeFromLine = mod.NeOtuLine{PortType: lineInfo.Physical.PortType,
							SignalRate:        lineInfo.Physical.OtuLine.SignalRate,
							TargetOutputPower: lineInfo.Physical.OtuLine.TargetOutputPowerLower}

						if ctlNeFromLine == (mod.NeOtuLine{}) && adpNeFromLine == (mod.NeOtuLine{}) {
							continue
						}
						lineDiff := h.terminalPointDiff(neId, ctlNeFromLine, adpNeFromLine)
						if len(lineDiff) == 0 {
							continue
						}
						tpLineRet = append(tpLineRet, mod.TpLineDiffRet{TpId: tpId, LineDiffs: lineDiff})
					}
					if resultLine.Physical.NodeType == "OPC4" { //为了下面做Opc的diff
						tpIdListFromAdapter = append(tpIdListFromAdapter, lineInfo.TpID)
					}
				}
			}
			log.Infof("adapter tpList is %s", tpIdListFromAdapter)

			//step4 opc4 diff
			var tpOpc4Ret []mod.TpOpcDiffRet
			retFromOpc4Que, err := h.tpIdList[neId].Pop()
			if err != nil {
				log.Errorf("no data in opc queue and ne is %s and err%v", neId, err)
				nodeType = "TPC4"
			} else {
				opc4Diff := h.terminalPointDiff(neId, retFromOpc4Que, tpIdListFromAdapter)
				if len(opc4Diff) != 0 {
					log.Infof("opc diff %v", opc4Diff)
					tpOpc4Ret = append(tpOpc4Ret, mod.TpOpcDiffRet{
						OpcDiffRet: opc4Diff,
					})
				}
				nodeType = "OPC4"
			}
			var diffRet []mod.DiffRet
			//入库
			if !h.diffRetToMgo(neId, nodeType, tpClientRet, tpLineRet, diffRet) {
				log.Errorf("into mgo failed")
			}
		}(v)
	}
	wg.Wait()
}

func (h *terminalHandle) terminalPointDiff(neId string, a, b interface{}) []mod.DiffRet {
	log.Info("entering mgo diff in******")
	changelog, err := diff.Diff(a, b)
	if err != nil {
		log.Errorf("diff err %#v", err)
	}

	log.Infof("diff ret is %v", changelog)
	var retDiff []mod.DiffRet
	for _, v := range changelog {
		retDiff = append(retDiff, mod.DiffRet{Type: v.Type, Path: v.Path, Controller: v.From, Adapter: v.To})
	}

	return retDiff
}

func (h *terminalHandle) diffRetToMgo(neId, nodeType string, tpClientDiffRet []mod.TpClientDiffRet,
	tpLineDiffRet []mod.TpLineDiffRet, tpOpcDiffRet []mod.DiffRet) bool {
	var newCheck *mod.DiffMgo
	if nodeType == "OPC4" {
		newCheck = &mod.DiffMgo{
			NeId:      neId,
			Opc4Diffs: tpOpcDiffRet,
		}
	} else {
		newCheck = &mod.DiffMgo{
			NeId:          neId,
			TpClientDiffs: tpClientDiffRet,
			TpLineDiffs:   tpLineDiffRet,
		}
	}

	filter := mod.DiffMgo{NeId: neId}

	if yes, err := h.mgo.Exist(context.Background(), db.INSPECTION, filter); err != nil || !yes {
		if _, e := h.mgo.InsertOne(context.Background(), db.INSPECTION, newCheck); e != nil {
			log.Errorf("add ne(%s) base diff failed(%+v)", neId, err)
			return false
		}
	} else if e := h.mgo.FindOneAndUpdate(context.Background(), db.INSPECTION, filter, newCheck); e != nil {
		log.Errorf("update ne(%s) base diff failed(%+v)", neId, err)
		return false
	}

	return true
}

//func (h *terminalHandle) opc4DiffRetToMgo(neId string,opc4DiffRet []mod.DiffRet) bool {
//	newCheck := &mod.DiffMgo{
//		NeId: neId,
//		Opc4Diffs: opc4DiffRet,
//	}
//	filter :=&mod.DiffMgo{NeId: neId}
//
//	if yes, err := h.mgo.Exist(context.Background(), db.INSPECTION, filter); err != nil || !yes {
//		if _, e := h.mgo.InsertOne(context.Background(), db.INSPECTION, newCheck); e != nil {
//			log.Errorf("add ne(%s) base diff failed(%+v)", neId, err)
//			return false
//		}
//	} else if e := h.mgo.FindOneAndUpdate(context.Background(), db.INSPECTION, filter, newCheck); e != nil {
//		log.Errorf("update ne(%s) base diff failed(%+v)", neId, err)
//		return false
//	}
//
//	return true
//}
