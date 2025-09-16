package client

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kart-io/notifyhub/notifiers"
)

// ================================
// Strong Typed Target Builders
// ================================

// EmailTarget creates a strongly typed email target with validation
func EmailTarget(email string) (notifiers.Target, error) {
	if !isValidEmailFormat(email) {
		return notifiers.Target{}, fmt.Errorf("invalid email format: %s", email)
	}
	return notifiers.Target{
		Type:  notifiers.TargetTypeEmail,
		Value: strings.TrimSpace(email),
	}, nil
}

// MustEmailTarget creates an email target and panics on validation error
func MustEmailTarget(email string) notifiers.Target {
	target, err := EmailTarget(email)
	if err != nil {
		panic(err)
	}
	return target
}

// EmailTargets creates multiple email targets with validation
func EmailTargets(emails ...string) ([]notifiers.Target, error) {
	targets := make([]notifiers.Target, 0, len(emails))
	for _, email := range emails {
		target, err := EmailTarget(email)
		if err != nil {
			return nil, fmt.Errorf("invalid email %s: %v", email, err)
		}
		targets = append(targets, target)
	}
	return targets, nil
}

// MustEmailTargets creates multiple email targets and panics on validation error
func MustEmailTargets(emails ...string) []notifiers.Target {
	targets, err := EmailTargets(emails...)
	if err != nil {
		panic(err)
	}
	return targets
}

// UserTarget creates a strongly typed user target
func UserTarget(userID, platform string) (notifiers.Target, error) {
	if strings.TrimSpace(userID) == "" {
		return notifiers.Target{}, fmt.Errorf("user ID cannot be empty")
	}
	if strings.TrimSpace(platform) == "" {
		return notifiers.Target{}, fmt.Errorf("platform cannot be empty")
	}
	return notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    strings.TrimSpace(userID),
		Platform: strings.TrimSpace(platform),
	}, nil
}

// MustUserTarget creates a user target and panics on validation error
func MustUserTarget(userID, platform string) notifiers.Target {
	target, err := UserTarget(userID, platform)
	if err != nil {
		panic(err)
	}
	return target
}

// GroupTarget creates a strongly typed group target
func GroupTarget(groupID, platform string) (notifiers.Target, error) {
	if strings.TrimSpace(groupID) == "" {
		return notifiers.Target{}, fmt.Errorf("group ID cannot be empty")
	}
	if strings.TrimSpace(platform) == "" {
		return notifiers.Target{}, fmt.Errorf("platform cannot be empty")
	}
	return notifiers.Target{
		Type:     notifiers.TargetTypeGroup,
		Value:    strings.TrimSpace(groupID),
		Platform: strings.TrimSpace(platform),
	}, nil
}

// MustGroupTarget creates a group target and panics on validation error
func MustGroupTarget(groupID, platform string) notifiers.Target {
	target, err := GroupTarget(groupID, platform)
	if err != nil {
		panic(err)
	}
	return target
}

// ChannelTarget creates a strongly typed channel target
func ChannelTarget(channelID, platform string) (notifiers.Target, error) {
	if strings.TrimSpace(channelID) == "" {
		return notifiers.Target{}, fmt.Errorf("channel ID cannot be empty")
	}
	if strings.TrimSpace(platform) == "" {
		return notifiers.Target{}, fmt.Errorf("platform cannot be empty")
	}
	return notifiers.Target{
		Type:     notifiers.TargetTypeChannel,
		Value:    strings.TrimSpace(channelID),
		Platform: strings.TrimSpace(platform),
	}, nil
}

// MustChannelTarget creates a channel target and panics on validation error
func MustChannelTarget(channelID, platform string) notifiers.Target {
	target, err := ChannelTarget(channelID, platform)
	if err != nil {
		panic(err)
	}
	return target
}

// ================================
// Platform-Specific Target Builders
// ================================

// FeishuUser creates a Feishu user target
func FeishuUser(userID string) (notifiers.Target, error) {
	return UserTarget(userID, "feishu")
}

