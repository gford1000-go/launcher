package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestLauncherWithNilCtx(t *testing.T) {

	_, err := New(nil, "sh", []string{}, "-c", "cat", "<<!")
	if err != errMissingContext {
		t.Fatal(err)
	}
}

func TestLauncherWithInvalidFile(t *testing.T) {

	_, err := New(context.TODO(), "zzzUnknownzzz", []string{})
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestLauncherNew(t *testing.T) {

	file := "sh"
	env := []string{"XYZ=ABC"}
	args := []string{"-c", "cat", "<<!"}

	l, err := New(context.TODO(), "sh", env, args...)
	if err != nil {
		t.Fatal(err)
	}

	hasher := func(arr []string) []byte {
		h := sha256.New()
		for _, a := range arr {
			h.Write([]byte(a))
		}
		return h.Sum(nil)
	}

	if l.GetFile() != file {
		t.Fatalf("mismatch in file: expected %v, got %v\n", file, l.GetFile())
	}

	if !bytes.Equal(hasher(l.GetEnv()), hasher(env)) {
		t.Fatalf("mismatch in args: expected %v, got %v\n", env, l.GetEnv())
	}

	if !bytes.Equal(hasher(l.GetArgs()), hasher(args)) {
		t.Fatalf("mismatch in args: expected %v, got %v\n", args, l.GetArgs())
	}

	if l.IsStarted() {
		t.Fatal("incorrectly saying has started")
	}

	if l.IsRunning() {
		t.Fatal("incorrectly saying is running")
	}
}

func TestLauncherWithCtxCancelBeforeNew(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New(ctx, "sh", []string{}, "-c", "cat", "<<!")
	if err != context.Canceled {
		t.Fatal(err)
	}
}

func TestLauncherWithCtxCancelAfterNew(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	l, err := New(ctx, "sh", []string{}, "-c", "cat", "<<!")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	cancel()

	err = l.Start()
	if err != context.Canceled {
		t.Fatal(err)
	}
}

func TestLauncherStdOut(t *testing.T) {

	l, err := New(context.Background(), "date", []string{})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if err := l.Start(); err != nil {
		t.Fatal(err)
	}

	var b = make([]byte, 100)
	_, err = l.cmdStdOut.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	l.Cancel()

	if l.IsRunning() {
		t.Fatal("still running")
	}

	// Should return a date, check for year
	s := strings.Split(string(b), "\n")
	if s[0][len(s[0])-4:] != fmt.Sprintf("%v", time.Now().Year()) {
		t.Fatalf("%q\n", s)
	}
}

func TestLauncherWithArg(t *testing.T) {

	foo := "foo"

	l, err := New(context.Background(), "echo", []string{}, foo)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if err := l.Start(); err != nil {
		t.Fatal(err)
	}

	var b = make([]byte, len(foo))
	_, err = l.cmdStdOut.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	l.Cancel()

	if l.IsRunning() {
		t.Fatal("still running")
	}

	if string(b) != foo {
		t.Fatalf("invalid response - expected %q, got %q\n", foo, string(b))
	}
}

func TestLauncherWithArgAndEnv(t *testing.T) {

	foo := "foo"

	l, err := New(context.Background(), "echo", []string{"XYZ=ABC"}, foo)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if err := l.Start(); err != nil {
		t.Fatal(err)
	}

	var b = make([]byte, len(foo))
	_, err = l.cmdStdOut.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	l.Cancel()

	if l.IsRunning() {
		t.Fatal("still running")
	}

	if string(b) != foo {
		t.Fatalf("invalid response - expected %q, got %q\n", foo, string(b))
	}
}

func TestLauncherWithStdIn(t *testing.T) {

	foo := "foo"
	bar := "bar"

	l, err := New(context.Background(), "sh", []string{}, "-c", "cat", "<<!")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if err := l.Start(); err != nil {
		t.Fatal(err)
	}

	for _, s := range []string{foo, " ", bar, "\\!"} {
		err = l.SendStdIn([]byte(s))
		if err != nil {
			t.Fatal(err)
		}
	}

	expected_result := fmt.Sprintf("%v %v", foo, bar)

	var b = make([]byte, len(expected_result))
	_, err = l.cmdStdOut.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	l.Cancel()

	if l.IsRunning() {
		t.Fatal("Still running")
	}

	if string(b) != expected_result {
		t.Fatalf("invalid response - expected %q, got %q\n", expected_result, string(b))
	}
}

func TestLauncherRunWithArg(t *testing.T) {

	foo := "foo"

	l, err := New(context.Background(), "echo", []string{}, foo)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if err := l.Run(); err != nil {
		t.Fatal(err)
	}

	if l.IsRunning() {
		t.Fatal("still running")
	}
}

func TestLauncherRunWithCtxCancelAfterNew(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	foo := "foo"

	l, err := New(ctx, "echo", []string{}, foo)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	cancel()
	err = l.Run()
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}

}
