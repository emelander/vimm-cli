package vault

import "testing"

func TestFindRomNameInLairIgnoresURL(t *testing.T) {
	data := []byte("vimm.net\nTest Game (USA).bin\nCRC: 1234abcd\n")
	candidates := []string{"Test Game (USA).bin"}
	got := FindRomNameInLair(data, candidates)
	if got != "Test Game (USA).bin" {
		t.Fatalf("expected rom name to match candidate, got %q", got)
	}
}
