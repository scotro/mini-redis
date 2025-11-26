package hello

import "testing"

func TestWorld(t *testing.T) {
	got := World()
	want := "Hello, World!"
	if got != want {
		t.Errorf("World() = %q, want %q", got, want)
	}
}
