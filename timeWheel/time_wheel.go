package timeWheel

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

type TaskElement struct {
	task  func()
	pos   int
	cycle int
	key   string
}

type TimeWheel struct {
	sync.Once
	interval       time.Duration
	slots          []*list.List
	ticker         *time.Ticker
	stopChan       chan struct{}
	addTaskChan    chan *TaskElement
	removeTaskChan chan string
	taskMap        map[string]*list.Element
	curSlot        int
}

func NewTimeWheel(slotNum int, interval time.Duration) *TimeWheel {
	if slotNum < 10 {
		slotNum = 10
	}
	if interval <= 0 {
		interval = time.Second
	}

	t := &TimeWheel{
		interval:       interval,
		slots:          make([]*list.List, 0, slotNum),
		ticker:         time.NewTicker(interval),
		stopChan:       make(chan struct{}),
		addTaskChan:    make(chan *TaskElement),
		removeTaskChan: make(chan string),
		taskMap:        make(map[string]*list.Element),
	}
	for i := 0; i < slotNum; i++ {
		t.slots = append(t.slots, list.New())
	}

	go t.run()

	return t
}

func (t *TimeWheel) Stop() {
	t.Do(func() {
		t.ticker.Stop()
		close(t.stopChan)
	})
}

func (t *TimeWheel) AddTask(key string, task func(), executeAt time.Time) {
	pos, cycle := t.getPosAndCircle(executeAt)
	t.addTaskChan <- &TaskElement{
		task:  task,
		pos:   pos,
		cycle: cycle,
		key:   key,
	}
}

func (t *TimeWheel) RemoveTask(key string) {
	t.removeTaskChan <- key
}

func (t *TimeWheel) run() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("TimeWheel run occurs panic, err: %v\n", r)
		}
	}()

	for {
		select {
		case <-t.stopChan:
			return
		case <-t.ticker.C:
			t.tick()
		case task := <-t.addTaskChan:
			t.addTask(task)
		case key := <-t.removeTaskChan:
			t.removeTask(key)
		}
	}
}

func (t *TimeWheel) tick() {
	l := t.slots[t.curSlot]
	defer t.circleIncr()
	t.execute(l)
}

func (t *TimeWheel) execute(l *list.List) {
	for e := l.Front(); e != nil; {
		task, _ := e.Value.(*TaskElement)
		if task.cycle > 0 {
			task.cycle--
			e = e.Next()
			continue
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("execute task panic, task: %+v\n", task)
				}
			}()
			task.task()
		}()

		next := e.Next()
		l.Remove(e)
		delete(t.taskMap, task.key)
		e = next
	}
}

func (t *TimeWheel) addTask(task *TaskElement) {
	l := t.slots[task.pos]
	if _, ok := t.taskMap[task.key]; ok {
		t.removeTask(task.key)
	}
	e := l.PushBack(task)
	t.taskMap[task.key] = e
}

func (t *TimeWheel) removeTask(key string) {
	e, ok := t.taskMap[key]
	if !ok {
		return
	}
	delete(t.taskMap, key)
	task, _ := e.Value.(*TaskElement)
	t.slots[task.pos].Remove(e)
}

func (t *TimeWheel) getPosAndCircle(executeAt time.Time) (int, int) {
	delay := int(time.Until(executeAt))
	cycle := delay / (int(t.interval) * len(t.slots))
	pos := (t.curSlot + delay/int(t.interval)) % len(t.slots)
	return pos, cycle
}

func (t *TimeWheel) circleIncr() {
	t.curSlot = (t.curSlot + 1) % len(t.slots)
}
