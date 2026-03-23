package config

import "testing"

func TestLoadNormalizesOrigins(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", " https://Console.Example.com ,https://console.example.com ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if len(cfg.CORS.AllowOrigins) != 1 || cfg.CORS.AllowOrigins[0] != "https://console.example.com" {
		t.Fatalf("CORS.AllowOrigins = %#v, want normalized deduplicated origin", cfg.CORS.AllowOrigins)
	}
}

func TestLoadRejectsInvalidCORSOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", "https://console.example.com/app")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}
