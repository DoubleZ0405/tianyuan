package main

import (
	"github.com/DoubleZ0405/tianyuaxn/db"

	"github.com/DoubleZ0405/tianyuaxn/common/websocket"
	"github.com/DoubleZ0405/tianyuaxn/proxy"
	"github.com/DoubleZ0405/tianyuaxn/worker"
	"time"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

// 程序入口方法
func main() {
	//trpc-go框架加载配置会超时、手动设置下、不影响请求
	plugin.SetupTimeout = 10 * time.Second
	s := trpc.NewServer()

	conf, err := config.Load(trpc.ServerConfigPath, config.WithCodec("yaml"), config.WithProvider("file"))
	if err != nil {
		panic(err)
	}

	//db
	dbname := conf.GetString("client.dbname", "ocs_tianyuan")
	mgo := db.NewMongoDBClient(dbname, "")

	sql, err := db.NewMysqlDbProxy(trpc.ServerConfigPath)
	if err != nil {
		log.Fatalf("create db proxy failed(%v)", err)
		panic(err)
	}

	//websocket
	ws := websocket.NewWsServer("0.0.0.0:8081")
	go startWsService(ws)

	//
	proxy.SetConfig(conf)
	//
	topo := worker.NewBaseCommon(conf, mgo, sql, ws)
	topo.Run()
	//baseNeDiff
	//basehandler := worker.NewBaseHandler(mgo, sql)
	//basehandler.Run()
	//
	////PropertyDiff
	//propertyhandler := worker.NewPropertyHandler(mgo, sql)
	//propertyhandler.Run()
	//
	////XC-Diff
	//crossConHandler := worker.NewCrossHandler(mgo, sql)
	//crossConHandler.Run()
	//
	////Tp Diff
	//tpOpcHandler := worker.NewTpOpcHandler(mgo, sql)
	//tpOpcHandler.Run()
	//otuLineHandler := worker.NewOtuLineHandler(mgo, sql)
	//otuLineHandler.Run()
	//otuClientHandler := worker.NewOtuClientHandler(mgo, sql)
	//otuClientHandler.Run()

	//算子 暂时不开启
	//chanSize := conf.GetInt("custom.channel.size", 100)
	//ch := make(chan mod.NeBase, chanSize)
	// calc可以多个实例
	//calcCnt := conf.GetInt("custom.calc.count", 1)
	//calcs := make([]worker.Calc, calcCnt)
	//for i := 0; i < calcCnt; i++ {
	//	calcs[i] = worker.NewCalc(i, topo, ch, basehandler)
	//	calcs[i].Run()
	//}

	//对外服务
	//apiHandler := worker.NewHandler(conf, mgo, sql, topo)
	//:= server.NewTrpcServer(apiHandler, mgo, sql)

	//rcc.RegisterRccService(s, service)
	log.Info("server start")
	err = s.Serve()
	if err != nil {
		log.Errorf("server start failed: %s", err)
		return
	}
	return
}

func startWsService(ws *websocket.WsServer) {
	err := ws.Start()
	if nil != err {
		panic(err)
	}
}
