package cli

import "testing"

func TestMatchAutoPrefix(t *testing.T) {
	m, err := newMatcher("auto", "mario")
	if err != nil {
		t.Fatalf("newMatcher error: %v", err)
	}
	ok, err := m.Match("Mario Kart 64")
	if err != nil || !ok {
		t.Fatalf("expected prefix match, ok=%v err=%v", ok, err)
	}
	ok, err = m.Match("Super Mario 64")
	if err != nil || ok {
		t.Fatalf("expected prefix to not match, ok=%v err=%v", ok, err)
	}
}

func TestMatchAutoGlob(t *testing.T) {
	m, err := newMatcher("auto", "*mario*")
	if err != nil {
		t.Fatalf("newMatcher error: %v", err)
	}
	ok, err := m.Match("Super Mario 64")
	if err != nil || !ok {
		t.Fatalf("expected glob match, ok=%v err=%v", ok, err)
	}
}

func TestMatchContains(t *testing.T) {
	m, err := newMatcher("contains", "mario")
	if err != nil {
		t.Fatalf("newMatcher error: %v", err)
	}
	ok, err := m.Match("Super Mario 64")
	if err != nil || !ok {
		t.Fatalf("expected contains match, ok=%v err=%v", ok, err)
	}
}

func TestMatchGlob(t *testing.T) {
	m, err := newMatcher("glob", "Mario *")
	if err != nil {
		t.Fatalf("newMatcher error: %v", err)
	}
	ok, err := m.Match("Mario Kart 64")
	if err != nil || !ok {
		t.Fatalf("expected glob match, ok=%v err=%v", ok, err)
	}
}

func TestMatchRegex(t *testing.T) {
	m, err := newMatcher("regex", "Mario\\s+Kart")
	if err != nil {
		t.Fatalf("newMatcher error: %v", err)
	}
	ok, err := m.Match("Mario Kart 64")
	if err != nil || !ok {
		t.Fatalf("expected regex match, ok=%v err=%v", ok, err)
	}
}

func TestQueryHint(t *testing.T) {
	if got := queryHint("*mario*", "auto"); got != "mario" {
		t.Fatalf("expected mario, got %q", got)
	}
	if got := queryHint("Mario Kart*", "auto"); got != "Mario Kart" {
		t.Fatalf("expected Mario Kart, got %q", got)
	}
	if got := queryHint("mario", "contains"); got != "mario" {
		t.Fatalf("expected mario, got %q", got)
	}
}
