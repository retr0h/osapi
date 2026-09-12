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

package agent

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"time"

	"github.com/avfs/avfs"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/osapi-io/osapi/internal/agent/identity"
	"github.com/osapi-io/osapi/internal/agent/pki"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/provider/command"
	dockerProv "github.com/osapi-io/osapi/internal/provider/container/docker"
	fileProv "github.com/osapi-io/osapi/internal/provider/file"
	"github.com/osapi-io/osapi/internal/provider/network/netinfo"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/dns"
	"github.com/osapi-io/osapi/internal/provider/network/ping"
	diskProv "github.com/osapi-io/osapi/internal/provider/node/disk"
	nodeHost "github.com/osapi-io/osapi/internal/provider/node/host"
	"github.com/osapi-io/osapi/internal/provider/node/load"
	memProv "github.com/osapi-io/osapi/internal/provider/node/mem"
	"github.com/osapi-io/osapi/internal/provider/node/ntp"
	"github.com/osapi-io/osapi/internal/provider/node/power"
	"github.com/osapi-io/osapi/internal/provider/node/sysctl"
	"github.com/osapi-io/osapi/internal/provider/node/timezone"
)

// SetEmbeddedFS overrides the embedded filesystem for testing.
func SetEmbeddedFS(
	f fs.FS,
) {
	embeddedFS = f
}

// ResetEmbeddedFS restores the default embedded filesystem.
func ResetEmbeddedFS() {
	embeddedFS = systemTemplates
}

// SetReadEmbeddedFile overrides the read function for testing.
func SetReadEmbeddedFile(
	fn func(string) ([]byte, error),
) {
	readEmbeddedFile = fn
}

// ResetReadEmbeddedFile restores the default read function.
func ResetReadEmbeddedFile() {
	readEmbeddedFile = func(path string) ([]byte, error) {
		return systemTemplates.ReadFile(path)
	}
}

// ExportProcessJobOperation exposes the private processJobOperation method for testing.
func ExportProcessJobOperation(
	a *Agent,
	req job.Request,
) (json.RawMessage, error) {
	return a.processJobOperation(req)
}

// ExportProcessNodeOperation invokes the "node" processor from the agent's
// registry directly, for testing node-specific dispatch.
func ExportProcessNodeOperation(
	a *Agent,
	req job.Request,
) (json.RawMessage, error) {
	req.Category = "node"
	return a.registry.Dispatch(req)
}

// ExportProcessNetworkOperation invokes the "network" processor from the
// agent's registry directly, for testing network-specific dispatch.
func ExportProcessNetworkOperation(
	a *Agent,
	req job.Request,
) (json.RawMessage, error) {
	req.Category = "network"
	return a.registry.Dispatch(req)
}

// ExportWriteStatusEvent exposes the private writeStatusEvent method for testing.
func ExportWriteStatusEvent(
	ctx context.Context,
	a *Agent,
	jobID string,
	event string,
	data map[string]interface{},
) error {
	return a.writeStatusEvent(ctx, jobID, event, data)
}

// ExportHandleJobMessage exposes the private handleJobMessage method for testing.
func ExportHandleJobMessage(
	a *Agent,
	msg jetstream.Msg,
) error {
	return a.handleJobMessage(msg)
}

// ExportExtractChanged exposes the private extractChanged function for testing.
func ExportExtractChanged(
	data json.RawMessage,
) *bool {
	return extractChanged(data)
}

// ExportConsumeQueryJobs exposes the private consumeQueryJobs method for testing.
func ExportConsumeQueryJobs(
	ctx context.Context,
	a *Agent,
	hostname string,
) error {
	return a.consumeQueryJobs(ctx, hostname)
}

// ExportConsumeModifyJobs exposes the private consumeModifyJobs method for testing.
func ExportConsumeModifyJobs(
	ctx context.Context,
	a *Agent,
	hostname string,
) error {
	return a.consumeModifyJobs(ctx, hostname)
}

