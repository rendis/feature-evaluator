package config

import "testing"

func TestLoadNormalizesOriginsAndHosts(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", " https://Console.Example.com ,https://console.example.com ")
	t.Setenv("EXTERNAL_API_ALLOW_HOSTS", " API.Example.com ,api.example.com ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if len(cfg.CORS.AllowOrigins) != 1 || cfg.CORS.AllowOrigins[0] != "https://console.example.com" {
		t.Fatalf("CORS.AllowOrigins = %#v, want normalized deduplicated origin", cfg.CORS.AllowOrigins)
	}
	if len(cfg.External.AllowHosts) != 1 || cfg.External.AllowHosts[0] != "api.example.com" {
		t.Fatalf("External.AllowHosts = %#v, want normalized deduplicated host", cfg.External.AllowHosts)
	}
}

func TestLoadRejectsInvalidCORSOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", "https://console.example.com/app")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadRejectsInvalidExternalAPIHost(t *testing.T) {
	t.Setenv("EXTERNAL_API_ALLOW_HOSTS", "https://api.example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}
