package timeWheel

import (
	"testing"
	"time"
)

func Test_timeWheel(t *testing.T) {
	timeWheel := NewTimeWheel(10, 100*time.Millisecond)
	defer timeWheel.Stop()

	timeWheel.AddTask("test_now", func() {
		t.Logf("test_now, %v", time.Now())
	}, time.Now())

	timeWheel.AddTask("test1", func() {
		t.Logf("test1, %v", time.Now())
	}, time.Now().Add(time.Second))

	timeWheel.AddTask("test2", func() {
		t.Logf("test2, %v", time.Now())
	}, time.Now().Add(5*time.Second))

	timeWheel.AddTask("test2", func() {
		t.Logf("test2, %v", time.Now())
	}, time.Now().Add(3*time.Second))

	timeWheel.AddTask("test2Task", func() {
		t.Logf("test2Task, %v", time.Now())
	}, time.Now().Add(3*time.Second))

	timeWheel.AddTask("test_panic", func() {
		panic("test_panic")
	}, time.Now().Add(2*time.Second))

	<-time.After(6 * time.Second)
}
