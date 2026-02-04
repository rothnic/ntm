package quota

// OpenCode quota parsing
// OpenCode delegates to underlying provider (Claude, Gemini, etc.) for quota tracking.
// The SDK provides session-level usage info which we parse here.

import (
	"regexp"
	"strconv"
)

var opencodeUsagePatterns = struct {
	// Usage patterns based on OpenCode session info
	Tokens   *regexp.Regexp
	Context  *regexp.Regexp
	Model    *regexp.Regexp
	Limited  *regexp.Regexp
}{
	Tokens:   regexp.MustCompile(`(?i)tokens?[:\s]+(\d+(?:,\d+)*)\s*(?:/\s*(\d+(?:,\d+)*))?`),
	Context:  regexp.MustCompile(`(?i)context[:\s]+(\d+(?:\.\d+)?)\s*%`),
	Model:    regexp.MustCompile(`(?i)model[:\s]+(\S+)`),
	Limited:  regexp.MustCompile(`(?i)(?:rate\s*limit|limited|exceeded|quota\s*exceeded)`),
}

var opencodeStatusPatterns = struct {
	Session *regexp.Regexp
	Model   *regexp.Regexp
	Status  *regexp.Regexp
}{
	Session: regexp.MustCompile(`(?i)session[:\s]+(\S+)`),
	Model:   regexp.MustCompile(`(?i)model[:\s]+(\S+)`),
	Status:  regexp.MustCompile(`(?i)status[:\s]+(running|idle|error)`),
}

// parseOpenCodeUsage parses OpenCode usage output
func parseOpenCodeUsage(info *QuotaInfo, output string) (bool, error) {
	found := false

	// Parse token usage (e.g., "tokens: 5,000 / 100,000")
	if match := opencodeUsagePatterns.Tokens.FindStringSubmatch(output); len(match) > 1 {
		// Remove commas and parse
		used := parseTokenCount(match[1])
		if used > 0 {
			if len(match) > 2 && match[2] != "" {
				max := parseTokenCount(match[2])
				if max > 0 {
					info.SessionUsage = float64(used) / float64(max) * 100
					found = true
				}
			}
		}
	}

	// Parse context usage percentage
	if match := opencodeUsagePatterns.Context.FindStringSubmatch(output); len(match) > 1 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			info.SessionUsage = val
			found = true
		}
	}

	// Check for rate limiting
	if opencodeUsagePatterns.Limited.MatchString(output) {
		info.IsLimited = true
		found = true
	}

	return found, nil
}

// parseOpenCodeStatus parses OpenCode status output
func parseOpenCodeStatus(info *QuotaInfo, output string) {
	// Parse model (use as account identifier for OpenCode)
	if match := opencodeStatusPatterns.Model.FindStringSubmatch(output); len(match) > 1 {
		info.AccountID = match[1] // Model name serves as identifier
	}
}

// parseTokenCount parses a token count that may include commas
func parseTokenCount(s string) int64 {
	// Remove commas
	clean := ""
	for _, c := range s {
		if c >= '0' && c <= '9' {
			clean += string(c)
		}
	}
	if val, err := strconv.ParseInt(clean, 10, 64); err == nil {
		return val
	}
	return 0
}
