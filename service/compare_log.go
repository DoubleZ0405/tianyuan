package service

import (
	"rcc/db"
	"trpc.group/trpc-go/trpc-go/log"
)

// CompareLog 是对比日志工厂
type CompareLog interface {
	InsertLog(neList []string, taskId string, compareStatus int8,
		compareType int8, compareItem int8)
	UpdateLog(neList []string, taskId string, item int8, status int8)
}

type compareLog struct {
	sql db.MysqlDbProxy
}

// NewCompareLog 是初始化
func NewCompareLog(sql db.MysqlDbProxy) CompareLog {
	cl := &compareLog{
		sql: sql,
	}
	return cl
}

func (h *compareLog) InsertLog(neList []string, taskId string, compareStatus int8,
	compareType int8, compareItem int8) {
	if err := h.sql.InsertCompareLog(neList, taskId, compareStatus, compareType, compareItem); err != nil {
		log.Errorf("task %s ne %s insert base diff failed", taskId, neList)
	}
}

func (h *compareLog) UpdateLog(neList []string, taskId string, item int8, status int8) {
	if err := h.sql.UpdateCompareLogStatus(neList, taskId, item, status); err != nil {
		log.Errorf("update taskId %s neId %s status failed err:%v",
			taskId, neList, err)
	}
}
