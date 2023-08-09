package blocks

import (
	"container/list"
	//	"fmt"

	"github.com/epfl-dcsl/schedsim/engine"
)

// Processor Interface describes the main processor functionality used
// in describing a topology
type Processor interface {
	engine.ActorInterface
	SetReqDrain(rd RequestDrain) // We might want to specify different drains for different processors or use the same drain for all
	SetCtxCost(cost float64)
}

// generic processor: All processors should have it as an embedded field
type genericProcessor struct {
	engine.Actor
	reqDrain RequestDrain
	ctxCost  float64
}

func (p *genericProcessor) SetReqDrain(rd RequestDrain) {
	p.reqDrain = rd
}

func (p *genericProcessor) SetCtxCost(cost float64) {
	p.ctxCost = cost
}

// RTCProcessor is a run to completion processor
type RTCProcessor struct {
	genericProcessor
	scale float64
}

// Run is the main processor loop
func (p *RTCProcessor) Run() {
	for {
		req := p.ReadInQueue()
		p.Wait(req.GetServiceTime() + p.ctxCost)
		if monitorReq, ok := req.(*MonitorReq); ok {
			monitorReq.finalLength = p.GetInQueueLen(0)
		}
		p.reqDrain.TerminateReq(req)
	}
}

// TSProcessor is a time sharing processor
type TSProcessor struct {
	genericProcessor
	quantum float64
}

// NewTSProcessor returns a new *TSProcessor
func NewTSProcessor(quantum float64) *TSProcessor {
	return &TSProcessor{quantum: quantum}
}

// Run is the main processor loop
func (p *TSProcessor) Run() {
	for {
		req := p.ReadInQueue()

		if req.GetServiceTime() <= p.quantum {
			p.Wait(req.GetServiceTime() + p.ctxCost)
			p.reqDrain.TerminateReq(req)
		} else {
			p.Wait(p.quantum + p.ctxCost)
			req.SubServiceTime(p.quantum)
			p.WriteInQueue(req)
		}
	}
}

// PSProcessor is a processor sharing processor
type PSProcessor struct {
	genericProcessor
	workerCount int
	count       int // how many concurrent requests
	reqList     *list.List
	curr        *list.Element
	prevTime    float64
}

// NewPSProcessor returns a new *PSProcessor
func NewPSProcessor() *PSProcessor {
	return &PSProcessor{workerCount: 1, reqList: list.New()}
}

// SetWorkerCount sets the number of workers in a processor sharing processor
func (p *PSProcessor) SetWorkerCount(count int) {
	p.workerCount = count
}

func (p *PSProcessor) getMinService() *list.Element {
	minS := p.reqList.Front().Value.(*Request).ServiceTime
	minI := p.reqList.Front()
	for e := p.reqList.Front(); e != nil; e = e.Next() {
		val := e.Value.(*Request).ServiceTime
		if val < minS {
			minS = val
			minI = e
		}
	}
	return minI
}

func (p *PSProcessor) getFactor() float64 {
	if p.workerCount > p.count {
		return 1.0
	}
	return float64(p.workerCount) / float64(p.count)
}

func (p *PSProcessor) updateServiceTimes() {
	currTime := engine.GetTime()
	diff := (currTime - p.prevTime) * p.getFactor()
	p.prevTime = currTime
	for e := p.reqList.Front(); e != nil; e = e.Next() {
		req := e.Value.(engine.ReqInterface)
		req.SubServiceTime(diff)
	}
}

// Run is the main processor loop
func (p *PSProcessor) Run() {
	var d float64
	d = -1
	for {
		intr, newReq := p.WaitInterruptible(d)
		//update times
		p.updateServiceTimes()
		if intr {
			req := p.curr.Value.(engine.ReqInterface)
			p.reqDrain.TerminateReq(req)
			p.reqList.Remove(p.curr)
			p.count--
		} else {
			p.count++
			p.reqList.PushBack(newReq)
		}
		if p.count > 0 {
			p.curr = p.getMinService()
			d = p.curr.Value.(engine.ReqInterface).GetServiceTime() / p.getFactor()
		} else {
			d = -1
		}
	}
}

type BoundedProcessor struct {
	genericProcessor
	bufSize int
}

func NewBoundedProcessor(bufSize int) *BoundedProcessor {
	return &BoundedProcessor{bufSize: bufSize}
}

// Run is the main processor loop
func (p *BoundedProcessor) Run() {
	var factor float64
	for {
		req := p.ReadInQueue()

		if colorReq, ok := req.(*ColoredReq); ok {
			if colorReq.color == 1 {
				factor = 2
			} else {
				factor = 1
			}
		}
		p.Wait(factor * req.GetServiceTime())
		len := p.GetOutQueueLen(0)
		if len < p.bufSize {
			p.WriteOutQueue(req)
		} else {
			p.reqDrain.TerminateReq(req)
		}
	}
}

type BoundedProcessor2 struct {
	genericProcessor
}

// Run is the main processor loop
func (p *BoundedProcessor2) Run() {
	var factor float64
	for {
		req := p.ReadInQueue()

		if colorReq, ok := req.(*ColoredReq); ok {
			if colorReq.color == 0 {
				factor = 2
			} else {
				factor = 1
			}
		}
		p.Wait(factor * req.GetServiceTime())
		p.reqDrain.TerminateReq(req)
	}
}

