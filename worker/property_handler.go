package worker

import (
	"context"
	rcc "git.code.oa.com/trpcprotocol/tianyuan/rcc_server_rcc"
	"github.com/r3labs/diff"
	"rcc/db"
	"rcc/mod"
	"rcc/proxy"
	"rcc/service"
	"strconv"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/log"
)

// PropertyHandler handler interface
type PropertyHandler interface {
	CompareNe(propertyQue map[string]db.CircleQueue, neList []string, taskId string)
	PropertyHandle()
	Run()
}

type propertyInfo struct {
	neId      string
	BaseQueue db.CircleQueue
}

type propertyHandle struct {
	compareLog      service.CompareLog
	neTopo          NeInfoStream
	streamData      map[string]baseInfo
	lock            sync.Mutex
	mgo             db.DbProxy
	sql             db.MysqlDbProxy
	handleDeadLock  sync.Mutex
	neInit          NeBaseInfo
	streamOld       map[string]interface{}
	minInterval     time.Duration
	neList          []string
	taskId          string
	propertyQueInfo map[string]db.CircleQueue
	flag            bool
}

func (h *propertyHandle) CompareNe(propertyQue map[string]db.CircleQueue, neList []string, taskId string) {
	log.Infof("entering property Compare ****")
	log.Infof("neList is %v", neList)
	log.Infof("que is %#v", propertyQue)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = neList
	h.taskId = taskId
	h.propertyQueInfo = make(map[string]db.CircleQueue)
	h.propertyQueInfo = propertyQue
	log.Infof("h.propertyQueInfo is %v", h.propertyQueInfo)
	h.flag = true
	h.do()
}

// NewPropertyHandler 是Property初始化
func NewPropertyHandler(mgo db.DbProxy, sql db.MysqlDbProxy) PropertyHandler {
	bH := &propertyHandle{
		mgo:             mgo,
		compareLog:      service.NewCompareLog(sql),
		propertyQueInfo: make(map[string]db.CircleQueue),
	}
	return bH
}

// 从队列缓存数据 处理
func (h *propertyHandle) PropertyHandle() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = h.neInit.neList
}

func (h *propertyHandle) Run() {
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
func (h *propertyHandle) run() {
	go h.do()
}

func (h *propertyHandle) do() {
	log.Infof("entering property diff do ****")
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
				int8(rcc.StatusType_RUNNING), int8(rcc.ItemType_PROPERTY_DIFF))

			//todo 兼容定制化巡检的定时队列obj获取
			ret, err := h.propertyQueInfo[neId].Pop()
			if err != nil {
				log.Errorf("no data in base queue and ne is %s and err%v", neId, err)
			}

			log.Infof("que is %#v", ret)
			streamData := ret.(NeProperty)
			var ctrNe, adpNe *mod.NeProperty
			ctrNe = &mod.NeProperty{WssMinDestSourceVoa: streamData.WssMinDestSourceVoa,
				WssMinSourceDestVoa: streamData.WssMinSourceDestVoa, WssMaxDestSourceVoa: streamData.WssMaxDestSourceVoa,
				WssMaxSourceDestVoa: streamData.WssMaxSourceDestVoa, CurrentSoftware: streamData.CurrentSoftware,
				HostName: streamData.HostName}

			adapterId := streamData.AdapterId
			result, err := proxy.GetNeInfoFromAdapter(adapterId, neId)
			if err != nil {
				h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_PROPERTY_DIFF),
					int8(rcc.StatusType_FAILED))
				log.Errorf("get ne from adapter failed(%v)", err)
				return
			}

			var cSoft, hName string
			var wssMind2s, wssMins2d, wssMaxd2s, wssMaxs2d float64
			for _, item := range result.Physical.Properties.Property {
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
				default:
					break
				}

				adpNe = &mod.NeProperty{WssMinDestSourceVoa: wssMind2s,
					WssMinSourceDestVoa: wssMins2d, WssMaxDestSourceVoa: wssMaxd2s,
					WssMaxSourceDestVoa: wssMaxs2d, CurrentSoftware: cSoft,
					HostName: hName}

				log.Infof("ctr property is %v", ctrNe)
				log.Infof("adp property is %v", adpNe)

				h.propertyDiff(neId, ctrNe, adpNe)
				h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_PROPERTY_DIFF),
					int8(rcc.StatusType_SUCCESS))
			}
		}(v)

		wg.Wait()
	}
}

func (h *propertyHandle) propertyDiff(neId string, a, b *mod.NeProperty) interface{} {
	log.Info("entering mgo diff in******")
	changelog, err := diff.Diff(a, b)
	if err != nil {
		log.Errorf("diff err %#v", err)
	}

	log.Infof("diff ret is %v", changelog)
	var propertyDiff []mod.DiffRet
	for _, v := range changelog {
		propertyDiff = append(propertyDiff, mod.DiffRet{Type: v.Type, Path: v.Path, Controller: v.From, Adapter: v.To})
	}
	if len(propertyDiff) <= 0 {
		//找到当前NE的历史对比结果，如果改正过后 要在mongoDb恢复到正常态数据
		propertyDiff = append(propertyDiff, mod.DiffRet{
			Type:       "normal",
			Path:       []string{"normal"},
			Controller: "normal",
			Adapter:    "normal",
		})
	}

	newCheck := &mod.DiffMgo{
		NeId:          neId,
		PropertyDiffs: propertyDiff,
	}
	filter := &mod.DiffMgo{NeId: neId}

	if yes, err := h.mgo.Exist(context.Background(), db.INSPECTION, filter); err != nil || !yes {
		if _, e := h.mgo.InsertOne(context.Background(), db.INSPECTION, newCheck); e != nil {
			log.Errorf("add ne(%s) property diff failed(%+v)", neId, err)
		}
	} else if e := h.mgo.FindOneAndUpdate(context.Background(), db.INSPECTION, filter, newCheck); e != nil {
		log.Errorf("update ne(%s) property diff failed(%+v)", neId, err)
	}
	return changelog
}
