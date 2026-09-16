# gocuts

A lightweight "go links" server written in Go. Configure it as a browser search
engine, type `go NAME` in the address bar, and get redirected to NAME's
destination — e.g. `go gmail` takes you to `https://mail.google.com`.

## Features

- **Go links via TOML config** — shortcuts are defined as
  `gocuts.$SHORTCUT_NAME.url` entries in a simple TOML file.
- **OpenSearch integration** — implements a search submission endpoint and a
  search suggestions endpoint, plus a discovery page with the required
  OpenSearch metadata, so browsers can add gocuts as a search engine.
- **Search suggestions** — while you type a partial name, the browser's
  suggestion dropdown lists matching shortcut names (with their destinations).
- **Works as a keyword or default search engine** — the optional leading `go `
  is stripped from queries, so `go gmail` works whether `go` is a keyword or
  gocuts is your default engine.
- **Privacy toggles** — suggestions and the public shortcut listing can each be
  disabled for deployments that consider them too revealing.
- **Small deployment** — static binary, two-stage Docker build based on
  distroless.

## Configuration

Shortcuts live in a TOML file. Dotted keys and tables are both valid:

```toml
gocuts.gmail.url = "https://mail.google.com"
gocuts.gh.url = "https://github.com"

[gocuts.docs]
url = "https://docs.google.com"
```

Names are case-insensitive; entries with missing or non-http(s) URLs are
skipped with a warning at startup. The config file path is required — supply it
with the `-config` flag or the `GOCUTS_CONFIG` environment variable.

### Options

| Flag          | Environment variable | Default | Description                                                        |
| ------------- | -------------------- | ------- | ------------------------------------------------------------------ |
| `-config`     | `GOCUTS_CONFIG`      | —       | Path to the TOML shortcuts file (required)                         |
| `-addr`       | `GOCUTS_ADDR`        | `:8080` | Listen address                                                     |
| `-base-url`   | `GOCUTS_BASE_URL`    | derived | External base URL used in OpenSearch templates                     |
| `-suggest`    | `GOCUTS_SUGGEST`     | `true`  | Serve `/suggest` and advertise it in `opensearch.xml`              |
| `-list`       | `GOCUTS_LIST`        | `true`  | List shortcuts on the home page and unknown-shortcut error pages   |

Boolean environment variables accept `true`/`false` (or `1`/`0`). By default
the OpenSearch templates are built from each request's `Host` header (honoring
`X-Forwarded-Proto`); set `-base-url` when running behind a proxy with an
external hostname.

## Running

### Local binary

```sh
go build -o gocuts .
./gocuts -config gocuts.toml
```

Then visit `http://localhost:8080/`.

### Container

Images are published on Docker Hub as `evandeaubl/gocuts`. The `latest` tag
tracks the most recent build, and each release version is tagged with the plain
version number (e.g. `1.2.3`):

```sh
docker pull evandeaubl/gocuts:latest
docker pull evandeaubl/gocuts:1.2.3
```

Run it with your config mounted at `/gocuts.toml` (the image's default
`GOCUTS_CONFIG`):

```sh
docker run -d --name gocuts -p 8080:8080 \
  -v /path/to/gocuts.toml:/gocuts.toml:ro \
  evandeaubl/gocuts
```

Extra flags can be appended to `docker run` (the config default comes from the
image environment, so it still applies):

```sh
docker run -d -p 8080:8080 \
  -v /path/to/gocuts.toml:/gocuts.toml:ro \
  evandeaubl/gocuts -suggest=false -list=false
```

Or configure entirely through the environment:

```sh
docker run -d -p 8080:8080 \
  -e GOCUTS_CONFIG=/gocuts.toml \
  -v /path/to/gocuts.toml:/gocuts.toml:ro \
  evandeaubl/gocuts
```

## Browser setup

Open the server's home page (`http://localhost:8080/`) in your browser:

- **Firefox** — use the address-bar search icon to "Add gocuts"; Firefox picks
  it up from the page's OpenSearch metadata.
- **Chrome** — Settings → Search engine → Manage search engines → Site search
  → Add, using keyword `go` and URL `http://localhost:8080/search?q=%s`.

With gocuts selected as the search engine (or via the `go` keyword), typing
`go gmail` redirects to the gmail shortcut's URL, and a partial name offers
completions from `/suggest`.

## Endpoints

| Endpoint          | Description                                                        |
| ----------------- | ------------------------------------------------------------------ |
| `GET /`           | Home page with OpenSearch discovery metadata and shortcut listing  |
| `GET /search`     | `q` = shortcut name; 302 redirect to its destination               |
| `GET /suggest`    | `q` = partial name; OpenSearch suggestions JSON                    |
| `GET /opensearch.xml` | OpenSearch description document (search + suggestions templates) |

## Building

From source:

```sh
go build .
```

Container image (two-stage build producing a distroless, non-root image that
listens on port 8080):

```sh
docker build -t evandeaubl/gocuts:latest .
```

CI (see `.github/workflows/build.yml`) runs tests and lint on every push and
pull request, builds multi-arch (amd64/arm64) images, and publishes them to
Docker Hub automatically: pushes to `main` update `latest`, and pushing a tag
like `1.2.3` publishes that release version.

For a release version, tag the same build accordingly:

```sh
docker tag evandeaubl/gocuts:latest evandeaubl/gocuts:1.2.3
docker push evandeaubl/gocuts:latest
docker push evandeaubl/gocuts:1.2.3
```
