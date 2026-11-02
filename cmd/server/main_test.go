package main

import "testing"

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := loadConfig(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.port != "8080" || cfg.maxGames != 200 || cfg.publicURL != "" {
		t.Errorf("got %+v", cfg)
	}
}

func TestLoadConfigValues(t *testing.T) {
	cfg, err := loadConfig(env(map[string]string{
		"PORT":       "9000",
		"PUBLIC_URL": "https://game.example.com/",
		"MAX_GAMES":  "5",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.port != "9000" || cfg.maxGames != 5 || cfg.publicURL != "https://game.example.com" {
		t.Errorf("got %+v", cfg)
	}
}

func TestLoadConfigRejectsBadValues(t *testing.T) {
	tests := []map[string]string{
		{"PUBLIC_URL": "game.example.com"},
		{"PUBLIC_URL": "ftp://game.example.com"},
		{"MAX_GAMES": "0"},
		{"MAX_GAMES": "many"},
		{"PORT": "not-a-port"},
	}
	for _, values := range tests {
		if _, err := loadConfig(env(values)); err == nil {
			t.Errorf("%v: expected an error", values)
		}
	}
}
