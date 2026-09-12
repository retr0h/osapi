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

package file

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/osapi-io/osapi/internal/controller/api/file/gen"
	"github.com/osapi-io/osapi/internal/job"
)

// GetFileStale returns deployments where the source object has been updated
// since the file was last deployed.
func (f *File) GetFileStale(
	ctx context.Context,
	_ gen.GetFileStaleRequestObject,
) (gen.GetFileStaleResponseObject, error) {
	if f.stateKV == nil {
		errMsg := "file state KV not available"
		f.logger.Error(errMsg)

		return gen.GetFileStale500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	keys, err := f.stateKV.Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return gen.GetFileStale200JSONResponse{
				Stale: []gen.StaleDeployment{},
				Total: 0,
			}, nil
		}

		errMsg := "failed to list file state keys"
		f.logger.Error(errMsg, slog.String("error", err.Error()))

		return gen.GetFileStale500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	var stale []gen.StaleDeployment

	for _, key := range keys {
		entry, err := f.stateKV.Get(ctx, key)
		if err != nil {
			f.logger.Debug(
				"skipping state entry",
				slog.String("key", key),
				slog.String("error", err.Error()),
			)

			continue
		}

		var state job.FileState
		if err := json.Unmarshal(entry.Value(), &state); err != nil {
			f.logger.Debug(
				"skipping malformed state entry",
				slog.String("key", key),
				slog.String("error", err.Error()),
			)

			continue
		}

		if state.UndeployedAt != "" {
			continue
		}

		hostname := extractHostname(key)

		provider := providerFromPath(state.Path)

		currentContent, err := f.objStore.GetBytes(ctx, state.ObjectName)
		if err != nil {
			// Object was deleted — include as stale with empty current_sha.
			stale = append(stale, gen.StaleDeployment{
				ObjectName:  state.ObjectName,
				Hostname:    hostname,
				Provider:    provider,
				Path:        state.Path,
				DeployedSha: state.SHA256,
				CurrentSha:  "",
				DeployedAt:  state.DeployedAt,
			})

			continue
		}

		currentSHA := computeSHA256(currentContent)
		if currentSHA != state.SHA256 {
			stale = append(stale, gen.StaleDeployment{
				ObjectName:  state.ObjectName,
				Hostname:    hostname,
				Provider:    provider,
				Path:        state.Path,
				DeployedSha: state.SHA256,
				CurrentSha:  currentSHA,
				DeployedAt:  state.DeployedAt,
			})
		}
	}

	if stale == nil {
		stale = []gen.StaleDeployment{}
	}

	return gen.GetFileStale200JSONResponse{
		Stale: stale,
		Total: len(stale),
	}, nil
}

// extractHostname extracts the hostname from a state KV key.
// The key format is "<hostname>.<sha256-of-path>" where the SHA is
// 64 hex characters. We remove the trailing dot + 64 chars.
func extractHostname(
	key string,
) string {
	// dot (1) + sha256 hex (64) = 65 chars from the end
	if len(key) > 65 {
		return key[:len(key)-65]
	}

	return key
}

// computeSHA256 returns the hex-encoded SHA-256 digest of data.
func computeSHA256(
	data []byte,
) string {
	h := sha256.Sum256(data)

	return hex.EncodeToString(h[:])
}

// providerFromPath derives the provider name from the deployment path.
func providerFromPath(
	path string,
) string {
	switch {
	case strings.HasPrefix(path, "/etc/systemd/system/"):
		return "service"
	case strings.HasPrefix(path, "/usr/local/share/ca-certificates/"):
		return "certificate"
	case strings.HasPrefix(path, "/etc/cron.d/") ||
		strings.HasPrefix(path, "/etc/cron.hourly/") ||
		strings.HasPrefix(path, "/etc/cron.daily/") ||
		strings.HasPrefix(path, "/etc/cron.weekly/") ||
		strings.HasPrefix(path, "/etc/cron.monthly/"):
		return "cron"
	default:
		return "file"
	}
}
