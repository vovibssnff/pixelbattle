package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfigReadsAdminAPITokenFromFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "admin_api_token")
	if err := os.WriteFile(tokenFile, []byte("  secret-token-from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ADMIN_API_TOKEN_FILE", tokenFile)
	t.Setenv("ADMIN_API_TOKEN", "")

	cfg, err := LoadConfig(filepath.Join(dir, "missing.env"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AdminAPIToken != "secret-token-from-file" {
		t.Fatalf("AdminAPIToken = %q, want file token", cfg.AdminAPIToken)
	}
}

func TestLoadConfigPrefersAdminAPITokenEnvOverFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "admin_api_token")
	if err := os.WriteFile(tokenFile, []byte("file-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ADMIN_API_TOKEN_FILE", tokenFile)
	t.Setenv("ADMIN_API_TOKEN", "env-token")

	cfg, err := LoadConfig(filepath.Join(dir, "missing.env"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AdminAPIToken != "env-token" {
		t.Fatalf("AdminAPIToken = %q, want env token", cfg.AdminAPIToken)
	}
}

func TestLoadConfigReadsSessionKeyFromFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	keyFile := filepath.Join(dir, "session_key")
	want := "0123456789abcdef0123456789abcdef"
	if err := os.WriteFile(keyFile, []byte(want+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SESSION_KEY_FILE", keyFile)
	t.Setenv("SESSION_KEY", "")

	cfg, err := LoadConfig(filepath.Join(dir, "missing.env"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.SessionKey != want {
		t.Fatalf("SessionKey = %q, want file key", cfg.SessionKey)
	}
}
