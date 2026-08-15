# trmnl-immich-stats

A [TRMNL](https://usetrmnl.com) plugin that shows storage and server
statistics for your [Immich](https://immich.app) server: photos, videos,
storage used, per-user usage, version, license status, and media stack
versions.

## How it works

```
┌──────────┐   polling   ┌────────────────────┐  x-api-key  ┌─────────┐
│ TRMNL    │ ──────────► │ trmnl-immich-stats │ ──────────► │ Immich  │
│ device   │ ◄────────── │ Go backend         │ ◄────────── │ server  │
└──────────┘     JSON    └────────────────────┘             └─────────┘
```

Immich requires an API key (`x-api-key` header) for every request, which the
TRMNL device cannot hold. This Go backend fetches and combines four Immich
endpoints into a single TRMNL-pollable payload:

- `GET /api/server/statistics` — photo/video counts and storage usage
  (total and per user).
- `GET /api/server/version` — server version.
- `GET /api/server/about` — build info, license flag, media stack versions.
- `GET /api/server/license` — license/product key data.

The `trmnl/` directory is a [trmnlp](https://github.com/owise1/trmnlp) plugin
project (Liquid templates + settings) that is pushed to your TRMNL plugin via
`trmnlp push`.

## Configuration

| Variable         | Required | Default | Description                     |
| ---------------- | -------- | ------- | ------------------------------- |
| `IMMICH_URL`     | yes      | —       | Base URL of your Immich server  |
| `IMMICH_API_KEY` | yes      | —       | Immich API key                  |
| `PORT`           | no       | `8080`  | HTTP listen port                |

## Run

```sh
# from source
go run .

# or with docker
docker compose up -d
```

## TRMNL plugin setup

1. Host this backend somewhere public (`https://<backend>/healthz` should
   return `ok`).
2. Create a new plugin in the TRMNL dashboard (or push via `trmnlp push`) with
   polling URL `https://<backend>/api/trmnl/stats`.
3. Set the **url** custom field to the public backend URL.
4. Set the refresh interval to 60 minutes (or lower) for fresh stats.

For local development, run `trmnlp serve` inside `trmnl/`.

## Development

```sh
task setup   # download Go modules
task dev     # run with air (auto-reload)
task test    # run unit tests
task build   # build the binary
```

## License

See [LICENSE](LICENSE).
