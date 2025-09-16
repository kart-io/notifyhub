package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/notifiers"
)

func main() {
	fmt.Println("🎯 NotifyHub Enhanced Target Validation and Suggestions")
	fmt.Println("=" + strings.Repeat("=", 65))

	// ================================
	// 1. Basic Target Validation
	// ================================
	fmt.Println("\n1. 📝 Basic Target Validation")

	testTargets := []string{
		"user@example.com", // Valid email
		"user@gmail",       // Missing TLD
		"user@gmai.com",    // Typo in domain
		"admin",            // Incomplete target
		"@johndoe",         // Slack user format
		"#general",         // Channel format
		"slack:@alice",     // Platform-specific format
		"invalid@",         // Invalid email
		"",                 // Empty target
	}

	for _, target := range testTargets {
		result := client.ValidateTargetString(target)
		fmt.Printf("\nTarget: '%s'\n", target)
		fmt.Printf("  Valid: %t (Score: %d)\n", result.Valid, result.Score)

		if len(result.Errors) > 0 {
			fmt.Printf("  Errors: %v\n", result.Errors)
		}

		if len(result.Warnings) > 0 {
			fmt.Printf("  Warnings: %v\n", result.Warnings)
		}

		if len(result.Suggestions) > 0 {
			fmt.Printf("  Suggestions: %v\n", result.Suggestions)
		}
	}

	// ================================
	// 2. Batch Target Validation
	// ================================
	fmt.Println("\n\n2. 📦 Batch Target Validation")

	batchTargets := []string{
		"admin@company.com",
		"user@gmai.com",
		"support@exampl.com",
		"@devteam",
		"#general",
		"invalid-email",
	}

	batchResult := client.ValidateTargetStrings(batchTargets)
	fmt.Printf("\nBatch Validation Summary:\n")
	fmt.Printf("  Total: %d targets\n", len(batchTargets))
	fmt.Printf("  Valid: %d\n", batchResult.ValidCount)
	fmt.Printf("  Errors: %d\n", batchResult.ErrorCount)
	fmt.Printf("  Warnings: %d\n", batchResult.WarningCount)
	fmt.Printf("  Average Score: %.1f\n", batchResult.AvgScore)

	// Show detailed results
	fmt.Println("\nDetailed Results:")
	for i, result := range batchResult.Results {
		fmt.Printf("  [%d] %s: %t", i+1, batchTargets[i], result.Valid)
		if len(result.Suggestions) > 0 {
			fmt.Printf(" → Suggestions: %v", result.Suggestions)
		}
		fmt.Println()
	}

	// ================================
	// 3. Custom Validation Configuration
	// ================================
	fmt.Println("\n\n3. ⚙️ Custom Validation Configuration")

	// Strict configuration
	strictConfig := &client.ValidationConfig{
		StrictEmailValidation: true,
		AllowLocalEmails:      false,
		RequireTLD:            true,
		MaxSuggestions:        2,
		EnableSpellCheck:      true,
	}

	// Permissive configuration
	permissiveConfig := &client.ValidationConfig{
		StrictEmailValidation: false,
		AllowLocalEmails:      true,
		RequireTLD:            false,
		MaxSuggestions:        5,
		EnableSpellCheck:      false,
	}

	testEmail := "admin@localhost"

	strictResult := client.ValidateTargetString(testEmail, strictConfig)
	permissiveResult := client.ValidateTargetString(testEmail, permissiveConfig)

	fmt.Printf("Target: %s\n", testEmail)
	fmt.Printf("Strict Config:     Valid=%t, Score=%d, Errors=%v\n",
		strictResult.Valid, strictResult.Score, strictResult.Errors)
	fmt.Printf("Permissive Config: Valid=%t, Score=%d, Warnings=%v\n",
		permissiveResult.Valid, permissiveResult.Score, permissiveResult.Warnings)

	// ================================
	// 4. Enhanced Target Builder with Validation
	// ================================
	fmt.Println("\n\n4. 🏗️ Enhanced Target Builder with Validation")

	// Builder without auto-fix
	fmt.Println("Without Auto-fix:")
	builder1 := client.NewValidatedTargetBuilder()
	builder1.AddTarget("admin@company.com"). // Valid
							AddTarget("user@gmai.com").  // Typo
							AddTarget("support@exampl"). // Incomplete
							AddTarget("@devops")         // Slack user

	targets1, err := builder1.Build()
	if err != nil {
		fmt.Printf("  Build failed: %v\n", err)
	} else {
		fmt.Printf("  Built %d valid targets\n", len(targets1))
	}

	summary1 := builder1.GetValidationSummary()
	fmt.Printf("  Validation: %d valid, %d errors, %.1f avg score\n",
		summary1.ValidCount, summary1.ErrorCount, summary1.AvgScore)

	// Builder with auto-fix
	fmt.Println("\nWith Auto-fix:")
	builder2 := client.NewValidatedTargetBuilder().WithAutoFix(true)
	builder2.AddTarget("admin@company.com"). // Valid
							AddTarget("user@gmai.com").      // Typo - will be auto-fixed
							AddTarget("support@exampl.com"). // Valid after typo fix
							AddTarget("@devops")             // Will try suggestions

	targets2, err := builder2.Build()
	if err != nil {
		fmt.Printf("  Build failed: %v\n", err)
	} else {
		fmt.Printf("  Built %d valid targets\n", len(targets2))
		for i, target := range targets2 {
			fmt.Printf("    [%d] %s:%s@%s\n", i+1, target.Type, target.Value, target.Platform)
		}
	}

	summary2 := builder2.GetValidationSummary()
	fmt.Printf("  Validation: %d valid, %d errors, %.1f avg score\n",
		summary2.ValidCount, summary2.ErrorCount, summary2.AvgScore)

	// ================================
	// 5. Target Suggestion Engine
	// ================================
	fmt.Println("\n\n5. 💡 Target Suggestion Engine")

	engine := client.NewSuggestionEngine()

	testInputs := []string{
		"john",       // Generic name
		"admin@",     // Incomplete email
		"@user",      // User format
		"#team",      // Channel format
		"support@gm", // Incomplete domain
	}

	for _, input := range testInputs {
		suggestions := engine.SuggestTargets(input)
		fmt.Printf("Input: '%s' → Suggestions: %v\n", input, suggestions)
	}

	// ================================
	// 6. Integration with Message Builder
	// ================================
	fmt.Println("\n\n6. 🔗 Integration with Message Builder")

	// Using validated targets in message builder
	validatedBuilder := client.NewValidatedTargetBuilder().WithAutoFix(true)
	validatedBuilder.AddTarget("team@company.com").
		AddTarget("admin@gmai.com"). // Will be auto-fixed
		AddTarget("@devops").        // Will try to fix
		AddTarget("#alerts")         // Will suggest platform

	validTargets, err := validatedBuilder.Build()
	if err != nil {
		log.Printf("Failed to build valid targets: %v", err)
		// Fallback to manual targets
		validTargets = []notifiers.Target{
			client.MustEmailTarget("fallback@company.com"),
		}
	}

	// Use validated targets in message
	message := client.NewAlert("Validated Alert", "Using validated targets").
		AsReliableSend().
		Build()

	// Add targets manually since we don't have a Targets() method that accepts []Target
	for _, target := range validTargets {
		message.Targets = append(message.Targets, target)
	}

	fmt.Printf("Message created with %d validated targets\n", len(message.Targets))

	// Show validation results
	validationResults := validatedBuilder.GetValidationResults()
	fmt.Println("\nValidation Details:")
	for i, result := range validationResults {
		resultJSON, _ := json.MarshalIndent(result, "  ", "  ")
		fmt.Printf("  [%d] %s\n", i+1, string(resultJSON))
	}

	// ================================
	// 7. Real-world Usage Examples
	// ================================
	fmt.Println("\n\n7. 🌍 Real-world Usage Examples")

	// Example: Processing user input from a form
	fmt.Println("Example: Processing user input from a form")
	userInputs := []string{
		"john.doe@company.com", // Perfect
		"jane@gmai.com",        // Typo
		"bob@localhost",        // Local email
		"@alice",               // Slack user
		"invalid-email",        // Invalid
	}

	processedTargets := []notifiers.Target{}
	for _, input := range userInputs {
		result := client.ValidateTargetString(input)
		if result.Valid {
			processedTargets = append(processedTargets, *result.Target)
			fmt.Printf("  ✅ Accepted: %s\n", input)
		} else {
			fmt.Printf("  ❌ Rejected: %s", input)
			if len(result.Suggestions) > 0 {
				fmt.Printf(" → Did you mean: %s?", result.Suggestions[0])
			}
			fmt.Println()
		}
	}

	fmt.Printf("\nProcessed %d valid targets from %d inputs\n",
		len(processedTargets), len(userInputs))

	// Example: Bulk email validation with reporting
	fmt.Println("\nExample: Bulk email validation with reporting")
	bulkEmails := []string{
		"user1@company.com",
		"user2@gmai.com",
		"user3@yahoo.co",
		"user4@outlook.com",
		"user5@exampl.com",
		"invalid-format",
	}

	bulkResult := client.ValidateTargetStrings(bulkEmails)

	fmt.Printf("Bulk Validation Report:\n")
	fmt.Printf("  📊 Total Emails: %d\n", len(bulkEmails))
	fmt.Printf("  ✅ Valid: %d (%.1f%%)\n",
		bulkResult.ValidCount,
		float64(bulkResult.ValidCount)/float64(len(bulkEmails))*100)
	fmt.Printf("  ❌ Invalid: %d (%.1f%%)\n",
		bulkResult.ErrorCount,
		float64(bulkResult.ErrorCount)/float64(len(bulkEmails))*100)
	fmt.Printf("  ⚠️  With Warnings: %d\n", bulkResult.WarningCount)
	fmt.Printf("  📈 Average Score: %.1f/100\n", bulkResult.AvgScore)

	fmt.Println("\n🎉 Target Validation and Suggestions Demo Complete!")
}
