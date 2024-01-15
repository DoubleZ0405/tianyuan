package worker

import (
	"context"
	"git.code.oa.com/trpc-go/trpc-go/log"
	rcc "git.code.oa.com/trpcprotocol/tianyuan/rcc_server_rcc"
	"github.com/r3labs/diff"
	"rcc/db"
	"rcc/mod"
	"rcc/proxy"
	"rcc/service"
	"strconv"
	"sync"
	"time"
)

// OcmHandler handler interface
type OcmHandler interface {
	CompareNe(ocmQue map[string]map[string]db.CircleQueue, neList []string, taskId string)
	OcmHandle()
	Run()
}

type ocmInfo struct {
	neId     string
	OcmQueue db.CircleQueue
}

type ocmHandle struct {
	compareLog     service.CompareLog
	neTopo         NeInfoStream
	streamData     map[string]ocmInfo
	lock           sync.Mutex
	mgo            db.DbProxy
	sql            db.MysqlDbProxy
	handleDeadLock sync.Mutex
	neInit         NeBaseInfo
	streamOld      map[string]interface{}
	minInterval    time.Duration
	neList         []string
	taskId         string
	ocmQueInfo     map[string]map[string]db.CircleQueue
	flag           bool
}

func (h *ocmHandle) CompareNe(ocmQue map[string]map[string]db.CircleQueue, neList []string, taskId string) {
	log.Infof("entering ocm Compare ****")
	log.Infof("neList is %v", neList)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = neList
	h.taskId = taskId
	h.ocmQueInfo = make(map[string]map[string]db.CircleQueue)
	h.ocmQueInfo = ocmQue
	h.flag = true
	h.do()
}

// NewOcmHandler 是Ocm
func NewOcmHandler(mgo db.DbProxy, sql db.MysqlDbProxy) OcmHandler {
	bH := &ocmHandle{
		mgo:        mgo,
		compareLog: service.NewCompareLog(sql),
		ocmQueInfo: make(map[string]map[string]db.CircleQueue),
	}
	return bH
}

func (h *ocmHandle) OcmHandle() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = h.neInit.neList
}

func (h *ocmHandle) Run() {
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

func (h *ocmHandle) run() {
	go h.do()
}

func (h *ocmHandle) do() {
	log.Infof("entering ocm diff do ****")
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
				int8(rcc.StatusType_RUNNING), int8(rcc.ItemType_OCM_DIFF))

			var aId = ""
			var adapterInfo *mod.AdapterNe
			var ocmRet []mod.OcmDiffRet

			if _, ok := h.ocmQueInfo[neId]; ok {
				for index, channelInfo := range h.ocmQueInfo[neId] {
					ret, err := channelInfo.Pop()
					log.Infof("channel %v index %s", ret, index)
					if err != nil {
						log.Errorf("no data in client queue and ne is %s and err %v",
							neId, err)
						continue
					}

					streamData := ret.(NeOcmGripChannel)
					if aId == "" {
						aId = streamData.AdapterId
						result, err := proxy.GetNeInfoFromAdapter(aId, neId)
						if err != nil {
							h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_OCM_DIFF),
								int8(rcc.StatusType_FAILED))
							log.Errorf("get ne from adapter failed(%v)", err)
							return
						}
						adapterInfo = result
					}

					var ctrNeOcm, adpNeOcm mod.Channels
					ctrNeOcm = mod.Channels{
						Index:          streamData.Index,
						LowerFrequency: streamData.LowerFrequency,
						UpperFrequency: streamData.UpperFrequency}

					for _, ocm := range adapterInfo.Physical.OcmGripGroups {
						if adapterInfo.Physical.UsedGripGroupID == ocm.Index {
							//将slice转map方便后续进行数值key检查
							adpNeOcmRearrange := make(map[string]mod.Channels)
							for _, channel := range ocm.Channels {
								adpNeOcmRearrange[strconv.Itoa(channel.Index)] = channel
							}

							if v, ok := adpNeOcmRearrange[strconv.Itoa(streamData.Index)]; ok {
								adpNeOcm = mod.Channels{
									Index:          v.Index,
									LowerFrequency: v.LowerFrequency,
									UpperFrequency: v.UpperFrequency}
							} else {
								adpNeOcm = mod.Channels{}
							}

							if ctrNeOcm == (mod.Channels{}) && adpNeOcm == (mod.Channels{}) {
								continue
							}

							channelDiff := h.channelDiff(ctrNeOcm, adpNeOcm)
							if len(channelDiff) == 0 {
								continue
							}

							ocmRet = append(ocmRet, mod.OcmDiffRet{
								Index: index, ChannelDiffs: channelDiff})
						}
					}
				}

				if !h.diffRetToMgo(neId, ocmRet) {
					log.Errorf("merge into mongo failed")
					h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_OCM_DIFF),
						int8(rcc.StatusType_FAILED))
				} else {
					h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_OCM_DIFF),
						int8(rcc.StatusType_SUCCESS))
				}
			}
		}(v)
	}
	wg.Wait()
}

func (h *ocmHandle) channelDiff(a, b interface{}) []mod.DiffRet {
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

func (h *ocmHandle) diffRetToMgo(neId string, ocmDiffRet []mod.OcmDiffRet) bool {
	newCheck := &mod.DiffMgo{
		NeId:     neId,
		OcmDiffs: ocmDiffRet,
	}
	filter := mod.DiffMgo{NeId: neId}

	if yes, err := h.mgo.Exist(context.Background(), db.INSPECTION, filter); err != nil || !yes {
		if _, e := h.mgo.InsertOne(context.Background(), db.INSPECTION, newCheck); e != nil {
			log.Errorf("add ne(%s) ocm diff failed(%+v)", neId, err)
			return false
		}
	} else if e := h.mgo.FindOneAndUpdate(context.Background(), db.INSPECTION, filter, newCheck); e != nil {
		log.Errorf("update ne(%s) ocm diff failed(%+v)", neId, err)
		return false
	}

	return true
}
