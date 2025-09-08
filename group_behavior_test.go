package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// This file contains comprehensive tests for group behavior across all output formats.
// It complements the basic WithGroup test in custom_handler_test.go by providing
// extensive coverage of slog group semantics compliance.

func TestGroupBehaviorAllFormats(t *testing.T) {
	testCases := []struct {
		name         string
		format       OutputFormat
		expectJSON   bool
		expectText   bool
		expectCustom bool
	}{
		{"Custom Format", FormatCustom, false, false, true},
		{"JSON Format", FormatJSON, true, false, false},
		{"Text Format", FormatText, false, true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testGroupBehaviorForFormat(t, tc.format, tc.expectJSON, tc.expectText, tc.expectCustom)
		})
	}
}

func testGroupBehaviorForFormat(t *testing.T, format OutputFormat, expectJSON, expectText, expectCustom bool) {
	var buf bytes.Buffer

	// Create logger with specified format
	log, err := New(
		WithConsoleFormat(format),
		WithFile(false),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	// Redirect output to buffer for testing
	// We'll use a custom handler for testing to capture output
	cfg := DefaultConfig()
	cfg.Console.Format = format
	cfg.Console.Color = false // Disable color for testing
	cfg.File.Enabled = false

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}

	switch format {
	case FormatJSON:
		handler = slog.NewJSONHandler(&buf, opts)
	case FormatText:
		handler = slog.NewTextHandler(&buf, opts)
	case FormatCustom:
		h, err := newCustomHandler(&buf, cfg, &cfg.Console, opts)
		if err != nil {
			t.Fatalf("Failed to create custom handler: %v", err)
		}
		handler = h
	}

	testLogger := slog.New(handler)

	// Test 1: Without group
	testLogger.Info("Connection established", "host", "localhost", "port", 3306)
	output1 := buf.String()
	buf.Reset()

	// Test 2: With single group
	dbLogger := testLogger.WithGroup("Database")
	dbLogger.Info("Connection established", "host", "localhost", "port", 3306)
	output2 := buf.String()
	buf.Reset()

	// Test 3: With nested groups
	mysqlLogger := dbLogger.WithGroup("MySQL")
	mysqlLogger.Info("Connection established", "host", "localhost", "port", 3306)
	output3 := buf.String()
	buf.Reset()

	// Test 4: Error with group
	mysqlLogger.Error("Query failed", "error", "connection timeout", "retry_count", 3)
	output4 := buf.String()
	buf.Reset()

	// Test 5: WithAttrs + Group
	configuredLogger := mysqlLogger.With(
		slog.String("service", "user-service"),
		slog.String("version", "1.0.0"),
	)
	configuredLogger.Warn("Performance degraded", "response_time", "500ms", "threshold", "200ms")
	output5 := buf.String()

	// Verify message is not affected by groups in all cases
	t.Run("MessageNotAffectedByGroups", func(t *testing.T) {
		expectedMessage := "Connection established"

		if !strings.Contains(output1, expectedMessage) {
			t.Errorf("Output1 should contain message: %q", output1)
		}
		if !strings.Contains(output2, expectedMessage) {
			t.Errorf("Output2 should contain message: %q", output2)
		}
		if !strings.Contains(output3, expectedMessage) {
			t.Errorf("Output3 should contain message: %q", output3)
		}

		// Check that group names are not in the message part
		if strings.Contains(output2, "Database: Connection established") {
			t.Errorf("Message should not contain group prefix in output2: %q", output2)
		}
		if strings.Contains(output3, "Database.MySQL: Connection established") {
			t.Errorf("Message should not contain group prefix in output3: %q", output3)
		}
	})

	// Verify attributes are properly grouped
	t.Run("AttributesProperlyGrouped", func(t *testing.T) {
		if expectJSON {
			// JSON format: attributes should be nested in group objects
			verifyJSONGroupBehavior(t, output1, output2, output3, output4, output5)
		} else if expectText || expectCustom {
			// Text/Custom format: attribute keys should have group prefixes
			verifyKeyPrefixGroupBehavior(t, output1, output2, output3, output4, output5)
		}
	})
}

