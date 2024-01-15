package service

import (
	"context"
	"github.com/DoubleZ0405/tianyuaxn/db"
	"github.com/DoubleZ0405/tianyuaxn/mod"
	"trpc.group/trpc-go/trpc-go/log"
)

// GetDiffResult 通过任务ID获取相关所有网元检查数据
func GetDiffResult(ctx context.Context, taskId string, sql db.MysqlDbProxy, mgo db.DbProxy) ([]*mod.DiffMgo, error) {
	var rs []*mod.DiffMgo

	nes, err := sql.GetCompareLogByTaskId(taskId)
	if err != nil {
		return nil, err
	}

	for _, ne := range nes {
		var diff []*mod.DiffMgo
		filter := &mod.DiffMgo{
			NeId: ne,
		}

		fail := mgo.Find(ctx, db.INSPECTION, filter, &diff)
		if fail != nil {
			return nil, fail
		}

		if len(diff) < 1 {
			log.Warnf("ne %s result not found in mongo", ne)
			continue
		}

		rs = append(rs, &mod.DiffMgo{
			Id:            diff[0].Id,
			NeId:          ne,
			BaseDiffs:     diff[0].BaseDiffs,
			OcmDiffs:      diff[0].OcmDiffs,
			PropertyDiffs: diff[0].PropertyDiffs,
			CrossConDiffs: diff[0].CrossConDiffs,
			TpClientDiffs: diff[0].TpClientDiffs,
			TpLineDiffs:   diff[0].TpLineDiffs,
			Opc4Diffs:     diff[0].Opc4Diffs,
		})
	}
	log.Infof("diff rs from mongo is %#v", rs)

	return rs, nil
}

// GetNeDiffResult  直接获取网元最新检查数据
func GetNeDiffResult(ctx context.Context, nes []string, mgo db.DbProxy) ([]*mod.DiffMgo, error) {
	var rs []*mod.DiffMgo
	var diff []*mod.DiffMgo

	for _, ne := range nes {
		filter := &mod.DiffMgo{
			NeId: ne,
		}

		fail := mgo.Find(ctx, db.INSPECTION, filter, &diff)
		if fail != nil {
			return nil, fail
		}

		if len(diff) < 1 {
			log.Warnf("ne %s result not found in mongo", ne)
			continue
		}

		rs = append(rs, diff[0])
	}

	return rs, nil
}