// ExportCreateConsumer exposes the private createConsumer method for testing.
func ExportCreateConsumer(
	ctx context.Context,
	a *Agent,
	streamName string,
	consumerName string,
	filterSubject string,
) error {
	return a.createConsumer(ctx, streamName, consumerName, filterSubject)
}

// ExportConsumerNamePrefix exposes the private consumerNamePrefix method for testing.
func ExportConsumerNamePrefix(
	a *Agent,
) string {
	return a.consumerNamePrefix()
}

// ExportHandleJobMessageJS exposes the private handleJobMessageJS method for testing.
func ExportHandleJobMessageJS(
	a *Agent,
	msg jetstream.Msg,
) error {
	return a.handleJobMessageJS(msg)
}

// ExportCheckDrainFlag exposes the private checkDrainFlag method for testing.
func ExportCheckDrainFlag(
	ctx context.Context,
	a *Agent,
	machineID string,
) bool {
	return a.checkDrainFlag(ctx, machineID)
}

// ExportHandleDrainDetection exposes the private handleDrainDetection method for testing.
func ExportHandleDrainDetection(
	ctx context.Context,
	a *Agent,
	machineID string,
	hostname string,
) {
	a.handleDrainDetection(ctx, machineID, hostname)
}

// ExportWriteFacts exposes the private writeFacts method for testing.
func ExportWriteFacts(
	ctx context.Context,
	a *Agent,
	machineID string,
	hostname string,
) {
	a.writeFacts(ctx, machineID, hostname)
}

// ExportStartFacts exposes the private startFacts method for testing.
func ExportStartFacts(
	ctx context.Context,
	a *Agent,
	machineID string,
	hostname string,
) {
	a.startFacts(ctx, machineID, hostname)
}

// ExportFactsKey exposes the private factsKey function for testing.
func ExportFactsKey(
	machineID string,
) string {
	return factsKey(machineID)
}

// ExportWriteRegistration exposes the private writeRegistration method for testing.
func ExportWriteRegistration(
	ctx context.Context,
	a *Agent,
	machineID string,
	hostname string,
) {
	a.writeRegistration(ctx, machineID, hostname)
}

// ExportDeregister exposes the private deregister method for testing.
func ExportDeregister(
	a *Agent,
	machineID string,
) {
	a.deregister(machineID)
}

// ExportStartHeartbeat exposes the private startHeartbeat method for testing.
func ExportStartHeartbeat(
	ctx context.Context,
	a *Agent,
	machineID string,
	hostname string,
) {
	a.startHeartbeat(ctx, machineID, hostname)
}

// ExportRegistryKey exposes the private registryKey function for testing.
func ExportRegistryKey(
	machineID string,
) string {
	return registryKey(machineID)
}

// SetAgentMachineID sets the agent's machineID field for testing.
func SetAgentMachineID(
	a *Agent,
	machineID string,
) {
	a.machineID = machineID
}

// GetAgentMachineID returns the agent's machineID field for testing.
func GetAgentMachineID(
	a *Agent,
) string {
	return a.machineID
}

// ExportFindPrevCondition exposes the private findPrevCondition function for testing.
func ExportFindPrevCondition(
	condType string,
	prev []job.Condition,
) *job.Condition {
	return findPrevCondition(condType, prev)
}

// ExportTransitionTime exposes the private transitionTime function for testing.
func ExportTransitionTime(
	condType string,
	newStatus bool,
	prev []job.Condition,
) time.Time {
	return transitionTime(condType, newStatus, prev)
}

// ExportEvaluateMemoryPressure exposes the private evaluateMemoryPressure function for testing.
func ExportEvaluateMemoryPressure(
	stats *memProv.Result,
	threshold int,
	prev []job.Condition,
) job.Condition {
	return evaluateMemoryPressure(stats, threshold, prev)
}

// ExportEvaluateHighLoad exposes the private evaluateHighLoad function for testing.
func ExportEvaluateHighLoad(
	loadAvg *load.Result,
	cpuCount int,
	multiplier float64,
	prev []job.Condition,
) job.Condition {
	return evaluateHighLoad(loadAvg, cpuCount, multiplier, prev)
}

