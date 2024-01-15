package worker

import (
	"context"
	rcc "git.code.oa.com/trpcprotocol/tianyuan/rcc_server_rcc"
	"github.com/imdario/mergo"
	"github.com/r3labs/diff"
	"rcc/db"
	"rcc/mod"
	"rcc/proxy"
	"rcc/service"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/log"
)

// CrossHandler 是handler interface
type CrossHandler interface {
	CompareNe(amplifierQue, apsQue map[string]map[string]db.CircleQueue, neList []string, taskId string)
	CrossHandle()
	Run()
}

type crossInfo struct {
	neId       string
	CrossQueue db.CircleQueue
}

type crossHandle struct {
	compareLog                   service.CompareLog
	neTopo                       NeInfoStream
	streamData                   map[string]crossInfo
	lock                         sync.Mutex
	mgo                          db.DbProxy
	sql                          db.MysqlDbProxy
	handleDeadLock               sync.Mutex
	neInit                       NeBaseInfo
	streamOld                    map[string]interface{}
	minInterval                  time.Duration
	neList                       []string
	taskId                       string
	apsQueInfo, amplifierQueInfo map[string]map[string]db.CircleQueue
	flag                         bool
	adpaterId                    string
}

func (h *crossHandle) CompareNe(amplifierQue, apsQue map[string]map[string]db.CircleQueue,
	neList []string, taskId string) {
	log.Infof("entering cross-con Compare ****")
	log.Infof("neList is %v", neList)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = neList
	h.taskId = taskId
	h.apsQueInfo = make(map[string]map[string]db.CircleQueue)
	h.apsQueInfo = apsQue
	h.amplifierQueInfo = make(map[string]map[string]db.CircleQueue)
	h.amplifierQueInfo = amplifierQue
	h.flag = true
	h.do()
}

// NewCrossHandler 是初始化
func NewCrossHandler(mgo db.DbProxy, sql db.MysqlDbProxy) CrossHandler {
	bH := &crossHandle{
		compareLog:       service.NewCompareLog(sql),
		mgo:              mgo,
		amplifierQueInfo: make(map[string]map[string]db.CircleQueue),
		apsQueInfo:       make(map[string]map[string]db.CircleQueue),
	}
	return bH
}

// 从队列缓存数据 处理
func (h *crossHandle) CrossHandle() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = h.neInit.neList
}

