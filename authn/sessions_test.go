package authn

import (
	"testing"
)

func TestSessionManager(t *testing.T) {

	sm := NewSessionManager(nil, true, nil)

	stats := sm.Statistics()
	if stats != (SessionStatistics{0, 0, 0}) {
		t.Error()
	}
	s11, n := sm.getSession("1")
	sm.watch(0, 0)
	stats = sm.Statistics()
	if stats != (SessionStatistics{1, 1, 1}) || !n {
		t.Error()
	}
	s12, n := sm.getSession("1")
	sm.watch(0, 0)
	stats = sm.Statistics()
	if stats != (SessionStatistics{2, 2, 1}) || !n {
		t.Error()
	}
	s21, n := sm.getSession("2")
	sm.watch(0, 0)
	stats = sm.Statistics()
	if stats != (SessionStatistics{3, 3, 2}) || !n {
		t.Error()
	}
	sm.leaveSession(s11)
	sm.watch(0, 0)
	stats = sm.Statistics()
	if stats != (SessionStatistics{2, 2, 2}) {
		t.Error()
	}
	s13, n := sm.getSession("1")
	sm.watch(0, 0)
	stats = sm.Statistics()
	if stats != (SessionStatistics{3, 3, 2}) || !n {
		t.Error()
	}
	sm.leaveSession(s12)
	sm.watch(0, 0)
	stats = sm.Statistics()
	if stats != (SessionStatistics{2, 2, 2}) {
		t.Error()
	}
	sm.leaveSession(s21)
	sm.watch(0, 0)
	stats = sm.Statistics()
	if stats != (SessionStatistics{1, 1, 1}) {
		t.Error()
	}
	sm.leaveSession(s13)
	sm.watch(0, 0)
	stats = sm.Statistics()
	if stats != (SessionStatistics{0, 0, 0}) {
		t.Error()
	}
}