// ExportEvaluateDiskPressure exposes the private evaluateDiskPressure function for testing.
func ExportEvaluateDiskPressure(
	disks []diskProv.Result,
	threshold int,
	prev []job.Condition,
) job.Condition {
	return evaluateDiskPressure(disks, threshold, prev)
}

// --- Package-level variable accessors for testing ---

// SetMarshalJSON overrides the marshalJSON function for testing.
func SetMarshalJSON(
	fn func(interface{}) ([]byte, error),
) {
	marshalJSON = fn
}

// ResetMarshalJSON restores the default marshalJSON function.
func ResetMarshalJSON() {
	marshalJSON = json.Marshal
}

// SetUnmarshalJSON overrides the unmarshalJSON function for testing.
func SetUnmarshalJSON(
	fn func([]byte, interface{}) error,
) {
	unmarshalJSON = fn
}

// ResetUnmarshalJSON restores the default unmarshalJSON function.
func ResetUnmarshalJSON() {
	unmarshalJSON = json.Unmarshal
}

// SetDefaultFactsInterval overrides the defaultFactsInterval for testing.
func SetDefaultFactsInterval(
	d time.Duration,
) {
	defaultFactsInterval = d
}

// ResetDefaultFactsInterval restores the default defaultFactsInterval.
func ResetDefaultFactsInterval() {
	defaultFactsInterval = 60 * time.Second
}

// SetHeartbeatInterval overrides the heartbeatInterval for testing.
func SetHeartbeatInterval(
	d time.Duration,
) {
	heartbeatInterval = d
}

// ResetHeartbeatInterval restores the default heartbeatInterval.
func ResetHeartbeatInterval() {
	heartbeatInterval = 10 * time.Second
}

// SetGetAgentHostnameFn overrides the getAgentHostnameFn for testing.
func SetGetAgentHostnameFn(
	fn func(string) (string, error),
) {
	getAgentHostnameFn = fn
}

// ResetGetAgentHostnameFn restores the default getAgentHostnameFn.
func ResetGetAgentHostnameFn() {
	getAgentHostnameFn = job.GetAgentHostname
}

// GetAgentHostname returns the agent's hostname field for testing.
func GetAgentHostname(
	a *Agent,
) string {
	return a.hostname
}

// SetAgentHostname sets the agent's hostname field for testing.
func SetAgentHostname(
	a *Agent,
	hostname string,
) {
	a.hostname = hostname
}

// SetDockerNewFn overrides the dockerNewFn used by the factory for testing.
// NOTE: This overrides the package-level var in the agent package, not cmd.
// For cmd-level tests, use the cmd package's own override.
func SetDockerNewFn(
	fn func() (*dockerProv.Client, error),
) {
	_ = fn // no-op: dockerNewFn lives in cmd package now
}

// ResetDockerNewFn is a no-op kept for backward compat with test suites.
func ResetDockerNewFn() {}

// SetProcStatusPath overrides the procStatusPath for testing.
func SetProcStatusPath(
	p string,
) {
	procStatusPath = p
}

// ResetProcStatusPath restores the default procStatusPath.
func ResetProcStatusPath() {
	procStatusPath = "/proc/self/status"
}

func ExportCheckCapabilities(
	logger *slog.Logger,
) []PreflightResult {
	return checkCapabilities(logger)
}

// --- Field accessors for Agent struct ---

// GetAgentAppConfig returns the agent's appConfig field for testing.
func GetAgentAppConfig(
	a *Agent,
) config.Config {
	return a.appConfig
}

