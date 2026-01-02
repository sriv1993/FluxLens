package auditlog

import (
	"testing"
)

func TestAppendAndVerify(t *testing.T) {
	c := NewChain()
	for i := 0; i < 10; i++ {
		if _, err := c.Append("ingest", map[string]int{"i": i}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	if c.Len() != 10 {
		t.Fatalf("expected len 10 got %d", c.Len())
	}
	ok, err := c.Verify()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("chain should verify")
	}
}

func TestAppendRequiresKind(t *testing.T) {
	c := NewChain()
	if _, err := c.Append("", nil); err == nil {
		t.Fatal("expected error for empty kind")
	}
}

func TestSequenceMonotonic(t *testing.T) {
	c := NewChain()
	prev := uint64(0)
	for i := 0; i < 5; i++ {
		r, _ := c.Append("k", nil)
		if i > 0 && r.Sequence != prev+1 {
			t.Fatalf("sequence not monotonic: prev %d cur %d", prev, r.Sequence)
		}
		prev = r.Sequence
	}
}

func TestPrevHashLinks(t *testing.T) {
	c := NewChain()
	a, _ := c.Append("k", 1)
	b, _ := c.Append("k", 2)
	if b.PrevHash != a.Hash {
		t.Fatal("PrevHash does not link to previous record")
	}
}

func TestTamperDetected_PayloadMutation(t *testing.T) {
	c := NewChain()
	for i := 0; i < 5; i++ {
		_, _ = c.Append("k", map[string]int{"i": i})
	}
	c.mu.Lock()
	c.records[2].Payload = []byte(`{"tampered":true}`)
	c.mu.Unlock()
	ok, err := c.Verify()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Fatal("expected tamper detected")
	}
}

func TestTamperDetected_HashMutation(t *testing.T) {
	c := NewChain()
	for i := 0; i < 5; i++ {
		_, _ = c.Append("k", i)
	}
	c.mu.Lock()
	c.records[1].Hash = "deadbeef"
	c.mu.Unlock()
	ok, _ := c.Verify()
	if ok {
		t.Fatal("expected tamper detected via hash mutation")
	}
}

func TestTamperDetected_RecordRemoval(t *testing.T) {
	c := NewChain()
	for i := 0; i < 5; i++ {
		_, _ = c.Append("k", i)
	}
	c.mu.Lock()
	// Remove the third record; subsequent records' PrevHash will not match.
	c.records = append(c.records[:2], c.records[3:]...)
	c.mu.Unlock()
	ok, _ := c.Verify()
	if ok {
		t.Fatal("expected tamper detected via record removal")
	}
}

func TestEmptyChainVerifies(t *testing.T) {
	c := NewChain()
	ok, err := c.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("empty chain should verify")
	}
}

func TestSnapshotIsCopy(t *testing.T) {
	c := NewChain()
	_, _ = c.Append("k", 1)
	snap := c.Snapshot()
	if len(snap) != 1 {
		t.Fatal("snapshot length")
	}
	snap[0].Hash = "mutated"
	// Verify chain still verifies after caller mutation of snapshot copy.
	ok, _ := c.Verify()
	if !ok {
		t.Fatal("caller mutation of snapshot must not affect chain")
	}
}
