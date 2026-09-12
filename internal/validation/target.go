// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package validation

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

// AgentTarget holds the routing-relevant fields of an active agent.
type AgentTarget struct {
	MachineID string
	Hostname  string
	State     string
	Labels    map[string]string
}

// AgentLister returns active agents with their hostnames and labels.
type AgentLister func(ctx context.Context) ([]AgentTarget, error)

var agentLister AgentLister

// labelSegmentRe matches NATS subject-safe segments (same as job.labelSegmentRegex).
var labelSegmentRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var (
	cacheMu      sync.Mutex
	cachedAgents []AgentTarget
	cacheExpiry  time.Time
	cacheTTL     = 5 * time.Second
)

// RegisterTargetValidator registers the valid_target custom validator and
// sets the AgentLister it uses. Call this at API server startup after the
// job client is created, or in test SetupSuite to inject a mock.
func RegisterTargetValidator(
	lister AgentLister,
) {
	// Cannot error: tag is non-empty and function is non-nil.
	_ = instance.RegisterValidation("valid_target", validTarget)
	agentLister = lister
	// Reset cache so the new lister is used immediately.
	cacheMu.Lock()
	cacheExpiry = time.Time{}
	cacheMu.Unlock()
}

// getAgents returns the cached agent list, refreshing from NATS when
// the cache has expired.
func getAgents() ([]AgentTarget, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if time.Now().Before(cacheExpiry) {
		return cachedAgents, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	agents, err := agentLister(ctx)
	if err != nil {
		return nil, err
	}

	cachedAgents = agents
	cacheExpiry = time.Now().Add(cacheTTL)

	return agents, nil
}

// validTarget checks whether the target is a valid routing pattern
// (_any, _all), a label matching an active agent, or a direct hostname.
func validTarget(
	fl validator.FieldLevel,
) bool {
	target := fl.Field().String()

	if target == "_any" || target == "_all" {
		return !allAgentsPending()
	}

	if agentLister == nil {
		return false
	}

	// Label target: validate format then check against active agents.
	if i := strings.IndexByte(target, ':'); i > 0 && i < len(target)-1 {
		key := target[:i]
		value := target[i+1:]
		if !isValidLabelFormat(key, value) {
			return false
		}
		return matchesLabel(key, value)
	}

	// Direct hostname target: check against active agents.
	return matchesHostname(target)
}

// isValidLabelFormat validates that label key and value segments are
// NATS subject-safe.
func isValidLabelFormat(
	key, value string,
) bool {
	if !labelSegmentRe.MatchString(key) {
		return false
	}
	for _, seg := range strings.Split(value, ".") {
		if !labelSegmentRe.MatchString(seg) {
			return false
		}
	}
	return true
}

// matchesLabel checks whether any active agent has a label whose key
// matches and whose value is a prefix match (hierarchical). For example,
// target "group:web" matches an agent with label group=web.dev.us-east.
func matchesLabel(
	key, value string,
) bool {
	agents, err := getAgents()
	if err != nil {
		return false
	}

	for _, a := range agents {
		if av, ok := a.Labels[key]; ok {
			if av == value || strings.HasPrefix(av, value+".") {
				return true
			}
		}
	}

	return false
}

// allAgentsPending returns true when every registered agent is in Pending
// state. Used to fail fast on _any/_all targets instead of timing out.
// Returns false when no agents are registered or the lister is unavailable.
func allAgentsPending() bool {
	if agentLister == nil {
		return false
	}

	agents, err := getAgents()
	if err != nil || len(agents) == 0 {
		return false
	}

	for _, a := range agents {
		if a.State != "Pending" {
			return false
		}
	}

	pendingTargetMu.Lock()
	pendingTarget = "_all"
	pendingTargetMu.Unlock()

	return true
}

// pendingTarget is set when a target matches a pending agent.
// Used by the error message formatter to provide a specific message.
var (
	pendingTargetMu sync.Mutex
	pendingTarget   string
)

// matchesHostname checks whether any active agent has the given hostname
// or machine ID. Returns false for agents in Pending state (awaiting
// PKI enrollment).
func matchesHostname(
	target string,
) bool {
	agents, err := getAgents()
	if err != nil {
		return false
	}

	for _, a := range agents {
		if a.Hostname == target || a.MachineID == target {
			if a.State == "Pending" {
				pendingTargetMu.Lock()
				pendingTarget = target
				pendingTargetMu.Unlock()

				return false
			}

			return true
		}
	}

	return false
}

// IsPendingTarget returns true if the last validation failure was
// due to a pending agent, and clears the flag.
func IsPendingTarget() (string, bool) {
	pendingTargetMu.Lock()
	defer pendingTargetMu.Unlock()

	t := pendingTarget
	pendingTarget = ""

	return t, t != ""
}

// ResolveTarget resolves a target string to a machine ID for NATS subject
// routing. If the target matches a hostname, the corresponding machine ID
// is returned. If it matches a machine ID directly, it is returned as-is.
// Broadcast and label targets are returned unchanged.
func ResolveTarget(
	target string,
) string {
	if target == "_any" || target == "_all" {
		return target
	}

	if strings.ContainsRune(target, ':') {
		return target
	}

	agents, err := getAgents()
	if err != nil {
		return target
	}

	// Check if target is a hostname → resolve to machine ID.
	for _, a := range agents {
		if a.Hostname == target {
			return a.MachineID
		}
	}

	// Check if target is already a machine ID.
	for _, a := range agents {
		if a.MachineID == target {
			return target
		}
	}

	return target
}
