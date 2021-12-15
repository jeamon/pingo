//go:build windows
// +build windows

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
// On Windows the Ping output entry looks like below:
// <Reply from 8.8.8.8: bytes=32 time=1160ms TTL=56>
// <Reply from 127.0.0.1: bytes=32 time<1ms TTL=128>
func getResponseTime(output string) (int, bool) {
	indexT := strings.Index(output, "time=")
	if indexT > 0 {
		indexM := strings.Index(output, "ms")
		value := output[(indexT + 5):indexM]
		response, _ := strconv.Atoi(value)
		return response, false
	} else {
		indexT := strings.Index(output, "time<")
		if indexT > 0 {
			indexM := strings.Index(output, "ms")
			value := output[(indexT + 5):indexM]
			response, _ := strconv.Atoi(value)
			return response, false
		}
	}

	// ignore these outputs entries.
	if strings.HasPrefix(output, "Pinging") || strings.HasPrefix(output, "Ping") ||
		strings.HasPrefix(output, "Packets") || strings.HasPrefix(output, "Approximate") ||
		strings.HasPrefix(output, "Minimum") {
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
		syntax = syntax + fmt.Sprintf(" -n %d", cfg.requests)
	} else {
		syntax = syntax + " -t"
	}

	if cfg.timeout > 0 {
		syntax = syntax + fmt.Sprintf(" -w %d", cfg.timeout)
	}

	if cfg.size > 0 {
		syntax = syntax + fmt.Sprintf(" -l %d", cfg.size)
	}

	cmd = exec.CommandContext(ctx, "cmd", "/C", syntax)

	return strconv.Itoa(cfg.threshold), cmd
}
