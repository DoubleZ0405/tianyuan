package worker

import (
	"context"
	rcc "git.code.oa.com/trpcprotocol/tianyuan/rcc_server_rcc"
	"github.com/r3labs/diff"
	"rcc/db"
	"rcc/mod"
	"rcc/proxy"
	"rcc/service"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/log"
)

// OtuLineHandler 是handler interface
type OtuLineHandler interface {
	CompareNe(lineQue map[string]map[string]db.CircleQueue, neList []string, taskId string)
	OtuLineHandle()
	Run()
}

type otuLineInfo struct {
	neId          string
	TerminalQueue db.CircleQueue
}

type otuLineHandle struct {
	compareLog     service.CompareLog
	neTopo         NeInfoStream
	streamData     map[string]otuLineInfo
	lock           sync.Mutex
	mgo            db.DbProxy
	sql            db.MysqlDbProxy
	handleDeadLock sync.Mutex
	neInit         NeBaseInfo
	streamOld      map[string]interface{}
	minInterval    time.Duration
	neList         []string
	taskId         string
	lineQueInfo    map[string]map[string]db.CircleQueue
	flag           bool
}

func (h *otuLineHandle) CompareNe(lineQue map[string]map[string]db.CircleQueue, neList []string, taskId string) {
	log.Infof("entering terminal Compare ****")
	log.Infof("neList is %v", neList)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = neList
	h.taskId = taskId
	h.lineQueInfo = make(map[string]map[string]db.CircleQueue)
	h.lineQueInfo = lineQue
	h.flag = true
	h.do()
}

// NewOtuLineHandler 初始化Line
func NewOtuLineHandler(mgo db.DbProxy, sql db.MysqlDbProxy) OtuLineHandler {
	bH := &otuLineHandle{
		mgo:         mgo,
		compareLog:  service.NewCompareLog(sql),
		lineQueInfo: make(map[string]map[string]db.CircleQueue),
	}
	return bH
}

// 从队列缓存数据 处理
func (h *otuLineHandle) OtuLineHandle() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = h.neInit.neList
}

func (h *otuLineHandle) Run() {
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
func (h *otuLineHandle) run() {
	go h.do()
}

//并发比对 上层算子可开多实例
//可用作同步方法
func (h *otuLineHandle) do() {
	log.Infof("entering terminal-point otuLine diff do ****")
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

			h.compareLog.InsertLog([]string{neId}, h.taskId, int8(rcc.CompareType_ONCE),
				int8(rcc.StatusType_RUNNING), int8(rcc.ItemType_OTU_LINE_DIFF))

			// step1 处理TPC4 L口 Diff
			var tpLineRet []mod.TpLineDiffRet
			if _, ok := h.lineQueInfo[neId]; ok {
				for tpId, lineInfo := range h.lineQueInfo[neId] {
					retLine, err := lineInfo.Pop()
					log.Infof("que is and tpId is %v  and %s", retLine, tpId)
					if err != nil {
						log.Errorf("no data in line queue and ne is %s and err%v", neId, err)
						continue
					}
					streamDataLine := retLine.(NeOtuLine)
					adpaterId := streamDataLine.AdapterId
					var ctlNeFromLine, adpNeFromLine mod.NeOtuLine
					ctlNeFromLine = mod.NeOtuLine{PortType: streamDataLine.PortType,
						SignalRate:        streamDataLine.SignalRate,
						TargetOutputPower: streamDataLine.TargetOutputPower}

					resultLine, err := proxy.GetNeInfoFromAdapter(adpaterId, neId)
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
							log.Infof("from adapter ********** is %#v", adpNeFromLine)
							if ctlNeFromLine == (mod.NeOtuLine{}) && adpNeFromLine == (mod.NeOtuLine{}) {
								continue
							}
							lineDiff := h.terminalPointDiff(neId, ctlNeFromLine, adpNeFromLine)
							if len(lineDiff) == 0 {
								continue
							}
							tpLineRet = append(tpLineRet, mod.TpLineDiffRet{TpId: tpId, LineDiffs: lineDiff})
						}
					}
				}

				//入库
				if !h.diffRetToMgo(neId, tpLineRet) {
					log.Errorf("into mgo failed")
					h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_OTU_LINE_DIFF),
						int8(rcc.StatusType_FAILED))
				} else {
					h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_OTU_LINE_DIFF),
						int8(rcc.StatusType_SUCCESS))
				}
			}
		}(v)
	}
	wg.Wait()
}

func (h *otuLineHandle) terminalPointDiff(neId string, a, b interface{}) []mod.DiffRet {
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

func (h *otuLineHandle) diffRetToMgo(neId string, tpLineDiffRet []mod.TpLineDiffRet) bool {
	newCheck := &mod.DiffMgo{
		NeId:        neId,
		TpLineDiffs: tpLineDiffRet,
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
