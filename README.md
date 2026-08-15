# trmnl-immich-stats

A [TRMNL](https://usetrmnl.com) plugin that shows storage and server
statistics for your [Immich](https://immich.app) server: photos, videos,
storage used, per-user usage, version, license status, and media stack
versions.

## How it works

This plugin polls your Immich server **directly** — no backend to host:

```
┌──────────┐   polling (x-api-key)   ┌─────────┐
│ TRMNL    │ ──────────────────────► │ Immich  │
│ device   │ ◄────────────────────── │ server  │
└──────────┘       JSON + transform  └─────────┘
```

The plugin polls four Immich read endpoints (statistics, version, about,
license) using an API key you store in a custom field. A small
[trmnlp](https://github.com/owise1/trmnlp) transform script
(`trmnl/src/transform.py`) merges the four responses into one payload the
Liquid templates render — including human-readable storage sizes.

## TRMNL plugin setup

1. Create a new plugin in the TRMNL dashboard (or push via `trmnlp push`) with
   polling URL `https://your-immich.example.com/api/server/statistics` — the
   transform handles the rest.
2. Set the custom fields:
   - **url** — the base URL of your Immich server (required).
   - **api_key** — an Immich API key (required). Create one in Immich under
     Account Settings > API Keys.
3. Set the refresh interval to 60 minutes (or lower) for fresh stats.

## Development

```sh
cd trmnl
# validate the transform against a sample payload
python3 src/transform.py < test/fixture.json

# preview the plugin (requires trmnlp with transform support, Ruby >= 4.0)
trmnlp serve
```

## Security notes

- Your Immich API key is stored in the TRMNL plugin custom fields and is sent
  as the `x-api-key` header on every poll. Only use a read-only API key.
- Transform scripts run by default when you clone or pull a plugin. This
  plugin's `src/transform.py` is safe, but review any third-party plugin's
  transform script before serving it — or set `transform_runtime: disabled`
  in `.trmnlp.yml`.

## License

See [LICENSE](LICENSE).
