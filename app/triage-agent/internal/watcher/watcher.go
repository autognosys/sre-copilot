// Package watcher watches Pod objects (not raw Events — Events are noisier
// and TTL out quickly) for status transitions matching known bad-deploy
// signatures, and calls a handler for each match.
package watcher

import (
	"context"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Incident is the extensible signature list. Add a new entry here to teach
// the watcher a new failure pattern to detect — this is the "known incident
// types" catalog referenced in the sre-copilot build log.
type Incident struct {
	Category string // matches the triage agent's root_cause_category vocabulary
	PodUID   string
	PodName  string
	Namespace string
	Reason   string // the K8s-reported reason, e.g. "OOMKilled", "CrashLoopBackOff"
	ExitCode int32
}

// Handler is called once per detected incident. Implementations should be
// fast and non-blocking — hand off to a goroutine/queue for the actual
// (slow, API-calling) triage work rather than doing it inline, so a single
// slow triage run doesn't stall the watch loop and cause missed events.
type Handler func(ctx context.Context, incident Incident)

type Watcher struct {
	clientset kubernetes.Interface
	namespace string
	handler   Handler
	log       *slog.Logger
}

func New(clientset kubernetes.Interface, namespace string, handler Handler, log *slog.Logger) *Watcher {
	return &Watcher{clientset: clientset, namespace: namespace, handler: handler, log: log}
}

// Run starts the informer and blocks until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	watchlist := cache.NewListWatchFromClient(
		w.clientset.CoreV1().RESTClient(),
		"pods",
		w.namespace,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		watchlist,
		&corev1.Pod{},
		0, // no periodic resync — we react to real update events only
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				pod, ok := newObj.(*corev1.Pod)
				if !ok {
					return
				}
				w.inspect(ctx, pod)
			},
		},
	)

	w.log.Info("watcher starting", "namespace", w.namespace)
	controller.Run(ctx.Done())
	return nil
}

// inspect checks a pod's container statuses against the known signature
// list. Extend this switch as new incident types get added to the catalog.
func (w *Watcher) inspect(ctx context.Context, pod *corev1.Pod) {
	for _, cs := range pod.Status.ContainerStatuses {
		term := cs.LastTerminationState.Terminated
		if term == nil {
			continue
		}

		switch term.Reason {
		case "OOMKilled":
			w.emit(ctx, pod, "oom_kill", term.Reason, term.ExitCode)
		default:
			// Not a signature we recognize yet. Left unhandled deliberately
			// rather than triaging everything — an unmatched termination
			// reason should be a prompt to add a new case here, not to
			// silently fall through to the LLM for every possible reason.
		}

		waiting := cs.State.Waiting
		if waiting != nil && waiting.Reason == "CrashLoopBackOff" {
			w.emit(ctx, pod, "crash_loop", waiting.Reason, term.ExitCode)
		}
	}
}

func (w *Watcher) emit(ctx context.Context, pod *corev1.Pod, category, reason string, exitCode int32) {
	incident := Incident{
		Category:  category,
		PodUID:    string(pod.UID),
		PodName:   pod.Name,
		Namespace: pod.Namespace,
		Reason:    reason,
		ExitCode:  exitCode,
	}
	w.log.Info("incident detected", "category", category, "pod", pod.Name, "reason", reason)
	w.handler(ctx, incident)
}
