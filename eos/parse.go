package eos

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

// stripEOSPreamble removes leading lines that are not part of a JSON payload.
// EOS commands occasionally emit `* <message>` lines on stdout (e.g. error or
// info annotations) before or after the JSON.  This function returns the first
// contiguous block that looks like JSON (starts with `[` or `{`).
func stripEOSPreamble(b []byte) []byte {
	for _, line := range strings.SplitAfter(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
			// Return from this point to end-of-output.
			idx := strings.Index(string(b), trimmed)
			if idx >= 0 {
				return []byte(strings.TrimSpace(string(b[idx:])))
			}
		}
	}
	return b
}

// shellQuote wraps s in single quotes, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func toUint64(v any) uint64 {
	switch val := v.(type) {
	case float64:
		return uint64(val)
	case uint64:
		return val
	case int64:
		return uint64(val)
	case int:
		return uint64(val)
	default:
		return 0
	}
}

var nsStatLinePattern = regexp.MustCompile(`^ALL\s+(.+?)\s{2,}(.+)$`)

func parseLabeledValues(output string) map[string]string {
	values := make(map[string]string)
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		matches := nsStatLinePattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		values[strings.TrimSpace(matches[1])] = strings.TrimSpace(matches[2])
	}

	return values
}

func parseUint(raw string) uint64 {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return 0
	}

	value, _ := strconv.ParseUint(fields[0], 10, 64)
	return value
}

func parseHumanBytes(raw string) uint64 {
	fields := strings.Fields(raw)
	if len(fields) < 2 {
		return 0
	}

	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	multiplier := float64(1)
	switch strings.ToUpper(fields[1]) {
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	case "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	return uint64(value * multiplier)
}

func cleanPath(rawPath string) string {
	if rawPath == "" {
		return "/"
	}

	cleaned := path.Clean(rawPath)
	if !strings.HasPrefix(cleaned, "/") {
		return "/" + cleaned
	}

	return cleaned
}

func splitHostPort(hp string) (string, int) {
	parts := strings.Split(hp, ":")
	host := parts[0]
	port := 0
	if len(parts) > 1 {
		port, _ = strconv.Atoi(parts[1])
	}
	return host, port
}

// hostOnly strips the port suffix from a host:port string.
func hostOnly(hostPort string) string {
	if idx := strings.LastIndex(hostPort, ":"); idx != -1 {
		return hostPort[:idx]
	}
	return hostPort
}

// HostOnly is the exported version of hostOnly for use by other packages.
func HostOnly(hostPort string) string { return hostOnly(hostPort) }
