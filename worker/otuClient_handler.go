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

// OtuClientHandler 是handler interface
type OtuClientHandler interface {
	CompareNe(clientQue map[string]map[string]db.CircleQueue, neList []string, taskId string)
	OtuClientHandle()
	Run()
}

type otuClientInfo struct {
	neId          string
	TerminalQueue db.CircleQueue
}

type otuClientHandle struct {
	compareLog     service.CompareLog
	neTopo         NeInfoStream
	streamData     map[string]otuClientInfo
	lock           sync.Mutex
	mgo            db.DbProxy
	sql            db.MysqlDbProxy
	handleDeadLock sync.Mutex
	neInit         NeBaseInfo
	streamOld      map[string]interface{}
	minInterval    time.Duration
	neList         []string
	taskId         string
	clientQueInfo  map[string]map[string]db.CircleQueue
	flag           bool
}

func (h *otuClientHandle) CompareNe(clientQue map[string]map[string]db.CircleQueue, neList []string, taskId string) {
	log.Infof("entering terminal Compare ****")
	log.Infof("neList is %v", neList)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = neList
	h.taskId = taskId
	h.clientQueInfo = make(map[string]map[string]db.CircleQueue)
	h.clientQueInfo = clientQue
	h.flag = true
	h.do()
}

// NewOtuClientHandler 初始化
func NewOtuClientHandler(mgo db.DbProxy, sql db.MysqlDbProxy) OtuClientHandler {
	bH := &otuClientHandle{
		mgo:           mgo,
		compareLog:    service.NewCompareLog(sql),
		clientQueInfo: make(map[string]map[string]db.CircleQueue),
	}
	return bH
}

// 从队列缓存数据 处理
func (h *otuClientHandle) OtuClientHandle() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = h.neInit.neList
}

func (h *otuClientHandle) Run() {
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
func (h *otuClientHandle) run() {
	go h.do()
}

//并发比对 上层算子可开多实例
//可用作同步方法
func (h *otuClientHandle) do() {
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

			h.compareLog.InsertLog([]string{neId}, h.taskId, int8(rcc.CompareType_ONCE),
				int8(rcc.StatusType_RUNNING), int8(rcc.ItemType_OTU_CLIENT_DIFF))

			// step1 处理TPC4 C口 Diff
			var tpClientRet []mod.TpClientDiffRet
			if _, ok := h.clientQueInfo[neId]; ok {
				for tpId, clientInfo := range h.clientQueInfo[neId] {
					ret, err := clientInfo.Pop()
					log.Infof("que is and tpId is %v %s", ret, tpId)
					if err != nil {
						log.Errorf("no data in client queue and ne is %s and err%v", neId, err)
						continue
					}
					streamData := ret.(NeOtuClient)
					adpaterId := streamData.AdapterId
					var ctrNeFroClient, adpNeFroClient mod.NeOtuClient
					ctrNeFroClient = mod.NeOtuClient{PortType: streamData.PortType, SignalRate: streamData.SignalRate,
						FecMod: streamData.FecMod}

					result, err := proxy.GetNeInfoFromAdapter(adpaterId, neId)
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

				//入库
				if !h.diffRetToMgo(neId, tpClientRet) {
					log.Errorf("into mgo failed")
					h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_OTU_CLIENT_DIFF),
						int8(rcc.StatusType_FAILED))
				} else {
					h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_OTU_CLIENT_DIFF),
						int8(rcc.StatusType_SUCCESS))
				}
			}
		}(v)
	}
	wg.Wait()
}

func (h *otuClientHandle) terminalPointDiff(neId string, a, b interface{}) []mod.DiffRet {
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

func (h *otuClientHandle) diffRetToMgo(neId string, tpClientDiffRet []mod.TpClientDiffRet) bool {
	newCheck := &mod.DiffMgo{
		NeId:          neId,
		TpClientDiffs: tpClientDiffRet,
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
