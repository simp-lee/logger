package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check basic defaults
	if cfg.Level != slog.LevelInfo {
		t.Errorf("Expected default level to be LevelInfo, got %v", cfg.Level)
	}
	if cfg.AddSource != false {
		t.Errorf("Expected default AddSource to be false, got %v", cfg.AddSource)
	}
	if cfg.TimeFormat != DefaultTimeFormat {
		t.Errorf("Expected default TimeFormat to be %s, got %s", DefaultTimeFormat, cfg.TimeFormat)
	}

	// Check console defaults
	if !cfg.Console.Enabled {
		t.Errorf("Expected default Console.Enabled to be true, got %v", cfg.Console.Enabled)
	}
	if !cfg.Console.Color {
		t.Errorf("Expected default Console.Color to be true, got %v", cfg.Console.Color)
	}
	if cfg.Console.Format != FormatCustom {
		t.Errorf("Expected default Format to be %s, got %s", FormatCustom, cfg.Console.Format)
	}
	if cfg.Console.Formatter != DefaultFormatter {
		t.Errorf("Expected default Formatter to be %s, got %s", DefaultFormatter, cfg.Console.Formatter)
	}

	// Check file defaults
	if cfg.File.Enabled {
		t.Errorf("Expected default File to be false, got %v", cfg.File)
	}
	if cfg.File.Path != "" {
		t.Errorf("Expected default File.Path to be empty, got %s", cfg.File.Path)
	}
	if cfg.File.Format != FormatCustom {
		t.Errorf("Expected default Format to be %s, got %s", FormatCustom, cfg.File.Format)
	}
	if cfg.File.Formatter != DefaultFormatter {
		t.Errorf("Expected default Formatter to be %s, got %s", DefaultFormatter, cfg.File.Formatter)
	}
	if cfg.File.MaxSizeMB != DefaultMaxSizeMB {
		t.Errorf("Expected default MaxSizeMB to be %d, got %d", DefaultMaxSizeMB, cfg.File.MaxSizeMB)
	}
	if cfg.File.RetentionDays != DefaultRetentionDays {
		t.Errorf("Expected default RetentionDays to be %d, got %d", DefaultRetentionDays, cfg.File.RetentionDays)
	}
}

