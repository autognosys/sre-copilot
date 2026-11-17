// Package costguard is the agent's budget circuit breaker. It exists because
// this agent runs long-term, unattended, watching a live cluster — a bug
// causing repeated re-triage of the same flapping pod (or any other runaway
// loop) needs a hard backstop, not just "hope the code is correct."
//
// Two independent controls live here:
//  1. Cooldown: don't re-triage the same pod (by UID) more than once per
//     window, regardless of how many times it restarts.
//  2. Monthly budget: track cumulative estimated spend; once the configured
//     ceiling is crossed, refuse further Claude calls until the tracker
//     resets (see Reset / the monthly rollover in main.go).
//
// State is in-memory only for v1 — the process restarting resets both the
// cooldown map and the spend counter. Acceptable for a single-replica agent;
// worth moving to a ConfigMap or small on-disk store if that ever matters
// (e.g. spend tracking surviving a pod restart mid-month).
package costguard

import (
	"fmt"
	"sync"
	"time"
)

// Pricing is USD per million tokens. Update if Anthropic's published rates
// change — see https://docs.claude.com/en/docs/about-claude/pricing.
type ModelPricing struct {
	InputPerMTok  float64
	OutputPerMTok float64
}

var KnownPricing = map[string]ModelPricing{
	"claude-haiku-4-5-20251001": {InputPerMTok: 1.00, OutputPerMTok: 5.00},
	"claude-sonnet-5":           {InputPerMTok: 2.00, OutputPerMTok: 10.00}, // promo rate through Aug 2026; recheck after
}

type Guard struct {
	mu sync.Mutex

	// cooldown
	cooldownWindow time.Duration
	lastTriaged    map[string]time.Time // key: pod UID

	// budget
	monthlyBudgetUSD float64
	spentUSD         float64
	periodStart      time.Time
}

func New(cooldownWindow time.Duration, monthlyBudgetUSD float64) *Guard {
	return &Guard{
		cooldownWindow:   cooldownWindow,
		lastTriaged:      make(map[string]time.Time),
		monthlyBudgetUSD: monthlyBudgetUSD,
		periodStart:      time.Now(),
	}
}

// ShouldTriage reports whether triage should proceed for this pod right now.
// It checks cooldown only — budget is checked separately at call time via
// AllowSpend, since we don't know the cost of a run until we've made it.
func (g *Guard) ShouldTriage(podUID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	last, seen := g.lastTriaged[podUID]
	if seen && time.Since(last) < g.cooldownWindow {
		return false
	}
	g.lastTriaged[podUID] = time.Now()
	return true
}

// AllowSpend reports whether the agent is still under budget for this
// billing period. Call this BEFORE each Claude API call, not just once per
// incident — a multi-round tool_use loop should stop mid-incident if it
// blows through the budget, not just refuse to start the next incident.
func (g *Guard) AllowSpend() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rolloverIfNewMonthLocked()
	return g.spentUSD < g.monthlyBudgetUSD
}

// RecordSpend logs actual token usage from a completed API call against the
// running total. inputTokens/outputTokens come straight off the API
// response's Usage field — never estimated ahead of time.
func (g *Guard) RecordSpend(model string, inputTokens, outputTokens int) (costUSD float64, err error) {
	pricing, ok := KnownPricing[model]
	if !ok {
		return 0, fmt.Errorf("costguard: unknown model %q — add it to KnownPricing before using it, so spend is never silently untracked", model)
	}

	cost := (float64(inputTokens)/1_000_000)*pricing.InputPerMTok +
		(float64(outputTokens)/1_000_000)*pricing.OutputPerMTok

	g.mu.Lock()
	defer g.mu.Unlock()
	g.rolloverIfNewMonthLocked()
	g.spentUSD += cost
	return cost, nil
}

// SpentThisPeriod returns the running total for the current billing period.
func (g *Guard) SpentThisPeriod() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.spentUSD
}

func (g *Guard) rolloverIfNewMonthLocked() {
	now := time.Now()
	if now.Year() != g.periodStart.Year() || now.Month() != g.periodStart.Month() {
		g.spentUSD = 0
		g.periodStart = now
	}
}