// SetAgentAppConfig sets the agent's appConfig field for testing.
// Also re-registers the node processor so that label changes are picked up
// by subsequent ExportProcessNodeOperation calls.
func SetAgentAppConfig(
	a *Agent,
	cfg config.Config,
) {
	a.appConfig = cfg
	// Re-register the node processor with the updated config so that
	// label/config-dependent dispatch picks up the change.
	if a.registry != nil {
		a.registry.processors["node"] = NewNodeProcessor(
			a.hostProvider,
			a.diskProvider,
			a.memProvider,
			a.loadProvider,
			sysctl.Provider(nil),
			ntp.Provider(nil),
			timezone.Provider(nil),
			power.Provider(nil),
			nil,
			nil,
			nil,
			nil,
			nil,
			cfg,
			a.logger,
		)
	}
}

// GetAgentHostProvider returns the agent's hostProvider field for testing.
func GetAgentHostProvider(
	a *Agent,
) nodeHost.Provider {
	return a.hostProvider
}

// SetAgentHostProvider sets the agent's hostProvider field for testing.
// This also updates the corresponding processor in the registry so that
// job processing uses the new provider.
func SetAgentHostProvider(
	a *Agent,
	p nodeHost.Provider,
) {
	a.hostProvider = p
	// Re-register the node processor with the new host provider so job
	// dispatch picks up the change.
	if a.registry != nil {
		a.registry.processors["node"] = NewNodeProcessor(
			p,
			a.diskProvider,
			a.memProvider,
			a.loadProvider,
			sysctl.Provider(nil),
			ntp.Provider(nil),
			timezone.Provider(nil),
			power.Provider(nil),
			nil,
			nil,
			nil,
			nil,
			nil,
			a.appConfig,
			a.logger,
		)
	}
}

// GetAgentNetinfoProvider returns the agent's netinfoProvider field for testing.
func GetAgentNetinfoProvider(
	a *Agent,
) netinfo.Provider {
	return a.netinfoProvider
}

// SetAgentNetinfoProvider sets the agent's netinfoProvider field for testing.
func SetAgentNetinfoProvider(
	a *Agent,
	p netinfo.Provider,
) {
	a.netinfoProvider = p
}

// ExportGetHostProvider returns the host provider from the registry for testing.
// Satisfies the TestProviderFactoryMethods suite.
func ExportGetHostProvider(
	a *Agent,
) nodeHost.Provider {
	return a.hostProvider
}

// ExportGetDiskProvider returns the disk provider from the registry for testing.
func ExportGetDiskProvider(
	a *Agent,
) diskProv.Provider {
	return a.diskProvider
}

// ExportGetMemProvider returns the mem provider from the registry for testing.
func ExportGetMemProvider(
	a *Agent,
) memProv.Provider {
	return a.memProvider
}

// ExportGetLoadProvider returns the load provider from the registry for testing.
func ExportGetLoadProvider(
	a *Agent,
) load.Provider {
	return a.loadProvider
}

// ExportGetDNSProvider returns the first dns.Provider found in the registry
// for testing.
func ExportGetDNSProvider(
	a *Agent,
) dns.Provider {
	if a.registry == nil {
		return nil
	}
	for _, p := range a.registry.AllProviders() {
		if v, ok := p.(dns.Provider); ok {
			return v
		}
	}
	return nil
}

// ExportGetPingProvider returns the first ping.Provider found in the registry
// for testing.
func ExportGetPingProvider(
	a *Agent,
) ping.Provider {
	if a.registry == nil {
		return nil
	}
	for _, p := range a.registry.AllProviders() {
		if v, ok := p.(ping.Provider); ok {
			return v
		}
	}
	return nil
}

// ExportGetCommandProvider returns the first command.Provider found in the
// registry for testing.
func ExportGetCommandProvider(
	a *Agent,
) command.Provider {
	if a.registry == nil {
		return nil
	}
	for _, p := range a.registry.AllProviders() {
		if v, ok := p.(command.Provider); ok {
			return v
		}
	}
	return nil
}

// ExportGetFileProvider returns the first fileProv.Provider found in the
// registry for testing.
func ExportGetFileProvider(
	a *Agent,
) fileProv.Provider {
	if a.registry == nil {
		return nil
	}
	for _, p := range a.registry.AllProviders() {
		if v, ok := p.(fileProv.Provider); ok {
			return v
		}
	}
	return nil
}