func (h *crossHandle) Run() {
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
func (h *crossHandle) run() {
	go h.do()
}

func (h *crossHandle) do() {
	log.Infof("entering cross diff do ****")
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
			// step1 处理amplifier Diff
			h.compareLog.InsertLog([]string{neId}, h.taskId, int8(rcc.CompareType_ONCE),
				int8(rcc.StatusType_RUNNING), int8(rcc.ItemType_CROSS_DIFF))

			var crossDiffAmplifierRet []mod.CrossConRet
			for crossConnId, amplifierInfo := range h.amplifierQueInfo[neId] {
				log.Infof("that cross(%s) and neId is (%s)", crossConnId, neId)
				ret, err := amplifierInfo.Pop()
				if err != nil {
					log.Errorf("no data in amplifier queue and ne is %s and err%v", neId, err)
					continue
				}
				streamData := ret.(NeAmplifier)
				h.adpaterId = streamData.AdapterId
				var ctrNeFroAmplifier, adpNeFroAmplifier mod.NeAmplifier
				ctrNeFroAmplifier = mod.NeAmplifier{TargetGain: streamData.TargetGain,
					TargetGainTilt:    streamData.TargetGainTilt,
					TargetAttenuation: streamData.TargetAttenuation}

				result, err := proxy.GetNeInfoFromAdapter(h.adpaterId, neId)
				if err != nil {
					h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_CROSS_DIFF),
						int8(rcc.StatusType_FAILED))
					log.Errorf("get ne from adapter failed(%v)", err)
					return
				}

				for _, v := range result.Physical.CrossConnections {
					if crossConnId == v.CrossConnectionID {
						adpNeFroAmplifier = mod.NeAmplifier{TargetGain: v.Amplifier.TargetGain,
							TargetGainTilt:    v.Amplifier.TargetGainTilt,
							TargetAttenuation: v.Amplifier.TargetAttenuation}
						if ctrNeFroAmplifier == (mod.NeAmplifier{}) && adpNeFroAmplifier == (mod.NeAmplifier{}) {
							continue
						}
						amplifierDiff := h.crossDiff(neId, ctrNeFroAmplifier, adpNeFroAmplifier)
						crossDiffAmplifierRet = append(crossDiffAmplifierRet, mod.CrossConRet{
							CrossConnectionID: crossConnId, AmplifierDiffs: amplifierDiff})
					}
				}
			}

			//step2 aps diff
			var crossDiffRetAps []mod.CrossConRet
			for crossConId, apsInfo := range h.apsQueInfo[neId] {
				retAps, err := apsInfo.Pop()
				if err != nil {
					log.Errorf("no data in amplifier queue and ne is %s and err%v", neId, err)
					continue
				}
				streamDataAps := retAps.(NeAps)
				h.adpaterId = streamDataAps.AdapterId

				var ctlNeFromAps, adpNeFromAps mod.NeAps
				ctlNeFromAps = mod.NeAps{HoldOffTime: streamDataAps.HoldOffTime, ForceToPort: streamDataAps.ForceToPort,
					Revertive: streamDataAps.Revertive, Name: streamDataAps.Name, ActivePath: streamDataAps.ActivePath}

				resultAps, err := proxy.GetNeInfoFromAdapter(h.adpaterId, neId)
				if err != nil {
					h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_CROSS_DIFF),
						int8(rcc.StatusType_FAILED))
					log.Errorf("get ne from adapter failed(%v)", err)
					return
				}
				for _, apsInfo := range resultAps.Physical.CrossConnections {
					if crossConId == apsInfo.CrossConnectionID {
						adpNeFromAps = mod.NeAps{HoldOffTime: apsInfo.Aps.HoldOffTime, ForceToPort: apsInfo.Aps.ForceToPort,
							Revertive: apsInfo.Aps.Revertive, Name: apsInfo.Aps.Name, ActivePath: apsInfo.Aps.ActivePath}

						if ctlNeFromAps == (mod.NeAps{}) && adpNeFromAps == (mod.NeAps{}) {
							continue
						}
						apsDiff := h.crossDiff(neId, ctlNeFromAps, adpNeFromAps)
						crossDiffRetAps = append(crossDiffRetAps, mod.CrossConRet{CrossConnectionID: crossConId, ApsDiffs: apsDiff})
					}
				}
			}

			// merge amplifier与aps 的diff结果数据
			for _, dest := range crossDiffRetAps {
				for index, source := range crossDiffAmplifierRet {
					if dest.CrossConnectionID == source.CrossConnectionID {
						if err := mergo.Merge(&source, dest, mergo.WithOverride, mergo.WithOverrideEmptySlice); err != nil {
							log.Fatal(err)
						}
						crossDiffAmplifierRet[index].ApsDiffs = source.ApsDiffs
					}
				}
			}

			//入库
			if !h.diffRetToMgo(neId, crossDiffAmplifierRet) {
				log.Errorf("into mgo failed")
				h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_CROSS_DIFF),
					int8(rcc.StatusType_FAILED))
			} else {
				h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_CROSS_DIFF),
					int8(rcc.StatusType_SUCCESS))
			}
		}(v)
	}
	wg.Wait()
}

func (h *crossHandle) crossDiff(neId string, a, b interface{}) []mod.DiffRet {
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
		log.Infof("normal cross and %v", retDiff)
		return retDiff
	}

	for _, v := range changelog {
		retDiff = append(retDiff, mod.DiffRet{Type: v.Type, Path: v.Path, Controller: v.From, Adapter: v.To})
	}

	return retDiff
}

func (h *crossHandle) diffRetToMgo(neId string, crossConDiffRet []mod.CrossConRet) bool {
	newCheck := &mod.DiffMgo{
		NeId:          neId,
		CrossConDiffs: crossConDiffRet,
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
