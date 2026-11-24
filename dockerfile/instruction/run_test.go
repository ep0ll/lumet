package instruction

import (
	"strings"
	"testing"
)

func TestCommandArgsUsesPlatformDefaultShell(t *testing.T) {
	linux, err := commandArgs(Sh("echo ok"), nil, "linux")
	if err != nil {
		t.Fatal(err)
	}
	wantLinux := []string{"/bin/sh", "-c", "echo ok"}
	if !sameArgs(linux, wantLinux) {
		t.Fatalf("Linux shell args = %#v, want %#v", linux, wantLinux)
	}

	windows, err := commandArgs(Sh("echo ok"), nil, "windows")
	if err != nil {
		t.Fatal(err)
	}
	wantWindows := []string{"cmd", "/S", "/C", "echo ok"}
	if !sameArgs(windows, wantWindows) {
		t.Fatalf("Windows shell args = %#v, want %#v", windows, wantWindows)
	}
}

func TestConfiguredShellOverridesPlatformDefault(t *testing.T) {
	got, err := commandArgs(Sh("echo ok"), []string{"/bin/bash", "-e", "-c"}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/bin/bash", "-e", "-c", "echo ok"}
	if !sameArgs(got, want) {
		t.Fatalf("configured shell args = %#v, want %#v", got, want)
	}
}

func TestExecFormPreservesArguments(t *testing.T) {
	got, err := commandArgs(Exec("echo", "hello world", "$HOME"), nil, "linux")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"echo", "hello world", "$HOME"}
	if !sameArgs(got, want) {
		t.Fatalf("exec args = %#v, want %#v", got, want)
	}
}

func sameArgs(got, want []string) bool {
	return strings.Join(got, "\x00") == strings.Join(want, "\x00")
}
