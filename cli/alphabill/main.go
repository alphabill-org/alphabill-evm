package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alphabill-org/alphabill/cli/alphabill/cmd"
	"github.com/alphabill-org/alphabill/observability"

	"github.com/alphabill-org/alphabill-evm/cli/alphabill/evm"
)

func main() {
	ctx := quitSignalContext()
	app := cmd.New(observability.NewFactory())
	if err := app.RegisterPartition(evm.NewEvmPartition()); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	err := app.Execute(ctx)
	if err != nil && !cancelledByQuitSignal(ctx) {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

var errQuitSignal = errors.New("received quit signal")

/*
quitSignalContext returns context.Context which will be cancelled (with cause errQuitSignal)
when one of the quit signals is sent to the program
*/
func quitSignalContext() context.Context {
	ctx, cancel := context.WithCancelCause(context.Background())

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)
		sig := <-sigChan
		cancel(fmt.Errorf("%s: %w", sig, errQuitSignal))
	}()

	return ctx
}

/*
cancelledByQuitSignal returns true when ctx has been cancelled with quit signal cause
*/
func cancelledByQuitSignal(ctx context.Context) bool {
	err := context.Cause(ctx)
	return err != nil && errors.Is(err, errQuitSignal)
}
