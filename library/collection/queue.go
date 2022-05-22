package collection

import "sync"

type Queue struct {
	queueList []interface{}

	mutex sync.Mutex
}

func NewFifoQueue() *Queue {
	return &Queue{}
}

func (q *Queue) Push(item interface{}) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queueList == nil {
		q.queueList = []interface{}{}
	}
	q.queueList = append(q.queueList, item)
}

func (q *Queue) Pop() interface{} {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queueList == nil {
		q.queueList = []interface{}{}
	}
	if len(q.queueList) == 0 {
		return nil
	}
	item := q.queueList[0]
	q.queueList = q.queueList[1:]
	return item
}

func (q *Queue) Length() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return len(q.queueList)
}
