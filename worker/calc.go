package worker

import (
	"github.com/DoubleZ0405/tianyuaxn/mod"
	"trpc.group/trpc-go/trpc-go/log"
)

// Calc interface 是算子的工厂方法
type Calc interface {
	Run()
}

// NewCalc 创建一个Calc对象
func NewCalc(id int, topo NeInfoStream, ch chan mod.NeBase, handler interface{}) Calc {
	return nil
	//return &calc{id: id, topo: topo, ch: ch, handler: handler}
}

// calc实例
type calc struct {
	topo    NeInfoStream
	ch      chan mod.NeBase
	handler Handler
	topoVer uint64
	neList  []string
	id      int
}

// 执行方法
func (c *calc) Run() {
	log.Infof("calc(%d) start", c.id)
	go c.run()
}

// 并发routine 执行方法
func (c *calc) run() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered in f", r)
		}
	}()

	for range c.ch {
		if neList, ver := c.topo.getNeList(c.topoVer); neList != nil {
			c.neList, c.topoVer = neList, ver
		}
		//todo
	}
}