// MustFeishuUser creates a Feishu user target and panics on error
func MustFeishuUser(userID string) notifiers.Target {
	return MustUserTarget(userID, "feishu")
}

// FeishuGroup creates a Feishu group target
func FeishuGroup(groupID string) (notifiers.Target, error) {
	return GroupTarget(groupID, "feishu")
}

// MustFeishuGroup creates a Feishu group target and panics on error
func MustFeishuGroup(groupID string) notifiers.Target {
	return MustGroupTarget(groupID, "feishu")
}

// SlackUser creates a Slack user target
func SlackUser(userID string) (notifiers.Target, error) {
	// Handle @username format
	if strings.HasPrefix(userID, "@") {
		userID = strings.TrimPrefix(userID, "@")
	}
	return UserTarget(userID, "slack")
}

// MustSlackUser creates a Slack user target and panics on error
func MustSlackUser(userID string) notifiers.Target {
	target, err := SlackUser(userID)
	if err != nil {
		panic(err)
	}
	return target
}

// SlackChannel creates a Slack channel target
func SlackChannel(channelID string) (notifiers.Target, error) {
	// Handle #channel format
	if strings.HasPrefix(channelID, "#") {
		channelID = strings.TrimPrefix(channelID, "#")
	}
	return ChannelTarget(channelID, "slack")
}

// MustSlackChannel creates a Slack channel target and panics on error
func MustSlackChannel(channelID string) notifiers.Target {
	target, err := SlackChannel(channelID)
	if err != nil {
		panic(err)
	}
	return target
}

// ================================
// Smart Target Parsing
// ================================

// ParseTarget intelligently parses a target string and returns the appropriate target
func ParseTarget(target string) (notifiers.Target, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return notifiers.Target{}, fmt.Errorf("target cannot be empty")
	}

	// Email detection
	if isValidEmailFormat(target) {
		return EmailTarget(target)
	}

	// Slack channel detection (#channel)
	if strings.HasPrefix(target, "#") {
		return SlackChannel(target)
	}

	// Slack user detection (@user)
	if strings.HasPrefix(target, "@") {
		return SlackUser(target)
	}

	// Platform:ID format detection (platform:identifier)
	if parts := strings.SplitN(target, ":", 2); len(parts) == 2 {
		platform := strings.ToLower(strings.TrimSpace(parts[0]))
		identifier := strings.TrimSpace(parts[1])

		switch platform {
		case "feishu", "lark":
			// Assume group by default for Feishu
			return FeishuGroup(identifier)
		case "slack":
			if strings.HasPrefix(identifier, "#") {
				return SlackChannel(identifier)
			} else if strings.HasPrefix(identifier, "@") {
				return SlackUser(identifier)
			} else {
				return SlackUser(identifier) // Default to user
			}
		case "email":
			return EmailTarget(identifier)
		default:
			return notifiers.Target{}, fmt.Errorf("unsupported platform: %s", platform)
		}
	}

	return notifiers.Target{}, fmt.Errorf("unable to parse target: %s", target)
}

// MustParseTarget parses a target and panics on error
func MustParseTarget(target string) notifiers.Target {
	parsed, err := ParseTarget(target)
	if err != nil {
		panic(err)
	}
	return parsed
}

// ParseTargets parses multiple target strings
func ParseTargets(targets ...string) ([]notifiers.Target, error) {
	parsed := make([]notifiers.Target, 0, len(targets))
	for _, target := range targets {
		p, err := ParseTarget(target)
		if err != nil {
			return nil, fmt.Errorf("failed to parse target '%s': %v", target, err)
		}
		parsed = append(parsed, p)
	}
	return parsed, nil
}

// MustParseTargets parses multiple targets and panics on error
func MustParseTargets(targets ...string) []notifiers.Target {
	parsed, err := ParseTargets(targets...)
	if err != nil {
		panic(err)
	}
	return parsed
}

// ================================
// Target Builder Pattern
// ================================

// TargetBuilder provides a fluent interface for building multiple targets
type TargetBuilder struct {
	targets []notifiers.Target
	errors  []error
}

