package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTokenFile(t *testing.T, dir, name, content string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	return path
}

func TestLoadTokenPrefersFlagFileOverEnvAndFlagToken(t *testing.T) {
	dir := t.TempDir()
	flagPath := writeTokenFile(t, dir, "flag.token", " flag-token\n", 0o600)
	envPath := writeTokenFile(t, dir, "env.token", "env-token\n", 0o600)
	t.Setenv(tokenFileEnv, envPath)

	got, err := loadToken("arg-token", flagPath)
	if err != nil {
		t.Fatalf("loadToken: %v", err)
	}
	if got != "flag-token" {
		t.Fatalf("token = %q, want flag-token", got)
	}
}

func TestLoadTokenUsesEnvFileBeforeFlagToken(t *testing.T) {
	dir := t.TempDir()
	envPath := writeTokenFile(t, dir, "env.token", "env-token\n", 0o600)
	t.Setenv(tokenFileEnv, envPath)

	got, err := loadToken("arg-token", "")
	if err != nil {
		t.Fatalf("loadToken: %v", err)
	}
	if got != "env-token" {
		t.Fatalf("token = %q, want env-token", got)
	}
}

func TestLoadTokenFallsBackToFlagToken(t *testing.T) {
	t.Setenv(tokenFileEnv, "")
	got, err := loadToken(" arg-token\n", "")
	if err != nil {
		t.Fatalf("loadToken: %v", err)
	}
	if got != "arg-token" {
		t.Fatalf("token = %q, want arg-token", got)
	}
}

func TestLoadTokenRejectsWorldReadableFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTokenFile(t, dir, "open.token", "secret", 0o644)
	if _, err := loadToken("", path); err == nil {
		t.Fatal("expected permissions error")
	}
}

func TestLoadTokenRejectsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTokenFile(t, dir, "empty.token", "\n", 0o600)
	if _, err := loadToken("", path); err == nil {
		t.Fatal("expected empty token error")
	}
}
