package worker

import (
	"context"
	"github.com/DoubleZ0405/tianyuaxn/db"
	"github.com/DoubleZ0405/tianyuaxn/mod"
	"github.com/DoubleZ0405/tianyuaxn/proxy"
	"github.com/DoubleZ0405/tianyuaxn/service"
	"github.com/r3labs/diff"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

// BaseHandler 是基础属性处理接口 工厂方法
type BaseHandler interface {
	CompareNe(baseQue map[string]db.CircleQueue, neList []string, taskId string)
	BaseHandle()
	Run()
}

type baseInfo struct {
	neId      string
	BaseQueue db.CircleQueue
}

type baseHandle struct {
	compareLog     service.CompareLog
	neTopo         NeInfoStream
	streamData     map[string]baseInfo
	lock           sync.Mutex
	mgo            db.DbProxy
	handleDeadLock sync.Mutex
	neInit         NeBaseInfo
	streamOld      map[string]interface{}
	minInterval    time.Duration
	neList         []string
	taskId         string
	baseQueInfo    map[string]db.CircleQueue
	flag           bool
}

func (h *baseHandle) CompareNe(baseQue map[string]db.CircleQueue, neList []string, taskId string) {
	log.Infof("entering base Compare ****")
	log.Infof("neList is %v", neList)
	log.Infof("que is %#v", baseQue)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = neList
	h.taskId = taskId
	h.baseQueInfo = make(map[string]db.CircleQueue)
	h.baseQueInfo = baseQue
	for k, v := range h.baseQueInfo {
		log.Infof("node is %s", k)
		log.Infof("que is %v", v)
	}
	log.Infof("h.baseQueInfo is %v", h.baseQueInfo)
	h.flag = true
	h.do()
}

// NewBaseHandler 是初始化必要参数
func NewBaseHandler(mgo db.DbProxy, sql db.MysqlDbProxy) BaseHandler {
	bH := &baseHandle{
		mgo:         mgo,
		compareLog:  service.NewCompareLog(sql),
		baseQueInfo: make(map[string]db.CircleQueue),
	}
	return bH
}

// 从队列缓存数据 处理
func (h *baseHandle) BaseHandle() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.neList = h.neInit.neList
}

func (h *baseHandle) Run() {
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
func (h *baseHandle) run() {
	go h.do()
}

// 并发比对 上层算子可开多实例
// 可用作同步方法
func (h *baseHandle) do() {
	log.Infof("entering diff do ****")
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
				int8(rcc.StatusType_RUNNING), int8(rcc.ItemType_BASE_DIFF))
			//todo 兼容定制化巡检的定时队列obj获取
			ret, err := h.baseQueInfo[neId].Pop()
			if err != nil {
				h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_BASE_DIFF),
					int8(rcc.StatusType_FAILED))
				log.Errorf("no data in base queue and ne is %s and err%v", neId, err)
			}

			log.Infof("que is %#v", ret)
			streamData := ret.(NeBase)
			var ctrNe, adpNe *mod.NeBase
			ctrNe = &mod.NeBase{Port: streamData.port, LoginName: streamData.loginName,
				LoginPasswd: streamData.loginPasswd, VendorType: streamData.vendorType,
				UsedGripGroupId: streamData.usedGripGroupId}

			adapterId := streamData.adapterId
			result, err := proxy.GetNeInfoFromAdapter(adapterId, neId)
			if err != nil {
				h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_BASE_DIFF),
					int8(rcc.StatusType_FAILED))
				log.Errorf("get ne from adapter failed(%v)", err)
				return
			}

			adpNe = &mod.NeBase{Port: result.Physical.Port, LoginName: result.Physical.LoginName,
				LoginPasswd: result.Physical.LoginPasswd, VendorType: result.Physical.VendorType,
				UsedGripGroupId: result.Physical.UsedGripGroupID}

			h.baseDiff(neId, ctrNe, adpNe)
			h.compareLog.UpdateLog([]string{neId}, h.taskId, int8(rcc.ItemType_BASE_DIFF),
				int8(rcc.StatusType_SUCCESS))
		}(v)
	}
	wg.Wait()
}

func (h *baseHandle) baseDiff(neId string, a, b *mod.NeBase) interface{} {
	log.Info("entering mgo diff in******")
	changelog, err := diff.Diff(a, b)
	if err != nil {
		log.Errorf("diff err %#v", err)
	}

	log.Infof("diff ret is %v", changelog)
	var baseDiff []mod.DiffRet
	for _, v := range changelog {
		baseDiff = append(baseDiff, mod.DiffRet{Type: v.Type, Path: v.Path, Controller: v.From, Adapter: v.To})
	}

	filter := &mod.DiffMgo{NeId: neId}
	if len(baseDiff) <= 0 {
		log.Infof("baseDiff: controller & adapter is the same")
		////找到当前NE的历史对比结果，如果改正过后 要在mongoDb恢复到正常态数据
		//if err := h.mgo.FindOneAndDelete(context.Background(), db.INSPECTION, filter); err != nil {
		//	log.Debugf("in BaseDiff, Find Ne(%s) is normal,backup err is (%s)", neId, err)
		//}
		baseDiff = append(baseDiff, mod.DiffRet{
			Type:       "normal",
			Path:       []string{"normal"},
			Controller: "normal",
			Adapter:    "normal",
		})
	}

	newCheck := &mod.DiffMgo{
		NeId:      neId,
		BaseDiffs: baseDiff,
	}

	if yes, err := h.mgo.Exist(context.Background(), db.INSPECTION, filter); err != nil || !yes {
		if _, e := h.mgo.InsertOne(context.Background(), db.INSPECTION, newCheck); e != nil {
			log.Errorf("add ne(%s) base diff failed(%+v)", neId, err)
		}
	} else if e := h.mgo.FindOneAndUpdate(context.Background(), db.INSPECTION, filter, newCheck); e != nil {
		log.Errorf("update ne(%s) base diff failed(%+v)", neId, err)
	}
	return changelog
}
