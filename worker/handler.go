package worker

import (
	"git.code.oa.com/trpc-go/trpc-go/config"
	"rcc/db"
)

// Handler 是handler工厂方法
type Handler interface {
	Run()
	CompareNe(taskId string, neList []string)
}

// NewHandler 是初始化
func NewHandler(conf config.Config, mgo db.DbProxy, sql db.MysqlDbProxy, topo NeInfoStream) Handler {
	h := &handler{
		mgo:  mgo,
		topo: topo,
		db:   sql,
	}
	h.InitConf(conf)
	return h
}

type handler struct {
	topo NeInfoStream
	mgo  db.DbProxy
	db   db.MysqlDbProxy
	conf config.Config
}

func (h *handler) Run() {
	//todo
}

func (h *handler) InitConf(conf config.Config) {
	h.conf = conf
}

func (h *handler) CompareNe(taskId string, neList []string) {
	h.topo.CompareNe(h.conf, taskId, neList, h.mgo, h.db)
	return
}
