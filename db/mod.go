package db

// CompareLog 是CompareLog的日志
type CompareLog struct {
	Id          int64  `db:"id"`
	NeId        string `db:"ne_id"`
	NeName      string `db:"friendly_name"`
	TaskId      string `db:"task_id"`
	CreateTime  int32  `db:"create_time"`
	EndTime     int32  `db:"end_time"`
	Status      int8   `db:"status"`
	CompareType int8   `db:"type"`
	Item        int8   `db:"item"`
}
