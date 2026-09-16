package main

import (
	"crypto/subtle"
	"encoding/json"
	"encoding/xml"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

const (
	shortName      = "gocuts"
	maxSuggestions = 10
)

type server struct {
	cfg     *config
	baseURL string
	suggest bool
	list    bool
	key     string
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleHome)
	mux.HandleFunc("GET /search", s.handleSearch)
	if s.suggest {
		mux.HandleFunc("GET /suggest", s.handleSuggest)
	}
	mux.HandleFunc("GET /opensearch.xml", s.handleOpenSearch)
	if s.key == "" {
		return mux
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("key")), []byte(s.key)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	dest, ok := s.cfg.lookup(stripGoPrefix(q))
	if !ok {
		msg := template.HTMLEscapeString(q) + " is not a known shortcut."
		if s.list {
			msg += " Available shortcuts:"
		}
		s.renderShortcuts(w, r, http.StatusNotFound, "Unknown shortcut", msg)
		return
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

func (s *server) handleSuggest(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	names, dests := s.cfg.matches(stripGoPrefix(q), maxSuggestions)
	w.Header().Set("Content-Type", "application/x-suggestions+json")
	_ = json.NewEncoder(w).Encode([]any{q, names, dests})
}

func (s *server) handleOpenSearch(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL
	if base == "" {
		base = requestBase(r)
	}
	keyParam := ""
	if s.key != "" {
		keyParam = "&key=" + url.QueryEscape(s.key)
	}
	urls := []openSearchURL{
		{Type: "text/html", Method: "get", Template: base + "/search?q={searchTerms}" + keyParam},
	}
	if s.suggest {
		urls = append(urls, openSearchURL{
			Type:     "application/x-suggestions+json",
			Method:   "get",
			Template: base + "/suggest?q={searchTerms}" + keyParam,
		})
	}
	doc := openSearchDescription{
		ShortName:     shortName,
		Description:   "gocuts go-links",
		InputEncoding: "UTF-8",
		Urls:          urls,
	}
	w.Header().Set("Content-Type", "application/opensearchdescription+xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(doc)
}

type openSearchURL struct {
	XMLName  xml.Name `xml:"Url"`
	Type     string   `xml:"type,attr"`
	Method   string   `xml:"method,attr"`
	Template string   `xml:"template,attr"`
}

type openSearchDescription struct {
	XMLName       xml.Name        `xml:"http://a9.com/-/spec/opensearch/1.1/ OpenSearchDescription"`
	ShortName     string          `xml:"ShortName"`
	Description   string          `xml:"Description"`
	InputEncoding string          `xml:"InputEncoding"`
	Urls          []openSearchURL `xml:"Url"`
}

func (s *server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.renderShortcuts(w, r, http.StatusOK, shortName, "")
}

func (s *server) renderShortcuts(w http.ResponseWriter, r *http.Request, status int, title, message string) {
	base := s.baseURL
	if base == "" {
		base = requestBase(r)
	}
	var rows []shortcutRow
	if s.list {
		names, dests := s.cfg.matches("", len(s.cfg.names))
		for i, n := range names {
			rows = append(rows, shortcutRow{Name: n, URL: dests[i]})
		}
	}
	keySuffix := ""
	osddHref := "/opensearch.xml"
	if s.key != "" {
		escaped := url.QueryEscape(s.key)
		keySuffix = "&key=" + escaped
		osddHref += "?key=" + escaped
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = pageTmpl.Execute(w, pageData{
		Title:          title,
		Message:        message,
		Base:           base,
		Rows:           rows,
		ListHidden:     !s.list,
		SuggestEnabled: s.suggest,
		KeySuffix:      keySuffix,
		OsddHref:       osddHref,
	})
}

type shortcutRow struct {
	Name string
	URL  string
}

type pageData struct {
	Title          string
	Message        string
	Base           string
	Rows           []shortcutRow
	ListHidden     bool
	SuggestEnabled bool
	KeySuffix      string
	OsddHref       string
}

var pageTmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<link rel="search" type="application/opensearchdescription+xml" href="{{.OsddHref}}" title="gocuts">
<style>
body { font-family: system-ui, sans-serif; max-width: 42rem; margin: 2rem auto; padding: 0 1rem; color: #222; }
table { border-collapse: collapse; width: 100%; }
th, td { text-align: left; padding: 0.35rem 0.75rem; border-bottom: 1px solid #ddd; }
code { background: #f4f4f4; padding: 0.1rem 0.3rem; border-radius: 3px; }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
{{if .Message}}<p>{{.Message}}</p>{{end}}
<h2>Install</h2>
<p>This page publishes OpenSearch metadata (<a href="{{.OsddHref}}">opensearch.xml</a>), so your browser
can add gocuts as a search engine. Firefox: pick "Add gocuts" from the address-bar search icon while on this
page. Chrome: Settings &rarr; Search engine &rarr; Manage search engines &rarr; Add, using keyword
<code>go</code> and URL <code>{{.Base}}/search?q=%s{{.KeySuffix}}</code>.</p>
<p>With gocuts as your search engine, typing <code>go NAME</code> in the address bar redirects to NAME's
destination{{if .SuggestEnabled}}; a partial name shows suggestions{{end}}.</p>
{{if .ListHidden}}
<p>Shortcut listing is disabled on this deployment.</p>
{{else}}
<h2>Shortcuts</h2>
<table>
<tr><th>Name</th><th>Destination</th></tr>
{{range .Rows}}<tr><td><code>{{.Name}}</code></td><td><a href="{{.URL}}">{{.URL}}</a></td></tr>
{{end}}</table>
{{end}}
</body>
</html>
`))

func requestBase(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		scheme = p
	}
	return scheme + "://" + r.Host
}

func stripGoPrefix(q string) string {
	lower := strings.ToLower(q)
	if strings.HasPrefix(lower, "go ") {
		return strings.TrimSpace(q[3:])
	}
	return q
}
