//go:build !windows
// +build !windows

package main

import (
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
