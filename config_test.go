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

	if cfg.Level != slog.LevelInfo {
		t.Errorf("Expected default level to be LevelInfo, got %v", cfg.Level)
	}
	if cfg.AddSource != false {
		t.Errorf("Expected default AddSource to be false, got %v", cfg.AddSource)
	}
	if cfg.TimeFormat != DefaultTimeFormat {
		t.Errorf("Expected default TimeFormat to be %s, got %s", DefaultTimeFormat, cfg.TimeFormat)
	}
	if cfg.Format != FormatCustom {
		t.Errorf("Expected default Format to be %s, got %s", FormatCustom, cfg.Format)
	}
	if cfg.Console != true {
		t.Errorf("Expected default Console to be true, got %v", cfg.Console)
	}
	if cfg.File != false {
		t.Errorf("Expected default File to be false, got %v", cfg.File)
	}
}

func TestOptions(t *testing.T) {
	cfg := DefaultConfig()

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

	WithFormat(FormatJSON)(cfg)
	if cfg.Format != FormatJSON {
		t.Errorf("Expected Format to be %s, got %s", FormatJSON, cfg.Format)
	}

	WithConsole(false)(cfg)
	if cfg.Console != false {
		t.Errorf("Expected Console to be false, got %v", cfg.Console)
	}

	WithColor(false)(cfg)
	if cfg.Color != false {
		t.Errorf("Expected Color to be false, got %v", cfg.Color)
	}

	WithFile(true)(cfg)
	if cfg.File != true {
		t.Errorf("Expected File to be true, got %v", cfg.File)
	}

	filePath := "test.log"
	WithFilePath(filePath)(cfg)
	if cfg.FilePath != filePath {
		t.Errorf("Expected FilePath to be %s, got %s", filePath, cfg.FilePath)
	}

	maxSize := 20
	WithMaxSizeMB(maxSize)(cfg)
	if cfg.MaxSizeMB != maxSize {
		t.Errorf("Expected MaxSizeMB to be %d, got %d", maxSize, cfg.MaxSizeMB)
	}

	retentionDays := 10
	WithRetentionDays(retentionDays)(cfg)
	if cfg.RetentionDays != retentionDays {
		t.Errorf("Expected RetentionDays to be %d, got %d", retentionDays, cfg.RetentionDays)
	}

	formatter := "{time} - {level}"
	WithFormatter(formatter)(cfg)
	if cfg.Formatter != formatter {
		t.Errorf("Expected Formatter to be %s, got %s", formatter, cfg.Formatter)
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
				Format:     FormatText,
				Console:    true,
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: &Config{
				Level:   slog.LevelDebug - 10,
				Console: true,
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: &Config{
				Level:   slog.LevelInfo,
				Format:  "invalid",
				Console: true,
			},
			wantErr: true,
		},
		{
			name: "file enabled without path",
			config: &Config{
				Level:  slog.LevelInfo,
				Format: FormatText,
				File:   true,
				// FilePath missing
				Console: false,
			},
			wantErr: true,
		},
		{
			name: "no destinations",
			config: &Config{
				Level:   slog.LevelInfo,
				Format:  FormatText,
				Console: false,
				File:    false,
			},
			wantErr: true,
		},
		{
			name: "valid file config",
			config: &Config{
				Level:     slog.LevelInfo,
				Format:    FormatText,
				File:      true,
				FilePath:  filepath.Join(os.TempDir(), "test.log"),
				MaxSizeMB: 5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
