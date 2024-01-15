package db

import (
	"errors"
	"rcc/mod"
)

// CircleQueue 是环状队列interface
type CircleQueue interface {
	Push(v interface{})
	Pop() (v interface{}, err error)
	IsFull() bool
	IsEmpty() bool
	Size() int
	Clear() bool
}

// NewCircleQueue 是创建环形队列，maxSize必须>0，传0自动改为1
func NewCircleQueue(maxSize int) CircleQueue {
	if maxSize <= 0 {
		maxSize = 2000
	}
	return &circleQueue{maxSize: maxSize, array: make([]interface{}, maxSize)}
}

func (q *circleQueue) Clear() bool {
	q = &circleQueue{maxSize: q.maxSize, array: make([]interface{}, q.maxSize)}
	return true
}

// QueueData 是queue data
type QueueData struct {
	port        int
	loginName   string
	loginPasswd string
	vendorType  string
}

// 环状队列实现
type circleQueue struct {
	maxSize  int
	array    []interface{}
	head     int
	tail     int
	baseInfo mod.NeBase
}

// 入队列，如果队满自动溢出
func (q *circleQueue) Push(v interface{}) {
	if q.IsFull() {
		q.Pop()
	}
	q.array[q.tail] = v
	//算法，计算队尾的位置
	q.tail = (q.tail + 1) % q.maxSize
	return
}

// 出队列
func (q *circleQueue) Pop() (v interface{}, err error) {
	if q.IsEmpty() {
		return QueueData{}, errors.New("queue empty")
	}
	v = q.array[q.head]
	q.head = (q.head + 1) % q.maxSize
	return v, err
}

// 队列是否满了
func (q *circleQueue) IsFull() bool {
	// 队满情况[ (tail+1)%maxSize == head ]
	return (q.tail+1)%q.maxSize == q.head
}

// 队列是否为空
func (q *circleQueue) IsEmpty() bool {
	return q.tail == q.head
}

// 队列多少个元素
func (q *circleQueue) Size() int {
	return (q.tail + q.maxSize - q.head) % q.maxSize
}
