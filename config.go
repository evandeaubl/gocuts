package main

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type shortcutDef struct {
	URL string `toml:"url"`
}

type fileConfig struct {
	Gocuts map[string]shortcutDef `toml:"gocuts"`
}

type config struct {
	names []string
	urls  map[string]string
}

func loadConfig(path string) (*config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg := &config{urls: make(map[string]string)}
	for name, def := range fc.Gocuts {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		u := strings.TrimSpace(def.URL)
		if err := validateURL(u); err != nil {
			fmt.Fprintf(os.Stderr, "gocuts: skipping shortcut %q: %v\n", name, err)
			continue
		}
		if _, dup := cfg.urls[name]; dup {
			fmt.Fprintf(os.Stderr, "gocuts: duplicate shortcut %q skipped\n", name)
			continue
		}
		cfg.urls[name] = u
		cfg.names = append(cfg.names, name)
	}
	sort.Strings(cfg.names)
	if len(cfg.names) == 0 {
		return nil, fmt.Errorf("config %s contains no valid shortcuts", path)
	}
	return cfg, nil
}

func validateURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("missing url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", raw, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("url %q must be an absolute http(s) URL", raw)
	}
	return nil
}

func (c *config) lookup(name string) (string, bool) {
	u, ok := c.urls[strings.ToLower(strings.TrimSpace(name))]
	return u, ok
}

func (c *config) matches(prefix string, limit int) ([]string, []string) {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	names := []string{}
	dests := []string{}
	for _, n := range c.names {
		if len(names) >= limit {
			break
		}
		if strings.HasPrefix(n, prefix) {
			names = append(names, n)
			dests = append(dests, c.urls[n])
		}
	}
	return names, dests
}