func TestOptions(t *testing.T) {
	cfg := DefaultConfig()

	// Check basic defaults
	WithLevel(slog.LevelDebug)(cfg)
	if cfg.Level != slog.LevelDebug {
		t.Errorf("Expected level to be LevelDebug, got %v", cfg.Level)
	}

	WithAddSource(true)(cfg)
	if cfg.AddSource != true {
		t.Errorf("Expected AddSource to be true, got %v", cfg.AddSource)
	}

	customFormat := "2006-01-02"
	WithTimeFormat(customFormat)(cfg)
	if cfg.TimeFormat != customFormat {
		t.Errorf("Expected TimeFormat to be %s, got %s", customFormat, cfg.TimeFormat)
	}

	// Check console defaults
	WithConsole(false)(cfg)
	if cfg.Console.Enabled != false {
		t.Errorf("Expected Console.Enabled to be false, got %v", cfg.Console.Enabled)
	}

	WithConsoleColor(false)(cfg)
	if cfg.Console.Color != false {
		t.Errorf("Expected Console.Color to be false, got %v", cfg.Console.Color)
	}

	WithConsoleFormat(FormatJSON)(cfg)
	if cfg.Console.Format != FormatJSON {
		t.Errorf("Expected Format to be %s, got %s", FormatJSON, cfg.Console.Format)
	}

	consoleFormatter := "{time} - {level}"
	WithConsoleFormatter(consoleFormatter)(cfg)
	if cfg.Console.Format != FormatCustom {
		t.Errorf("Expected Console.Format to be %s after setting formatter, got %s", FormatCustom, cfg.Console.Format)
	}
	if cfg.Console.Formatter != consoleFormatter {
		t.Errorf("Expected Formatter to be %s, got %s", consoleFormatter, cfg.Console.Formatter)
	}

	// Check file defaults
	WithFile(true)(cfg)
	if cfg.File.Enabled != true {
		t.Errorf("Expected File to be true, got %v", cfg.File.Enabled)
	}

	filePath := "test.log"
	WithFilePath(filePath)(cfg)
	if cfg.File.Path != filePath {
		t.Errorf("Expected FilePath to be %s, got %s", filePath, cfg.File.Path)
	}
	if !cfg.File.Enabled {
		t.Errorf("Expected File.Enabled to be true after setting path, got false")
	}

	WithFileFormat(FormatJSON)(cfg)
	if cfg.File.Format != FormatJSON {
		t.Errorf("Expected Format to be %s, got %s", FormatJSON, cfg.File.Format)
	}

	formatter := "{time} - {level}"
	WithFileFormatter(formatter)(cfg)
	if cfg.File.Format != FormatCustom {
		t.Errorf("Expected File.Format to be %s after setting formatter, got %s", FormatCustom, cfg.File.Format)
	}
	if cfg.File.Formatter != formatter {
		t.Errorf("Expected Formatter to be %s, got %s", formatter, cfg.File.Formatter)
	}

	maxSize := 20
	WithMaxSizeMB(maxSize)(cfg)
	if cfg.File.MaxSizeMB != maxSize {
		t.Errorf("Expected MaxSizeMB to be %d, got %d", maxSize, cfg.File.MaxSizeMB)
	}

	retentionDays := 10
	WithRetentionDays(retentionDays)(cfg)
	if cfg.File.RetentionDays != retentionDays {
		t.Errorf("Expected RetentionDays to be %d, got %d", retentionDays, cfg.File.RetentionDays)
	}

	// Test compatibility methods
	WithFormat(FormatText)(cfg)
	if cfg.Console.Format != FormatText {
		t.Errorf("Expected Console.Format to be %s, got %s", FormatText, cfg.Console.Format)
	}
	if cfg.File.Format != FormatText {
		t.Errorf("Expected File.Format to be %s, got %s", FormatText, cfg.File.Format)
	}

	commonFormatter := "{level}: {message}"
	WithFormatter(commonFormatter)(cfg)
	if cfg.Console.Format != FormatCustom {
		t.Errorf("Expected Console.Format to be %s, got %s", FormatCustom, cfg.Console.Format)
	}
	if cfg.File.Format != FormatCustom {
		t.Errorf("Expected File.Format to be %s, got %s", FormatCustom, cfg.File.Format)
	}
	if cfg.Console.Formatter != commonFormatter {
		t.Errorf("Expected Console.Formatter to be %s, got %s", commonFormatter, cfg.Console.Formatter)
	}
	if cfg.File.Formatter != commonFormatter {
		t.Errorf("Expected File.Formatter to be %s, got %s", commonFormatter, cfg.File.Formatter)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Level:      slog.LevelInfo,
				TimeFormat: DefaultTimeFormat,
				TimeZone:   time.Local,
				Console: ConsoleConfig{
					Enabled: true,
					Format:  FormatText,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: &Config{
				Level:   slog.LevelDebug - 10,
				Console: ConsoleConfig{Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "invalid console format",
			config: &Config{
				Level: slog.LevelInfo,
				Console: ConsoleConfig{
					Enabled: true,
					Format:  "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid file format",
			config: &Config{
				Level: slog.LevelInfo,
				Console: ConsoleConfig{
					Enabled: true,
					Format:  FormatText,
				},
				File: FileConfig{
					Enabled: true,
					Format:  "invalid",
					Path:    "test.log",
				},
			},
			wantErr: true,
		},
		{
			name: "file enabled without path",
			config: &Config{
				Level: slog.LevelInfo,
				Console: ConsoleConfig{
					Enabled: false,
				},
				File: FileConfig{
					Enabled: true,
					// Path missing
					Format: FormatText,
				},
			},
			wantErr: true,
		},
		{
			name: "no destinations",
			config: &Config{
				Level: slog.LevelInfo,
				Console: ConsoleConfig{
					Enabled: false,
				},
				File: FileConfig{
					Enabled: false,
				},
			},
			wantErr: true,
		},
		{
			name: "valid file config",
			config: &Config{
				Level: slog.LevelInfo,
				Console: ConsoleConfig{
					Enabled: false,
				},
				File: FileConfig{
					Enabled:   true,
					Format:    FormatText,
					Path:      filepath.Join(os.TempDir(), "test.log"),
					MaxSizeMB: 5,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig(), name = %v, error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
