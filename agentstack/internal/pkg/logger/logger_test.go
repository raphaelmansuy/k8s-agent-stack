package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	log, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = log.Sync() }()

	if log == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestNewNop(t *testing.T) {
	log := NewNop()
	if log == nil {
		t.Fatal("NewNop() returned nil logger")
	}
}
