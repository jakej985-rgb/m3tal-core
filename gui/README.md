# M3TAL Core Web GUI

The embedded dashboard is a React/Vite application. Its production build in `gui/dist` is embedded into `m3tal-api` with `go:embed` and served at `/gui/`.

## Development

```bash
cd gui
npm ci
npm run dev
```

The Vite dev server proxies API requests to the local Core API. Build production assets with:

```bash
cd gui
npm ci
npm run build
```

## Notes

- Do not add workload-specific plugin UI to Core; plugin UIs are served separately under `/plugins/<name>/`.
- API requests use the standard `/api/v2` envelope.
- Authentication uses `X-API-Token`, `Authorization: Bearer`, or the GUI's token query parameter.
