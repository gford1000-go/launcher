package launcher

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

var errMissingContext = errors.New("context must be provided")
var errIncompleteStdIntransfer = errors.New("command did not receive all bytes sent to stdin")

// New creates a new instance of Launcher, initialising but not launching
// the requested file as a child process.
func New(ctx context.Context, file string, env []string, arg ...string) (*Launcher, error) {
	if ctx == nil {
		return nil, errMissingContext
	}
	myCtx, cancel := context.WithCancel(ctx)

	path, err := exec.LookPath(file)
	if err != nil {
		cancel()
		return nil, err
	}

	l := &Launcher{
		file:   file,
		path:   path,
		ctx:    myCtx,
		cancel: cancel,
	}

	if err := l.initialise(env, arg...); err != nil {
		return nil, err
	}

	return l, nil
}

// Launcher wraps exec.Cmd behaviours
type Launcher struct {
	file      string
	path      string
	ctx       context.Context
	cancel    context.CancelFunc
	cmd       *exec.Cmd
	cmdWriter io.WriteCloser
	cmdStdOut io.ReadCloser
	cmdStdErr io.ReadCloser
}

// GetFile returns the requested file details
func (l *Launcher) GetFile() string {
	return l.file
}

// GetPath returns the fully qualified path identified for the file
func (l *Launcher) GetPath() string {
	return l.path
}

// GetArgs returns the arguments supplied to create the instance
func (l *Launcher) GetArgs() []string {
	return l.copyStringArray(l.cmd.Args[1:])
}

// GetEnv returns the environment supplied to create the instance
func (l *Launcher) GetEnv() []string {
	return l.copyStringArray(l.cmd.Env)
}

// IsStarted returns true if Start() has been called successfully
func (l *Launcher) IsStarted() bool {
	return l.cmd.Process != nil
}

// IsRunning returns true if the underlying process has started
// and has not exited in some way
func (l *Launcher) IsRunning() bool {
	select {
	case <-l.ctx.Done():
		return false
	default:
		return l.IsStarted() && (l.cmd.ProcessState == nil)
	}
}

// Close should be called to release all resources
func (l *Launcher) Close() error {
	var err error

	// Cancel the context for this instance
	l.cancel()

	// Close pipe
	if l.cmdWriter != nil {
		err = l.cmdWriter.Close()
	}
	return err
}

// copyStringArray replicates a string array
func (l *Launcher) copyStringArray(s []string) []string {
	r := []string{}
	r = append(r, s...)
	return r
}

// initialise prepares the process identified by LookPath for the file,
// wiring up Stdin, Stdout and Stderr
func (l *Launcher) initialise(env []string, arg ...string) error {
	select {
	case <-l.ctx.Done():
		return l.ctx.Err()
	default:
	}

	l.cmd = exec.CommandContext(l.ctx, l.path, l.copyStringArray(arg)...)
	l.cmd.Env = l.copyStringArray(env)

	pw, err := l.cmd.StdinPipe()
	if err != nil {
		return err
	}
	l.cmdWriter = pw

	pr, err := l.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	l.cmdStdOut = pr

	pr, err = l.cmd.StderrPipe()
	if err != nil {
		return err
	}
	l.cmdStdErr = pr

	return nil
}

// Start attempts to launch the underlying process
func (l *Launcher) Start() error {
	select {
	case <-l.ctx.Done():
		return l.ctx.Err()
	default:
	}
	return l.cmd.Start()
}

// Run attempts to launch the underlying process
// and waits until it completes
func (l *Launcher) Run() error {
	select {
	case <-l.ctx.Done():
		return l.ctx.Err()
	default:
	}
	return l.cmd.Run()
}

// Cancel ends processing
func (l *Launcher) Cancel() {
	l.cancel()
}

// SendStdIn passes the supplied bytes to the stdin of the
// underlying process, provided it is still running
func (l *Launcher) SendStdIn(b []byte) error {
	n, err := l.cmdWriter.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return errIncompleteStdIntransfer
	}
	return nil
}
