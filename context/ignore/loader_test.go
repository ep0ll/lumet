package ignore

import (
	"reflect"
	"testing"
)

func TestCleanPaths(t *testing.T) {
	got, err := cleanPaths([]string{
		".dockerignore",
		"src/../.dexnore",
		".dexnore",
		".dockerignore",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".dockerignore", ".dexnore"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanPaths() = %#v, want %#v", got, want)
	}
}

func TestCleanPathsRejectsTraversal(t *testing.T) {
	if _, err := cleanPaths([]string{"../outside"}); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}
