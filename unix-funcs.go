//go:build !windows
// +build !windows

package main

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getResponseTime extracts time value from Ping output
// and tells if this is a failure message or not.
// -1 means the output is not a successful reply.
// true means the output states for a ping failure.
// false means to ignore the output (statistics data).
// On Linux the Ping output entry looks like below:
// <64 bytes from 127.0.0.1: icmp_seq=1 ttl=64 time=0.041 ms>
func getResponseTime(output string) (int, bool) {
	indexT := strings.Index(output, "time=")
	if indexT > 0 {
		indexM := strings.Index(output, " ms")
		value := output[(indexT + 5):indexM]
		response, _ := strconv.Atoi(value)
		return response, false
	}

	// ignore these outputs entries.
	if strings.HasPrefix(output, "PING") || strings.HasPrefix(output, "---") ||
		strings.HasPrefix(output, "rtt") || strings.Contains(output, "%") {
		return -1, false
	}

	return -1, true
}

// buildPingCommand constructs full command to run. The ping should
// run indefinitely by default unless a requests is defined.
func buildPingCommand(ip string, ctx context.Context) (string, *exec.Cmd) {
	cfg := dbs.getConfig(ip)
	cfg.start = getCurrentTime()
	var cmd *exec.Cmd

	syntax := fmt.Sprintf("ping %s", ip)

	if cfg.requests > 0 {
		syntax = syntax + fmt.Sprintf(" -c %d", cfg.requests)
	}

	if cfg.timeout > 0 {
		syntax = syntax + fmt.Sprintf(" -W %d", cfg.timeout)
	}

	if cfg.size > 0 {
		syntax = syntax + fmt.Sprintf(" -s %d", cfg.size)
	}

	cmd = exec.CommandContext(ctx, LinuxShell, "-c", syntax)

	return strconv.Itoa(cfg.threshold), cmd
}
