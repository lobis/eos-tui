package eos

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

var shellSafeArgPattern = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

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

func shellDisplayJoin(args []string) string {
	display := make([]string, len(args))
	for i, arg := range args {
		if arg != "" && shellSafeArgPattern.MatchString(arg) {
			display[i] = arg
		} else {
			display[i] = shellQuote(arg)
		}
	}
	return strings.Join(display, " ")
}

// ShellJoin is the exported version of shellJoin for use by other packages.
func ShellJoin(args []string) string { return shellJoin(args) }

// ShellDisplayJoin is the exported version of shellDisplayJoin for use by other packages.
func ShellDisplayJoin(args []string) string { return shellDisplayJoin(args) }

func normalizeClusterInstance(target string) string {
	target = strings.TrimSpace(target)
	if target == "" || strings.ContainsAny(target, " \t\r\n") {
		return ""
	}

	if at := strings.LastIndex(target, "@"); at != -1 {
		target = target[at+1:]
	}
	target = hostOnly(target)
	target = strings.Split(target, ".")[0]
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}

	for _, marker := range []string{"-ns-", "-mgm-", "-qdb-", "-fst-"} {
		if idx := strings.Index(target, marker); idx > 0 {
			target = target[:idx]
			break
		}
	}
	for _, suffix := range []string{"-ns", "-mgm", "-qdb", "-fst"} {
		if strings.HasSuffix(target, suffix) {
			target = strings.TrimSuffix(target, suffix)
			break
		}
	}

	if target == "" || strings.ContainsAny(target, " \t\r\n") {
		return ""
	}
	return target
}

// NormalizeClusterInstance extracts the logical EOS instance name from a
// cluster alias or hostname such as "eospilot", "root@eospilot.cern.ch", or
// "root@eospilot-ns-02.cern.ch".
func NormalizeClusterInstance(target string) string { return normalizeClusterInstance(target) }

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

func parseMonitoringAssignments(output []byte) map[string]string {
	values := make(map[string]string)
	for _, rawLine := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "*") {
			continue
		}

		for _, field := range strings.Fields(line) {
			if !strings.Contains(field, "=") {
				continue
			}
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 || parts[0] == "" {
				continue
			}
			values[parts[0]] = parts[1]
		}
	}
	return values
}

func splitCSVList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
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
