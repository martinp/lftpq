package site

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cfg := Config{
		Sites: []Site{Site{
			Name:         "foo",
			Dir:          "/site",
			MaxAge:       "24h",
			Patterns:     []string{"^match"},
			Filters:      []string{"^skip"},
			SkipSymlinks: true,
			Parser:       "show",
			LocalDir:     "/tmp/{{ .Name }}",
			Priorities:   []string{"important"},
			Deduplicate:  true,
		}},
	}
	if err := cfg.Load(); err != nil {
		t.Fatal(err)
	}

	site := cfg.Sites[0]
	if want := time.Duration(24) * time.Hour; site.maxAge != want {
		t.Errorf("Expected %s, got %s", want, site.maxAge)
	}
	if len(site.patterns) == 0 {
		t.Error("Expected non-empty patterns")
	}
	if len(site.filters) == 0 {
		t.Error("Expected non-empty filters")
	}
	if len(site.priorities) == 0 {
		t.Error("Expected non-empty priorities")
	}
	if site.localDir == nil {
		t.Error("Expected template to be compiled")
	}
	if site.parser == nil {
		t.Error("Expected parser to be set")
	}
}
