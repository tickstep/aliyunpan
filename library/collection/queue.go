package collection

import (
	"strings"
	"sync"
)

type QueueItem interface {
	HashCode() string
}

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

func (q *Queue) PushUnique(item interface{}) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queueList == nil {
		q.queueList = []interface{}{}
	} else {
		for _, qItem := range q.queueList {
			if strings.Compare(item.(QueueItem).HashCode(), qItem.(QueueItem).HashCode()) == 0 {
				return
			}
		}
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

func (q *Queue) Remove(item interface{}) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queueList == nil {
		q.queueList = []interface{}{}
	}
	if len(q.queueList) == 0 {
		return
	}
	j := 0
	for _, qItem := range q.queueList {
		if strings.Compare(item.(QueueItem).HashCode(), qItem.(QueueItem).HashCode()) != 0 {
			q.queueList[j] = qItem
			j++
		}
	}
	q.queueList = q.queueList[:j]
	return
}

func (q *Queue) Contains(item interface{}) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queueList == nil {
		q.queueList = []interface{}{}
	}
	if len(q.queueList) == 0 {
		return false
	}
	for _, qItem := range q.queueList {
		if strings.Compare(item.(QueueItem).HashCode(), qItem.(QueueItem).HashCode()) == 0 {
			return true
		}
	}
	return false
}