// LimitedPSProcessor is a limited processor sharing processor
// i.e., at any given time we have a limit, N, such that if another request, r, arrives when N are already running, r would need to wait for one of the N to finish
// type LimitedPSProcessor struct {
// 	genericProcessor
// 	workerCount int // cores
// 	count       int // how many concurrent requests
// 	reqList     *list.List
// 	curr        *list.Element
// 	limit		int
// 	prevTime    float64
// 	excessList	*list.List
// }

// func newLimitedPSProcessor() *LimitedPSProcessor {
// 	return &LimitedPSProcessor{workerCount: 1, reqList: list.New(), limit: 1, excessList: list.New()}
// }

// // SetWorkerCount sets the number of workers in a processor sharing processor
// func (p *LimitedPSProcessor) SetWorkerCount(count int) {
// 	p.workerCount = count
// }

// func (p *LimitedPSProcessor) setLimit(limit int) {
// 	p.limit = limit
// }

// func (p *LimitedPSProcessor) getMinService() *list.Element {
// 	minS := p.reqList.Front().Value.(*Request).ServiceTime // the service time of minI
// 	minI := p.reqList.Front() // represents the task that requires minS
// 	for e := p.reqList.Front(); e != nil; e = e.Next() {
// 		val := e.Value.(*Request).ServiceTime
// 		if val < minS {
// 			minS = val
// 			minI = e
// 		}
// 	}
// 	return minI
// }

// func (p *LimitedPSProcessor) getFactor() float64 {
// 	// return min(1, ratio between number of cores to number of concurrent requests)
// 	if p.workerCount > p.count {
// 		return 1.0
// 	}
// 	return float64(p.workerCount) / float64(p.count)
// }

// func (p *LimitedPSProcessor) updateServiceTimes() {
// 	currTime := engine.GetTime()
// 	diff := (currTime - p.prevTime) * p.getFactor()
// 	p.prevTime = currTime
// 	for e := p.reqList.Front(); e != nil; e = e.Next() {
// 		req := e.Value.(engine.ReqInterface)
// 		req.SubServiceTime(diff)
// 	}
// }

// // Run is the main processor loop
// func (p *LimitedPSProcessor) Run() {
// 	var d float64
// 	d = -1
// 	for {
// 		intr, newReq := p.WaitInterruptible(d)
// 		p.updateServiceTimes()

// 		if (p.count == p.limit) {
// 			p.excessList.PushBack(newReq)
// 		} 
// 		if intr {
// 			req := p.curr.Value.(engine.ReqInterface)
// 			p.reqDrain.TerminateReq(req)
// 			p.reqList.Remove(p.curr)
// 			p.count--
// 			if(p.excessList.Len() > 0) {
// 				front := p.excessList.Front()
// 				p.reqList.PushBack(front)
// 				p.excessList.Remove(front)
// 			}
// 		} else {
// 			p.count++
// 			p.reqList.PushBack(newReq)
// 		}
// 		if p.count > 0 {
// 			p.curr = p.getMinService()
// 			d = p.curr.Value.(engine.ReqInterface).GetServiceTime() / p.getFactor()
// 		} else {
// 			d = -1
// 		}
		
// 	}
// }


// LimitedPSProcessor is a limited processor sharing processor
// i.e., at any given time we have a limit, N, such that if another request, r, arrives when N are already running, r would need to wait for one of the N to finish
type LimitedPSProcessor struct {
	PSProcessor
	limit		int
	overflow	*list.List
}

func newLimitedPSProcessor() *LimitedPSProcessor {
	return &LimitedPSProcessor{PSProcessor: *NewPSProcessor(), limit: 1, overflow: list.New()}
}

func (p *LimitedPSProcessor) setLimit(limit int) {
	p.limit = limit
}

func (p *LimitedPSProcessor) updateServiceTimesIncludingOverflow() {
	currTime := engine.GetTime()
	diff := (currTime - p.prevTime) * p.getFactor()
	p.prevTime = currTime
	for e := p.reqList.Front(); e != nil; e = e.Next() {
		req := e.Value.(engine.ReqInterface)
		req.SubServiceTime(diff)
	}
	for e := p.overflow.Front(); e != nil; e = e.Next() {
		req := e.Value.(engine.ReqInterface)
		req.SubServiceTime(diff)
	}
}

func (p *LimitedPSProcessor) Run() {
	var d float64
	d = -1
	for {
		intr, newReq := p.WaitInterruptible(d)
		p.updateServiceTimesIncludingOverflow()

		if (p.count == p.limit) {
			p.overflow.PushBack(newReq)
		} 
		if intr {
			req := p.curr.Value.(engine.ReqInterface)
			p.reqDrain.TerminateReq(req)
			p.reqList.Remove(p.curr)
			p.count--
			if(p.overflow.Len() > 0) {
				front := p.overflow.Front()
				p.reqList.PushBack(front)
				p.count++
				p.overflow.Remove(front)
			}
		} else {
			p.count++
			p.reqList.PushBack(newReq)
		}
		if p.count > 0 {
			p.curr = p.getMinService()
			d = p.curr.Value.(engine.ReqInterface).GetServiceTime() / p.getFactor()
		} else {
			d = -1
		}
		
	}
}
