package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/autognosys/sre-copilot/triage-agent/internal/costguard"
	"github.com/autognosys/sre-copilot/triage-agent/internal/watcher"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	namespace := envOr("WATCH_NAMESPACE", "default")
	cooldown := envDurationOr("TRIAGE_COOLDOWN", 15*time.Minute)
	monthlyBudget := envFloatOr("MONTHLY_BUDGET_USD", 20.00)

	clientset, err := buildClientset()
	if err != nil {
		log.Error("failed to build k8s client", "err", err)
		os.Exit(1)
	}

	guard := costguard.New(cooldown, monthlyBudget)
	log.Info("cost guard configured", "cooldown", cooldown, "monthly_budget_usd", monthlyBudget)

	handler := func(ctx context.Context, incident watcher.Incident) {
		if !guard.ShouldTriage(incident.PodUID) {
			log.Info("skipping — pod in cooldown", "pod", incident.PodName)
			return
		}

		// TODO(next pass): wire this to the Claude tool_use loop. For now,
		// just confirm detection + cooldown are working correctly before
		// spending any API budget on the reasoning step.
		log.Info("would triage now (Claude reasoning not yet wired up)",
			"category", incident.Category,
			"pod", incident.PodName,
			"namespace", incident.Namespace,
			"reason", incident.Reason,
			"exit_code", incident.ExitCode,
			"spent_this_period_usd", guard.SpentThisPeriod(),
		)
	}

	w := watcher.New(clientset, namespace, handler, log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := w.Run(ctx); err != nil {
		log.Error("watcher exited with error", "err", err)
		os.Exit(1)
	}
}

// buildClientset tries in-cluster config first (the normal path when running
// as a Deployment), falling back to $KUBECONFIG for local development on
// symphony against the real cluster.
func buildClientset() (kubernetes.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(cfg)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envFloatOr(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f
		}
	}
	return fallback
}
