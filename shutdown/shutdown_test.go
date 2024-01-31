package shutdown

import (
	"testing"
)

func init() {
	sendSignal = false
}

func TestOrder(t *testing.T) {
	id := 0

	Add(func() error {
		id++
		t.Logf("id: %d", id)
		if id != 1 {
			t.Error("Expected id to be 1")
		}
		return nil
	})

	Add(func() error {
		id++
		t.Logf("id: %d", id)
		if id != 2 {
			t.Error("Expected id to be 2")
		}
		return nil
	})

	Shutdown()

	if id != 2 {
		t.Error("Expected id to be 2")
	}
}

func TestRemove(t *testing.T) {
	id := 0

	firstUlid := Add(func() error {
		id++
		t.Logf("id: %d", id)
		if id != 1 {
			t.Error("Expected id to be 1")
		}
		return nil
	})

	Remove(firstUlid)

	Add(func() error {
		id++
		t.Logf("id: %d", id)
		if id != 1 {
			t.Error("Expected id to be 1")
		}
		return nil
	})

	Shutdown()

	if id != 1 {
		t.Error("Expected id to be 1")
	}
}

func TestPriority(t *testing.T) {
	id := 0

	Add(func() error {
		id++
		t.Logf("id: %d", id)
		if id != 4 {
			t.Error("Expected id to be 4")
		}
		return nil
	})

	AddP(1, func() error {
		id++
		t.Logf("id: %d", id)
		if id != 2 {
			t.Error("Expected id to be 2")
		}
		return nil
	})

	AddP(2, func() error {
		id++
		t.Logf("id: %d", id)
		if id != 3 {
			t.Error("Expected id to be 3")
		}
		return nil
	})

	AddP(0, func() error {
		id++
		t.Logf("id: %d", id)
		if id != 1 {
			t.Error("Expected id to be 1")
		}
		return nil
	})

	Shutdown()

	if id != 4 {
		t.Error("Expected id to be 4")
	}
}