// NewTargetBuilder creates a new target builder
func NewTargetBuilder() *TargetBuilder {
	return &TargetBuilder{
		targets: make([]notifiers.Target, 0),
		errors:  make([]error, 0),
	}
}

// Email adds an email target
func (tb *TargetBuilder) Email(email string) *TargetBuilder {
	target, err := EmailTarget(email)
	if err != nil {
		tb.errors = append(tb.errors, err)
	} else {
		tb.targets = append(tb.targets, target)
	}
	return tb
}

// Emails adds multiple email targets
func (tb *TargetBuilder) Emails(emails ...string) *TargetBuilder {
	for _, email := range emails {
		tb.Email(email)
	}
	return tb
}

// User adds a user target
func (tb *TargetBuilder) User(userID, platform string) *TargetBuilder {
	target, err := UserTarget(userID, platform)
	if err != nil {
		tb.errors = append(tb.errors, err)
	} else {
		tb.targets = append(tb.targets, target)
	}
	return tb
}

// Group adds a group target
func (tb *TargetBuilder) Group(groupID, platform string) *TargetBuilder {
	target, err := GroupTarget(groupID, platform)
	if err != nil {
		tb.errors = append(tb.errors, err)
	} else {
		tb.targets = append(tb.targets, target)
	}
	return tb
}

// Channel adds a channel target
func (tb *TargetBuilder) Channel(channelID, platform string) *TargetBuilder {
	target, err := ChannelTarget(channelID, platform)
	if err != nil {
		tb.errors = append(tb.errors, err)
	} else {
		tb.targets = append(tb.targets, target)
	}
	return tb
}

// FeishuUser adds a Feishu user target
func (tb *TargetBuilder) FeishuUser(userID string) *TargetBuilder {
	return tb.User(userID, "feishu")
}

// FeishuGroup adds a Feishu group target
func (tb *TargetBuilder) FeishuGroup(groupID string) *TargetBuilder {
	return tb.Group(groupID, "feishu")
}

// SlackUser adds a Slack user target
func (tb *TargetBuilder) SlackUser(userID string) *TargetBuilder {
	target, err := SlackUser(userID)
	if err != nil {
		tb.errors = append(tb.errors, err)
	} else {
		tb.targets = append(tb.targets, target)
	}
	return tb
}

// SlackChannel adds a Slack channel target
func (tb *TargetBuilder) SlackChannel(channelID string) *TargetBuilder {
	target, err := SlackChannel(channelID)
	if err != nil {
		tb.errors = append(tb.errors, err)
	} else {
		tb.targets = append(tb.targets, target)
	}
	return tb
}

// Parse adds a parsed target
func (tb *TargetBuilder) Parse(target string) *TargetBuilder {
	parsed, err := ParseTarget(target)
	if err != nil {
		tb.errors = append(tb.errors, err)
	} else {
		tb.targets = append(tb.targets, parsed)
	}
	return tb
}

// ParseAll adds multiple parsed targets
func (tb *TargetBuilder) ParseAll(targets ...string) *TargetBuilder {
	for _, target := range targets {
		tb.Parse(target)
	}
	return tb
}

// Build returns the built targets and any validation errors
func (tb *TargetBuilder) Build() ([]notifiers.Target, error) {
	if len(tb.errors) > 0 {
		return nil, fmt.Errorf("target validation errors: %v", tb.errors)
	}
	return tb.targets, nil
}

// MustBuild returns the built targets and panics on validation error
func (tb *TargetBuilder) MustBuild() []notifiers.Target {
	targets, err := tb.Build()
	if err != nil {
		panic(err)
	}
	return targets
}

// Count returns the number of targets added
func (tb *TargetBuilder) Count() int {
	return len(tb.targets)
}

// HasErrors returns true if there are validation errors
func (tb *TargetBuilder) HasErrors() bool {
	return len(tb.errors) > 0
}

// Errors returns all validation errors
func (tb *TargetBuilder) Errors() []error {
	return tb.errors
}

