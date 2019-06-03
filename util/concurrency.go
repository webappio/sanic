package util

import (
	"context"
	"golang.org/x/sync/errgroup"
	"os/exec"
)

//RunContextuallyInParallel runs a set of goroutines which take a context and waits for them to terminate
//if any of the goroutines fail with an error, they will all immediately have their context cancelled and this method will return that error
func RunContextuallyInParallel(ctx context.Context, funcs ...func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	eg, ctx := errgroup.WithContext(ctx)
	errGroupWait := make(chan error)
	errGroupErrors := make(chan error)
	for _, f := range funcs {
		finalF := f
		eg.Go(func() error {
			err := finalF(ctx)
			errGroupErrors <- err
			return err
		})
	}
	go func() { errGroupWait <- eg.Wait() }()
	select {
	case err := <-errGroupWait:
		return err
	case err := <-errGroupErrors:
		return err
	}
}

//RunCmdContextually waits a given exec.Cmd "in" the given context.  There are two cases:
// 1. If the command finishes before the context is finished, the result of cmd.Run is returned
// 2. If the context is cancelled before the command finishes, the command's process is killed forcefully
//    and this method returns immediately.
func WaitCmdContextually(cmd *exec.Cmd, ctx context.Context) error {
	cmdDone := make(chan error)
	go func() { cmdDone <- cmd.Wait() }()
	select {
	case err := <-cmdDone:
		return err
	case <-ctx.Done():
		return cmd.Process.Kill()
	}
}