// GetAgentCachedFacts returns the agent's cachedFacts field for testing.
func GetAgentCachedFacts(
	a *Agent,
) *job.FactsRegistration {
	return a.cachedFacts
}

// SetAgentCachedFacts sets the agent's cachedFacts field for testing.
func SetAgentCachedFacts(
	a *Agent,
	facts *job.FactsRegistration,
) {
	a.cachedFacts = facts
}

// GetAgentState returns the agent's state field for testing.
func GetAgentState(
	a *Agent,
) string {
	return a.state
}

// SetAgentState sets the agent's state field for testing.
func SetAgentState(
	a *Agent,
	state string,
) {
	a.state = state
}

// SetAgentLifecycle sets the agent's ctx/cancel and consumerCtx/consumerCancel for testing.
func SetAgentLifecycle(
	ctx context.Context,
	consumerCtx context.Context,
	a *Agent,
	cancel context.CancelFunc,
	consumerCancel context.CancelFunc,
) {
	a.ctx = ctx
	a.cancel = cancel
	a.consumerCtx = consumerCtx
	a.consumerCancel = consumerCancel
}

// GetAgentWG returns the agent's wg field for testing (as a pointer to allow Wait).
func WaitAgentWG(
	a *Agent,
) {
	a.wg.Wait()
}

// SetGetIdentityFn overrides the getIdentityFn variable for testing.
func SetGetIdentityFn(
	fn func(avfs.VFS, string) (*identity.Identity, error),
) {
	getIdentityFn = fn
}

// ResetGetIdentityFn restores the default getIdentityFn variable.
func ResetGetIdentityFn() {
	getIdentityFn = identity.GetIdentity
}

// ExportHandlePKIEnrollment exposes the private handlePKIEnrollment method for testing.
func ExportHandlePKIEnrollment(
	ctx context.Context,
	a *Agent,
) error {
	return a.handlePKIEnrollment(ctx)
}

// GetAgentPKIManager returns the agent's pkiManager field for testing.
func GetAgentPKIManager(
	a *Agent,
) *pki.Manager {
	return a.pkiManager
}

// SetAgentPKIManager sets the agent's pkiManager field for testing.
func SetAgentPKIManager(
	a *Agent,
	m *pki.Manager,
) {
	a.pkiManager = m
}

// ExportPublishEnrollmentRequest exposes the private publishEnrollmentRequest method for testing.
func ExportPublishEnrollmentRequest(
	a *Agent,
) error {
	return a.publishEnrollmentRequest()
}

// SetMarshalJSONEnrollment overrides the marshalJSONEnrollment function for testing.
func SetMarshalJSONEnrollment(
	fn func(interface{}) ([]byte, error),
) {
	marshalJSONEnrollment = fn
}

// ResetMarshalJSONEnrollment restores the default marshalJSONEnrollment function.
func ResetMarshalJSONEnrollment() {
	marshalJSONEnrollment = json.Marshal
}

// SetAgentNATSClient sets the agent's natsClient field for testing.
func SetAgentNATSClient(
	a *Agent,
	c NATSPublisher,
) {
	a.natsClient = c
}

// ExportUnwrapJobEnvelope exposes the private unwrapJobEnvelope method for testing.
func ExportUnwrapJobEnvelope(
	a *Agent,
	data []byte,
) ([]byte, error) {
	return a.unwrapJobEnvelope(data)
}

// ExportStartEnrollmentListener exposes the private startEnrollmentListener method for testing.
func ExportStartEnrollmentListener(
	ctx context.Context,
	a *Agent,
) {
	a.startEnrollmentListener(ctx)
}

// ExportHandleEnrollmentResponse exposes the private handleEnrollmentResponse method for testing.
func ExportHandleEnrollmentResponse(
	a *Agent,
	msg *nats.Msg,
) {
	a.handleEnrollmentResponse(msg)
}