// ================================
// Validation Helpers
// ================================

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isValidEmailFormat validates email format using regex
func isValidEmailFormat(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 || len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// ValidateTarget validates a target
func ValidateTarget(target notifiers.Target) error {
	if target.Type == "" {
		return fmt.Errorf("target type cannot be empty")
	}
	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}

	switch target.Type {
	case notifiers.TargetTypeEmail:
		if !isValidEmailFormat(target.Value) {
			return fmt.Errorf("invalid email format: %s", target.Value)
		}
	case notifiers.TargetTypeUser, notifiers.TargetTypeGroup, notifiers.TargetTypeChannel:
		if target.Platform == "" {
			return fmt.Errorf("platform is required for %s targets", target.Type)
		}
	default:
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}

	return nil
}

// ValidateTargets validates multiple targets
func ValidateTargets(targets []notifiers.Target) error {
	for i, target := range targets {
		if err := ValidateTarget(target); err != nil {
			return fmt.Errorf("target %d: %v", i, err)
		}
	}
	return nil
}

// ================================
// Enhanced Target Validation and Suggestions API
// ================================

// ValidationResult represents the result of target validation
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Target      *notifiers.Target `json:"target,omitempty"`
	Errors      []string          `json:"errors,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
	Score       int               `json:"score"` // Confidence score 0-100
}

// ValidationConfig contains configuration for target validation
type ValidationConfig struct {
	StrictEmailValidation bool `json:"strict_email_validation"`
	AllowLocalEmails      bool `json:"allow_local_emails"`
	RequireTLD            bool `json:"require_tld"`
	MaxSuggestions        int  `json:"max_suggestions"`
	EnableSpellCheck      bool `json:"enable_spell_check"`
}

// DefaultValidationConfig returns default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		StrictEmailValidation: true,
		AllowLocalEmails:      false,
		RequireTLD:            true,
		MaxSuggestions:        3,
		EnableSpellCheck:      true,
	}
}

// ValidateTargetString performs comprehensive target validation with suggestions
func ValidateTargetString(target string, config ...*ValidationConfig) *ValidationResult {
	cfg := DefaultValidationConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	result := &ValidationResult{
		Valid:       false,
		Errors:      make([]string, 0),
		Warnings:    make([]string, 0),
		Suggestions: make([]string, 0),
		Score:       0,
	}

	target = strings.TrimSpace(target)
	if target == "" {
		result.Errors = append(result.Errors, "target cannot be empty")
		return result
	}

	// Try to parse and validate the target
	parsed, err := ParseTarget(target)
	if err == nil {
		result.Valid = true
		result.Target = &parsed
		result.Score = 90

		// Additional validation based on target type
		switch parsed.Type {
		case notifiers.TargetTypeEmail:
			validateEmailAdvanced(target, result, cfg)
		case notifiers.TargetTypeUser:
			validateUserTarget(target, result, cfg)
		case notifiers.TargetTypeGroup:
			validateGroupTarget(target, result, cfg)
		case notifiers.TargetTypeChannel:
			validateChannelTarget(target, result, cfg)
		}
	} else {
		result.Errors = append(result.Errors, err.Error())
		generateTargetSuggestions(target, result, cfg)
	}

	return result
}

// validateEmailAdvanced performs advanced email validation
func validateEmailAdvanced(email string, result *ValidationResult, cfg *ValidationConfig) {
	email = strings.TrimSpace(email)

	// Basic format check
	if !isValidEmailFormat(email) {
		result.Valid = false
		result.Score = 0
		result.Errors = append(result.Errors, "invalid email format")
		generateEmailSuggestions(email, result, cfg)
		return
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		result.Valid = false
		result.Score = 0
		result.Errors = append(result.Errors, "email must contain exactly one @ symbol")
		return
	}

	local, domain := parts[0], parts[1]

	// Local part validation
	if len(local) == 0 {
		result.Valid = false
		result.Score = 0
		result.Errors = append(result.Errors, "email local part cannot be empty")
		return
	}
	if len(local) > 64 {
		result.Warnings = append(result.Warnings, "email local part is longer than recommended (64 characters)")
		result.Score -= 10
	}

	// Domain validation
	if len(domain) == 0 {
		result.Valid = false
		result.Score = 0
		result.Errors = append(result.Errors, "email domain cannot be empty")
		return
	}

	// Check for TLD requirement
	if cfg.RequireTLD && !strings.Contains(domain, ".") {
		if cfg.AllowLocalEmails {
			result.Warnings = append(result.Warnings, "email appears to be local (no TLD)")
			result.Score -= 20
		} else {
			result.Valid = false
			result.Score = 0
			result.Errors = append(result.Errors, "email domain must contain a TLD")
			return
		}
	}

	// Check for common typos in domains
	if cfg.EnableSpellCheck {
		suggestions := checkCommonEmailTypos(domain)
		if len(suggestions) > 0 {
			result.Warnings = append(result.Warnings, "possible typo in domain")
			result.Suggestions = append(result.Suggestions, suggestions...)
			result.Score -= 15
		}
	}
}

// validateUserTarget validates user targets
func validateUserTarget(target string, result *ValidationResult, cfg *ValidationConfig) {
	if strings.HasPrefix(target, "@") && len(target) < 3 {
		result.Warnings = append(result.Warnings, "user ID seems too short")
		result.Score -= 10
	}

	if strings.Contains(target, " ") {
		result.Warnings = append(result.Warnings, "user ID contains spaces, which may not be valid")
		result.Score -= 15
	}
}

// validateGroupTarget validates group targets
func validateGroupTarget(target string, result *ValidationResult, cfg *ValidationConfig) {
	if len(target) < 2 {
		result.Warnings = append(result.Warnings, "group ID seems too short")
		result.Score -= 10
	}
}

// validateChannelTarget validates channel targets
func validateChannelTarget(target string, result *ValidationResult, cfg *ValidationConfig) {
	if strings.HasPrefix(target, "#") && len(target) < 3 {
		result.Warnings = append(result.Warnings, "channel name seems too short")
		result.Score -= 10
	}
}

// generateTargetSuggestions generates suggestions for invalid targets
func generateTargetSuggestions(target string, result *ValidationResult, cfg *ValidationConfig) {
	target = strings.TrimSpace(target)

	// Check if it looks like an email but has issues
	if strings.Contains(target, "@") {
		generateEmailSuggestions(target, result, cfg)
		return
	}

	// Check if it looks like a Slack channel or user
	if strings.HasPrefix(target, "#") || strings.HasPrefix(target, "@") {
		if !strings.Contains(target, ":") {
			result.Suggestions = append(result.Suggestions,
				fmt.Sprintf("slack:%s", target),
				fmt.Sprintf("feishu:%s", target))
		}
		return
	}

	// Generic suggestions
	result.Suggestions = append(result.Suggestions,
		fmt.Sprintf("%s@example.com", target),
		fmt.Sprintf("slack:@%s", target),
		fmt.Sprintf("#%s", target))

	// Limit suggestions
	if len(result.Suggestions) > cfg.MaxSuggestions {
		result.Suggestions = result.Suggestions[:cfg.MaxSuggestions]
	}
}

// generateEmailSuggestions generates email-specific suggestions
func generateEmailSuggestions(email string, result *ValidationResult, cfg *ValidationConfig) {
	email = strings.TrimSpace(email)

	// Missing @ symbol
	if !strings.Contains(email, "@") {
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("%s@example.com", email))
		return
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return
	}

	local, domain := parts[0], parts[1]

	// Missing domain
	if domain == "" {
		result.Suggestions = append(result.Suggestions,
			fmt.Sprintf("%s@gmail.com", local),
			fmt.Sprintf("%s@example.com", local))
		return
	}

	// Common domain typos
	suggestions := checkCommonEmailTypos(domain)
	for _, suggestion := range suggestions {
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("%s@%s", local, suggestion))
	}

	// Limit suggestions
	if len(result.Suggestions) > cfg.MaxSuggestions {
		result.Suggestions = result.Suggestions[:cfg.MaxSuggestions]
	}
}

// checkCommonEmailTypos checks for common email domain typos
func checkCommonEmailTypos(domain string) []string {
	commonDomains := map[string][]string{
		"gmai":    {"gmail.com"},
		"gmial":   {"gmail.com"},
		"gmail":   {"gmail.com"},
		"gmali":   {"gmail.com"},
		"gamil":   {"gmail.com"},
		"yahooo":  {"yahoo.com"},
		"yaho":    {"yahoo.com"},
		"hotmial": {"hotmail.com"},
		"hotmai":  {"hotmail.com"},
		"outlok":  {"outlook.com"},
		"outloo":  {"outlook.com"},
		"exampl":  {"example.com"},
		"examle":  {"example.com"},
		"exmaple": {"example.com"},
	}

	domain = strings.ToLower(strings.TrimSpace(domain))

	// Direct match
	if suggestions, exists := commonDomains[domain]; exists {
		return suggestions
	}

	// Fuzzy matching for close typos
	suggestions := []string{}
	for typo, corrections := range commonDomains {
		if levenshteinDistance(domain, typo) <= 2 {
			suggestions = append(suggestions, corrections...)
		}
	}

	return suggestions
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = minInt(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// minInt returns the minimum of three integers
func minInt(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// ================================
// Batch Target Validation
// ================================

// BatchValidationResult represents results for multiple target validation
type BatchValidationResult struct {
	Results      []*ValidationResult `json:"results"`
	ValidCount   int                 `json:"valid_count"`
	ErrorCount   int                 `json:"error_count"`
	WarningCount int                 `json:"warning_count"`
	AvgScore     float64             `json:"avg_score"`
}

// ValidateTargetStrings validates multiple targets at once
func ValidateTargetStrings(targets []string, config ...*ValidationConfig) *BatchValidationResult {
	cfg := DefaultValidationConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	result := &BatchValidationResult{
		Results: make([]*ValidationResult, len(targets)),
	}

	totalScore := 0
	for i, target := range targets {
		validation := ValidateTargetString(target, cfg)
		result.Results[i] = validation

		if validation.Valid {
			result.ValidCount++
		} else {
			result.ErrorCount++
		}

		if len(validation.Warnings) > 0 {
			result.WarningCount++
		}

		totalScore += validation.Score
	}

	if len(targets) > 0 {
		result.AvgScore = float64(totalScore) / float64(len(targets))
	}

	return result
}

// ================================
// Enhanced TargetBuilder with Validation
// ================================

// ValidatedTargetBuilder extends TargetBuilder with validation features
type ValidatedTargetBuilder struct {
	*TargetBuilder
	config            *ValidationConfig
	validationResults []*ValidationResult
	autoFix           bool
}

// NewValidatedTargetBuilder creates a new validated target builder
func NewValidatedTargetBuilder(config ...*ValidationConfig) *ValidatedTargetBuilder {
	cfg := DefaultValidationConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	return &ValidatedTargetBuilder{
		TargetBuilder:     NewTargetBuilder(),
		config:            cfg,
		validationResults: make([]*ValidationResult, 0),
		autoFix:           false,
	}
}

// WithAutoFix enables automatic fixing of common issues
func (vtb *ValidatedTargetBuilder) WithAutoFix(autoFix bool) *ValidatedTargetBuilder {
	vtb.autoFix = autoFix
	return vtb
}

// AddTarget adds a target with validation
func (vtb *ValidatedTargetBuilder) AddTarget(target string) *ValidatedTargetBuilder {
	validation := ValidateTargetString(target, vtb.config)
	vtb.validationResults = append(vtb.validationResults, validation)

	if validation.Valid {
		vtb.targets = append(vtb.targets, *validation.Target)
	} else if vtb.autoFix && len(validation.Suggestions) > 0 {
		// Try the first suggestion
		suggestionValidation := ValidateTargetString(validation.Suggestions[0], vtb.config)
		if suggestionValidation.Valid {
			vtb.targets = append(vtb.targets, *suggestionValidation.Target)
			vtb.validationResults[len(vtb.validationResults)-1] = suggestionValidation
		} else {
			vtb.errors = append(vtb.errors, fmt.Errorf("invalid target: %s", target))
		}
	} else {
		vtb.errors = append(vtb.errors, fmt.Errorf("invalid target: %s", target))
	}

	return vtb
}

// GetValidationResults returns all validation results
func (vtb *ValidatedTargetBuilder) GetValidationResults() []*ValidationResult {
	return vtb.validationResults
}

// GetValidationSummary returns a summary of validation results
func (vtb *ValidatedTargetBuilder) GetValidationSummary() *BatchValidationResult {
	result := &BatchValidationResult{
		Results: vtb.validationResults,
	}

	totalScore := 0
	for _, validation := range vtb.validationResults {
		if validation.Valid {
			result.ValidCount++
		} else {
			result.ErrorCount++
		}

		if len(validation.Warnings) > 0 {
			result.WarningCount++
		}

		totalScore += validation.Score
	}

	if len(vtb.validationResults) > 0 {
		result.AvgScore = float64(totalScore) / float64(len(vtb.validationResults))
	}

	return result
}

// ================================
// Target Suggestion Engine
// ================================

// SuggestionEngine provides intelligent target suggestions
type SuggestionEngine struct {
	config *ValidationConfig
}

// NewSuggestionEngine creates a new suggestion engine
func NewSuggestionEngine(config ...*ValidationConfig) *SuggestionEngine {
	cfg := DefaultValidationConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}
	return &SuggestionEngine{config: cfg}
}

// SuggestTargets suggests possible targets based on input
func (se *SuggestionEngine) SuggestTargets(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return []string{}
	}

	suggestions := []string{}

	// Email suggestions
	if strings.Contains(input, "@") || strings.Contains(input, ".") {
		suggestions = append(suggestions, se.suggestEmails(input)...)
	}

	// Platform-specific suggestions
	if strings.HasPrefix(input, "#") || strings.HasPrefix(input, "@") {
		suggestions = append(suggestions, se.suggestPlatformTargets(input)...)
	}

	// Generic suggestions
	if len(suggestions) == 0 {
		suggestions = append(suggestions, se.suggestGeneric(input)...)
	}

	// Limit and deduplicate
	return se.limitAndDedupe(suggestions)
}

// suggestEmails suggests email completions
func (se *SuggestionEngine) suggestEmails(input string) []string {
	suggestions := []string{}

	if !strings.Contains(input, "@") {
		// Suggest adding common domains
		suggestions = append(suggestions,
			input+"@gmail.com",
			input+"@example.com",
			input+"@company.com")
	} else {
		// Suggest domain completions
		parts := strings.Split(input, "@")
		if len(parts) == 2 {
			local, domain := parts[0], parts[1]
			if domain == "" || len(domain) < 4 {
				suggestions = append(suggestions,
					local+"@gmail.com",
					local+"@yahoo.com",
					local+"@outlook.com")
			}
		}
	}

	return suggestions
}

// suggestPlatformTargets suggests platform-specific targets
func (se *SuggestionEngine) suggestPlatformTargets(input string) []string {
	suggestions := []string{}

	if strings.HasPrefix(input, "#") {
		// Channel suggestions
		suggestions = append(suggestions,
			"slack:"+input,
			"feishu:"+input)
	} else if strings.HasPrefix(input, "@") {
		// User suggestions
		suggestions = append(suggestions,
			"slack:"+input,
			"feishu:"+input)
	}

	return suggestions
}

// suggestGeneric suggests generic target formats
func (se *SuggestionEngine) suggestGeneric(input string) []string {
	return []string{
		input + "@example.com",
		"slack:@" + input,
		"#" + input,
		"feishu:" + input,
	}
}

// limitAndDedupe limits suggestions and removes duplicates
func (se *SuggestionEngine) limitAndDedupe(suggestions []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, suggestion := range suggestions {
		if !seen[suggestion] && len(result) < se.config.MaxSuggestions {
			seen[suggestion] = true
			result = append(result, suggestion)
		}
	}

	return result
}
