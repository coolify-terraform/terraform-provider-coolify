package validate

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// coolifyFrequencyPattern matches Coolify schedule frequencies:
// standard 5-field cron, optional 6-field when allowSixFields is true, and
// human strings from Coolify VALID_CRON_STRINGS (bootstrap/helpers/constants.php):
// every_minute, hourly, daily, weekly, monthly, yearly, and @hourly/@daily/etc.
//
// Bare names without "@" are accepted by Coolify; rejecting them causes  plan-time
// false failures for valid API values.
const (
	coolifyHuman = `(?:every_minute|hourly|daily|weekly|monthly|yearly|@(?:annually|yearly|monthly|weekly|daily|hourly))`
	// 5-field cron: five tokens separated by spaces
	coolifyCron5 = `(?:\S+\s+){4}\S+`
	// 5- or 6-field cron (scheduled tasks)
	coolifyCron5or6 = `(?:\S+\s+){4,5}\S+`
)

var (
	coolifyFrequency5Regexp    = regexp.MustCompile(`^(?:` + coolifyCron5 + `|` + coolifyHuman + `)$`)
	coolifyFrequency5or6Regexp = regexp.MustCompile(`^(?:` + coolifyCron5or6 + `|` + coolifyHuman + `)$`)
)

// CoolifyFrequency validates cron / Coolify human schedule strings (5-field cron).
func CoolifyFrequency() validator.String {
	return stringvalidator.RegexMatches(coolifyFrequency5Regexp,
		`must be a valid cron expression (e.g. "0 2 * * *") or Coolify human schedule (daily, hourly, @daily, every_minute, …)`)
}

// CoolifyFrequencyAllowSeconds validates 5- or 6-field cron plus Coolify human schedules.
func CoolifyFrequencyAllowSeconds() validator.String {
	return stringvalidator.RegexMatches(coolifyFrequency5or6Regexp,
		`must be a valid cron expression (e.g. "*/5 * * * *" or six-field) or Coolify human schedule (daily, @daily, …)`)
}
