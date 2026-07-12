package database

import (
	"context"
	"testing"
	"time"
)

// Stop must interrupt the listener even while it is blocked in
// WaitForNotification, and return only once the loop has exited and its
// dedicated connection is closed. It used to leave both behind: the stop
// channel was only checked between notifications.
func TestListenerStopInterruptsWait(t *testing.T) {
	l := NewNotificationListener(dbe.config.URL, nil, nil)
	if err := l.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Let the listener connect and block waiting for notifications
	time.Sleep(200 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		l.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return: listener stuck in WaitForNotification")
	}

	// The listener must be restartable after a stop
	if err := l.Start(context.Background()); err != nil {
		t.Fatalf("restart after Stop failed: %v", err)
	}
	l.Stop()
}