func verifyJSONGroupBehavior(t *testing.T, output1, output2, output3, output4, output5 string) {
	// Test 1: No groups - attributes at root level
	var data1 map[string]interface{}
	if err := json.Unmarshal([]byte(output1), &data1); err != nil {
		t.Fatalf("Failed to parse JSON output1: %v", err)
	}
	if data1["host"] != "localhost" {
		t.Errorf("Expected host=localhost at root level, got: %v", data1)
	}

	// Test 2: Single group - attributes under Database
	var data2 map[string]interface{}
	if err := json.Unmarshal([]byte(output2), &data2); err != nil {
		t.Fatalf("Failed to parse JSON output2: %v", err)
	}
	database, ok := data2["Database"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Database group in output2, got: %v", data2)
	}
	if database["host"] != "localhost" {
		t.Errorf("Expected host=localhost under Database group, got: %v", database)
	}

	// Test 3: Nested groups - attributes under Database.MySQL
	var data3 map[string]interface{}
	if err := json.Unmarshal([]byte(output3), &data3); err != nil {
		t.Fatalf("Failed to parse JSON output3: %v", err)
	}
	database3, ok := data3["Database"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Database group in output3, got: %v", data3)
	}
	mysql, ok := database3["MySQL"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected MySQL group under Database, got: %v", database3)
	}
	if mysql["host"] != "localhost" {
		t.Errorf("Expected host=localhost under Database.MySQL group, got: %v", mysql)
	}
}

func verifyKeyPrefixGroupBehavior(t *testing.T, output1, output2, output3, output4, output5 string) {
	// Test 1: No groups - plain attribute keys
	if !strings.Contains(output1, "host=localhost") {
		t.Errorf("Expected host=localhost in output1: %q", output1)
	}

	// Test 2: Single group - attributes with Database prefix
	if !strings.Contains(output2, "Database.host=localhost") {
		t.Errorf("Expected Database.host=localhost in output2: %q", output2)
	}
	if !strings.Contains(output2, "Database.port=3306") {
		t.Errorf("Expected Database.port=3306 in output2: %q", output2)
	}

	// Test 3: Nested groups - attributes with Database.MySQL prefix
	if !strings.Contains(output3, "Database.MySQL.host=localhost") {
		t.Errorf("Expected Database.MySQL.host=localhost in output3: %q", output3)
	}
	if !strings.Contains(output3, "Database.MySQL.port=3306") {
		t.Errorf("Expected Database.MySQL.port=3306 in output3: %q", output3)
	}

	// Test 4: Error attributes also properly grouped
	if !strings.Contains(output4, "Database.MySQL.error=") {
		t.Errorf("Expected Database.MySQL.error in output4: %q", output4)
	}
	if !strings.Contains(output4, "Database.MySQL.retry_count=3") {
		t.Errorf("Expected Database.MySQL.retry_count=3 in output4: %q", output4)
	}

	// Test 5: WithAttrs combined with groups
	if !strings.Contains(output5, "Database.MySQL.service=user-service") {
		t.Errorf("Expected Database.MySQL.service=user-service in output5: %q", output5)
	}
	if !strings.Contains(output5, "Database.MySQL.response_time=500ms") {
		t.Errorf("Expected Database.MySQL.response_time=500ms in output5: %q", output5)
	}
}

// Test backward compatibility - ensure existing group tests still pass
func TestGroupBackwardCompatibility(t *testing.T) {
	var buf bytes.Buffer

	cfg := DefaultConfig()
	outputCfg := &mockOutputConfig{
		format:    FormatCustom,
		color:     false,
		formatter: "{message} {attrs}",
	}

	handler, err := newCustomHandler(&buf, cfg, outputCfg, nil)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Add group to handler
	newHandler := handler.WithGroup("test_group")

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "test message",
	}
	record.AddAttrs(slog.String("key", "value"))

	err = newHandler.Handle(context.Background(), record)
	if err != nil {
		t.Fatalf("Handler.Handle failed: %v", err)
	}

	output := buf.String()

	// With proper slog behavior, message should not contain group prefix
	if !strings.Contains(output, "test message") {
		t.Errorf("Output doesn't contain original message: %q", output)
	}
	// But attributes should be prefixed with group
	if !strings.Contains(output, "test_group.key=value") {
		t.Errorf("Output doesn't contain grouped attribute: %q", output)
	}
}
