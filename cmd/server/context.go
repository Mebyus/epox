package main

import (
	"context"
	"os"
	"os/signal"
)

func newProcessContext() context.Context {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	return ctx
}
