// Package server 服务实现
package server

import (
	"context"
	"git.code.oa.com/trpc-go/trpc-go/log"
	rcc "git.code.oa.com/trpcprotocol/tianyuan/rcc_server_rcc"
	uuid "github.com/satori/go.uuid"
	"rcc/db"
	"rcc/mod"
	"rcc/service"
	"rcc/worker"
)

const (
	TIME_LAYOUT = "2006-01-01 12:33:36"
)

// TrpcServer 结构
type TrpcServer struct {
	handler worker.Handler
	dba     db.MysqlDbProxy
	mgo     db.DbProxy
}

// NewTrpcServer 创建trpc server
func NewTrpcServer(handler worker.Handler, mgo db.DbProxy, dba db.MysqlDbProxy) *TrpcServer {
	return &TrpcServer{handler: handler, dba: dba, mgo: mgo}
}

// CompareNe 工厂
func (s *TrpcServer) CompareNe(ctx context.Context, req *rcc.CompareNeReq, rsp *rcc.CompareNeRsp) error {
	taskId := uuid.NewV4()
	rsp.TaskId = taskId.String()
	info := &rcc.CommonRep{}
	neIdList := req.NeId
	log.Infof("entering **** ,%#v", neIdList)
	if len(neIdList) == 0 {
		info.Rc = rcc.ReturnCode_RC_INVALID_PARAM
		info.Msg = "入参不能为空"
		rsp.Info = info
		return nil
	}

	s.handler.CompareNe(taskId.String(), neIdList)

	info.Rc = rcc.ReturnCode_RC_OK
	info.Msg = "compare ne success"
	rsp.Info = info

	return nil
}

// GetCompareNeRet 是工厂
func (s *TrpcServer) GetCompareNeRet(ctx context.Context, req *rcc.GetCompareNeRetReq, rsp *rcc.GetCompareNeRetRsp) error {
	taskId := req.GetTaskId()

	rs, err := service.GetDiffResult(ctx, taskId, s.dba, s.mgo)
	if err != nil {
		info := &rcc.CommonRep{}
		info.Rc = rcc.ReturnCode_RC_FAILED
		info.Msg = "get diff result failed"
		rsp.Info = info
		return err
	}

	info := &rcc.CommonRep{}
	info.Rc = 0
	info.Msg = "ok"
	rsp.Info = info

	rsp.NeDiffs = mod.DiffMongoToPb(rs)

	return nil
}

// GetCurrentNeRet 工厂
func (s *TrpcServer) GetCurrentNeRet(ctx context.Context, req *rcc.CompareNeReq, rsp *rcc.GetCompareNeRetRsp) error {
	nes := req.GetNeId()

	rs, err := service.GetNeDiffResult(ctx, nes, s.mgo)
	if err != nil {
		info := &rcc.CommonRep{}
		info.Rc = rcc.ReturnCode_RC_FAILED
		info.Msg = "get diff result failed"
		rsp.Info = info
		return err
	}

	info := &rcc.CommonRep{}
	info.Rc = 0
	info.Msg = "ok"
	rsp.Info = info

	rsp.NeDiffs = mod.DiffMongoToPb(rs)

	return nil
}

// GetComparedLogs 方法
func (s *TrpcServer) GetComparedLogs(ctx context.Context, req *rcc.GetComparedLogReq, rsp *rcc.GetComparedLogRsp) error {
	rs, err := s.dba.GetCompareLog(ctx, req.GetPageNum(), req.GetHowMany())
	if err != nil {
		info := &rcc.CommonRep{}
		info.Rc = rcc.ReturnCode_RC_NO_RES
		info.Msg = "failed"
		rsp.Info = info

		return err
	}

	count, fail := s.dba.CountCompareLog()
	if fail != nil {
		info := &rcc.CommonRep{}
		info.Rc = rcc.ReturnCode_RC_NO_RES
		info.Msg = "failed"
		rsp.Info = info

		return fail
	}

	rsp.Info = &rcc.CommonRep{
		Rc:  rcc.ReturnCode_RC_OK,
		Msg: "ok"}

	var logs []*rcc.ComparedLog
	for _, v := range rs {
		logs = append(logs, &rcc.ComparedLog{
			Id:         v.Id,
			NeId:       v.NeId,
			NeName:     v.NeName,
			TaskId:     v.TaskId,
			CreateTime: v.CreateTime,
			EndTime:    v.EndTime,
			Type:       rcc.CompareType(v.CompareType),
			Status:     rcc.StatusType(v.Status),
			Item:       rcc.ItemType(v.Item)})
	}
	rsp.Logs = logs
	rsp.Total = count

	return nil
}
