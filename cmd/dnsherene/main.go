package main

import (
	"context"
	"dnsherene/internal/config"
	"dnsherene/internal/notification"
	"dnsherene/internal/output"
	"dnsherene/internal/runner"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// main 加载配置并执行一次续期任务。
func main() {
	cfg, err := config.Load()
	if err != nil {
		output.WritePublicErrorReport(os.Stderr, err)
		os.Exit(1)
	}

	notifier, err := notification.NewManager(cfg.Notification)
	if err != nil {
		output.WritePublicErrorReport(os.Stderr, err)
		os.Exit(1)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithTimeout(signalCtx, cfg.RunTimeout)
	defer cancel()

	info, err := runner.Execute(ctx, cfg)
	if notifyErr := notifier.Notify(ctx, info); notifyErr != nil {
		output.WritePrefixedPublicErrorReport(os.Stderr, "notification_error", notifyErr)
	}
	if err != nil {
		output.WritePublicErrorReport(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("renewed_total=%d\n", info.RenewedTotal)
}
