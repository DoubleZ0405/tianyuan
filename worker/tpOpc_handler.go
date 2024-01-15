package worker

import (
	"context"
	rcc "git.code.oa.com/trpcprotocol/tianyuan/rcc_server_rcc"
	"github.com/r3labs/diff"
	"rcc/db"
	"rcc/mod"
	"rcc/proxy"
	"rcc/service"
	"strings"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/log"
)

// TpOpcHandler handler interface
type TpOpcHandler interface {
	CompareNe(tpIdListQue map[string]db.CircleQueue, neList []string, taskId string)
	TpOpcHandler()
	Run()
}

type tpOpcInfo struct {
	neId          string
	TerminalQueue db.CircleQueue
}

type tpOpcHandle struct {
	compareLog     service.CompareLog
	neTopo         NeInfoStream
	streamData     map[string]tpOpcInfo
	lock           sync.Mutex
	mgo            db.DbProxy
	sql            db.MysqlDbProxy
	handleDeadLock sync.Mutex
	neInit         NeBaseInfo
	streamOld      map[string]interface{}
	minInterval    time.Duration
	neList         []string
	taskId         string
	flag           bool
	tpIdList       map[string]db.CircleQueue
}

func (h *tpOpcHandle) CompareNe(tpIdListQue map[string]db.CircleQueue, neList []string, taskId string) {
	log.Infof("entering terminal Compare ****")
	log.Infof("neList is %v", neList)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = neList
	h.taskId = taskId
	h.tpIdList = make(map[string]db.CircleQueue)
	h.tpIdList = tpIdListQue
	h.flag = true
	h.do()
}

// NewTpOpcHandler 初始化
func NewTpOpcHandler(mgo db.DbProxy, sql db.MysqlDbProxy) TpOpcHandler {
	bH := &tpOpcHandle{
		mgo:        mgo,
		compareLog: service.NewCompareLog(sql),
		tpIdList:   make(map[string]db.CircleQueue),
	}
	return bH
}

// 从队列缓存数据 处理
func (h *tpOpcHandle) TpOpcHandler() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = h.neInit.neList
}

func (h *tpOpcHandle) Run() {
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
func (h *tpOpcHandle) run() {
	go h.do()
}

//并发比对 上层算子可开多实例
//可用作同步方法
func (h *tpOpcHandle) do() {
	log.Infof("entering terminal-point opc4 diff do ****")
	for _, v := range h.neList {
		neId := v

		h.compareLog.InsertLog([]string{neId}, h.taskId, int8(rcc.CompareType_ONCE),
			int8(rcc.StatusType_RUNNING), int8(rcc.ItemType_TP_OPC_DIFF))
		//step1 opc4 diff
		var nodeId, adapterId string
		var tpIdListFromAdapter []string

		if len(h.tpIdList) != 0 {
			for k := range h.tpIdList {
				splitK := strings.Split(k, ",")
				log.Infof("split is %s", splitK)
				nodeId = splitK[0]
				adapterId = splitK[1]
				log.Infof("nodeid is %s", nodeId)
				log.Infof("adapterid is %s", adapterId)
				if neId == nodeId {
					log.Infof("entering adapter diff ***")
					retFromOpc4Que, err := h.tpIdList[neId+","+adapterId].Pop()
					if err != nil {
						log.Errorf("no data in opc queue and ne is %s and err%v", neId, err)
						return
					}
					resultLine, err := proxy.GetNeInfoFromAdapter(adapterId, neId)
					log.Infof("adapter is %s", resultLine)
					if err != nil {
						log.Errorf("get ne from adapter failed(%v)", err)
						return
					}
					for _, lineInfo := range resultLine.TerminationPoint {
						if resultLine.Physical.NodeType == "OPC4" { //为了下面做Opc的diff
							tpIdListFromAdapter = append(tpIdListFromAdapter, lineInfo.TpID)
						}
					}
					log.Infof("adapter tpList is %s", tpIdListFromAdapter)

					opc4Diff := h.terminalPointDiff(neId, retFromOpc4Que, tpIdListFromAdapter)
					if len(opc4Diff) != 0 {
						log.Infof("opc diff %v", opc4Diff)
						//入库
						if !h.opc4DiffRetToMgo(neId, opc4Diff) {
							log.Errorf("into mgo failed")
							h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_TP_OPC_DIFF),
								int8(rcc.StatusType_FAILED))
						} else {
							h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_TP_OPC_DIFF),
								int8(rcc.StatusType_SUCCESS))
						}
					}
				}
			}
		}
	}
}

func (h *tpOpcHandle) terminalPointDiff(neId string, a, b interface{}) []mod.DiffRet {
	log.Info("entering mgo diff in******")
	var retDiff []mod.DiffRet

	changelog, err := diff.Diff(a, b)
	if err != nil {
		log.Errorf("diff err %#v", err)
	}

	log.Infof("diff ret is %v", changelog)
	if len(changelog) <= 0 {
		//找到当前NE的历史对比结果，如果改正过后 要在mongoDb恢复到正常态数据
		retDiff = append(retDiff, mod.DiffRet{
			Type:       "normal",
			Path:       []string{"normal"},
			Controller: "normal",
			Adapter:    "normal",
		})
		return retDiff
	}
	for _, v := range changelog {
		retDiff = append(retDiff, mod.DiffRet{Type: v.Type, Path: v.Path, Controller: v.From, Adapter: v.To})
	}

	return retDiff
}

func (h *tpOpcHandle) opc4DiffRetToMgo(neId string, opc4DiffRet []mod.DiffRet) bool {
	if len(opc4DiffRet) <= 0 {
		//找到当前NE的历史对比结果，如果改正过后 要在mongoDb恢复到正常态数据
		opc4DiffRet = append(opc4DiffRet, mod.DiffRet{
			Type:       "normal",
			Path:       []string{"normal"},
			Controller: "normal",
			Adapter:    "normal",
		})
	}

	newCheck := &mod.DiffMgo{
		NeId:      neId,
		Opc4Diffs: opc4DiffRet,
	}
	filter := &mod.DiffMgo{NeId: neId}

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
