package cmdutils

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/solo-io/go-utils/threadsafe"
)

var (
	_            Cmd   = &LocalCmd{}
	_            Cmder = &LocalCmder{}
	defaultCmder       = &LocalCmder{}
)

// Command is a convenience wrapper over defaultCmder.Command
func Command(ctx context.Context, command string, args ...string) Cmd {
	return defaultCmder.Command(ctx, command, args...).
		WithStdout(io.Discard).
		WithStderr(io.Discard)
}

// LocalCmder is a factory for LocalCmd, implementing Cmder
type LocalCmder struct{}

// Command returns a Cmd which includes the running process's `Environment`
func (c *LocalCmder) Command(ctx context.Context, name string, arg ...string) Cmd {
	var combinedOutput threadsafe.Buffer
	cmd := &LocalCmd{
		Cmd:            exec.CommandContext(ctx, name, arg...),
		combinedOutput: &combinedOutput,
	}

	// By default, assign the env variables for the command
	// Consumers of this Cmd can then override it, if they want
	return cmd.WithEnv(os.Environ()...)
}

// LocalCmd wraps os/exec.Cmd, implementing the cmdutils.Cmd interface
type LocalCmd struct {
	*exec.Cmd
	combinedOutput *threadsafe.Buffer
}

// WithEnv sets env
func (cmd *LocalCmd) WithEnv(env ...string) Cmd {
	//disable DEBUG=1 from getting through to command
	for i, pair := range env {
		if strings.HasPrefix(pair, "DEBUG") {
			env = append(env[:i], env[i+1:]...)
			break
		}
	}

	cmd.Env = env
	return cmd
}

// WithStdin sets stdin
func (cmd *LocalCmd) WithStdin(r io.Reader) Cmd {
	cmd.Stdin = r
	return cmd
}

// WithStdout set stdout
func (cmd *LocalCmd) WithStdout(w io.Writer) Cmd {
	cmd.Stdout = w
	return cmd
}

// WithStderr sets stderr
func (cmd *LocalCmd) WithStderr(w io.Writer) Cmd {
	cmd.Stderr = w
	return cmd
}

// Run runs the command
// If the returned error is non-nil, it should be of type *RunError
func (cmd *LocalCmd) Run() *RunError {
	var combinedOutput threadsafe.Buffer

	if printCommands {
		// Print to stderr to avoid interfering with stdout intended for parsing
		fmt.Fprintf(os.Stderr, "+ %s\n", PrettyCommand(false, cmd.Args[0], cmd.Args[1:]...))
	}

	// Debug logging for race investigation

	fmt.Fprintf(os.Stderr, "[DEBUG] Starting command execution in Run(): %s (buffer addr: %p)\n", cmd.Args[0], &combinedOutput)

	cmd.Stdout = io.MultiWriter(cmd.Stdout, &combinedOutput)
	cmd.Stderr = io.MultiWriter(cmd.Stderr, &combinedOutput)

	if err := cmd.Cmd.Run(); err != nil {
		return &RunError{
			command:    cmd.Args,
			output:     combinedOutput.Bytes(),
			inner:      err,
			stackTrace: err,
		}
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Completed command execution in Run(): %s\n", cmd.Args[0])

	return nil
}

// Start starts the command but doesn't block
// If the returned error is non-nil, it should be of type *RunError
func (cmd *LocalCmd) Start() *RunError {
	if printCommands {
		// Print to stderr to avoid interfering with stdout intended for parsing
		fmt.Fprintf(os.Stderr, "+ %s\n", PrettyCommand(false, cmd.Args[0], cmd.Args[1:]...))
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Starting command execution in Start(): %s (buffer addr: %p)\n", cmd.Args[0], &cmd.combinedOutput)

	cmd.Stdout = io.MultiWriter(cmd.Stdout, cmd.combinedOutput)
	cmd.Stderr = io.MultiWriter(cmd.Stderr, cmd.combinedOutput)

	if err := cmd.Cmd.Start(); err != nil {
		return &RunError{
			command:    cmd.Args,
			output:     cmd.combinedOutput.Bytes(),
			inner:      err,
			stackTrace: err,
		}
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Leaving command execution in Start(): %s (buffer addr: %p)\n", cmd.Args[0], &cmd.combinedOutput)
	return nil
}

// Wait waits for the command to finish
// If the returned error is non-nil, it should be of type *RunError
func (cmd *LocalCmd) Wait() *RunError {
	fmt.Fprintf(os.Stderr, "[DEBUG] Starting execution in Wait(): %s (buffer addr: %p)\n", cmd.Args[0], &cmd.combinedOutput)

	if err := cmd.Cmd.Wait(); err != nil {
		return &RunError{
			command:    cmd.Args,
			output:     cmd.combinedOutput.Bytes(),
			inner:      err,
			stackTrace: err,
		}
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Completed execution in Wait(): %s (buffer addr: %p)\n", cmd.Args[0], &cmd.combinedOutput)
	return nil
}

// Output returns the output of the command
// If the returned error is non-nil, it should be of type *RunError
func (cmd *LocalCmd) Output() []byte {
	return cmd.combinedOutput.Bytes()
}

func (cmd *LocalCmd) PrettyCommand() string {
	return PrettyCommand(true, cmd.Args[0], cmd.Args[1:]...)
}
