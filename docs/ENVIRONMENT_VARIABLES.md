# M3TAL Environment Variables Reference

This document provides a comprehensive reference for all environment variables used by the M3TAL ecosystem. These variables are crucial for configuring M3TAL's behavior, networking, storage, and security.

All M3TAL environment variables are read from the `/etc/m3tal/.env` file. This file is managed by the `m3tal config wizard` and can also be updated using `m3tal config set KEY value`. Both the M3TAL CLI and all Docker Compose stacks utilize this file via the `--env-file` option.

## Quick Reference Table

| Variable Name | Default Value | Description |
|---------------|---------------|-------------|
| **Core** | | |
| `STATE_DIR` | `./state` | Directory for storing M3TAL's state database and configuration. |
| `LOG_LEVEL` | `info` | Controls the verbosity of M3TAL's logs. |
| `DASHBOARD_SECRET` | `change_me_immediately` | Secret key for the M3TAL Dashboard. **Auto-generated on first `m3tal init`.** |
| `API_TOKEN` | `change_me_api_token` | API token for authentication. **Auto-generated on first `m3tal init`.** |
| `ADMIN_PASSWORD` | `admin_pass` | Password for the default admin user. |
| `TZ` | `America/Denver` | Timezone for M3TAL services. |
| `DEBUG_MODE` | `false` | Enables debug logging and potentially other development-specific features. |
| `METRICS_ENABLED` | `true` | Enables the collection and exposition of metrics. |
| **Auth** | | |
| `DASHBOARD_SECRET` | `change_me_immediately` | Secret key for the M3TAL Dashboard. **Auto-generated on first `m3tal init`.** |
| `API_TOKEN` | `change_me_api_token` | API token for authentication. **Auto-generated on first `m3tal init`.** |
| `ADMIN_PASSWORD` | `admin_pass` | Password for the default admin user. |
| **Network** | | |
| `HTTP_PORT` | `8080` | The port the M3TAL API daemon listens on. |
| `NETWORK_NAME` | `m3tal` | The name of the Docker network M3TAL services will join. |
| `LOCAL_IP` | `127.0.0.1` | The IP address M3TAL should bind to locally. |
| `DOMAIN` | `localhost` | The primary domain for M3TAL services. **Setting this enables `dash.DOMAIN` and `api.DOMAIN` routes.** |
| `TRAEFIK_WEB_PORT` | `80` | The host port Traefik listens on for HTTP traffic. |
| `TRAEFIK_WEBHTTPS_PORT` | `443` | The host port Traefik listens on for HTTPS traffic. |
| `TRAEFIK_DASHBOARD_PORT` | `8080` | The internal port Traefik uses for its own dashboard. |
| **Storage** | | |
| `BASE_STORAGE_PATH` | `./data` | **Controls where M3TAL media data is stored.** Defaults to `/mnt` in production deployments. |
| `MEDIA_PATH` | `./data/media` | Path within `BASE_STORAGE_PATH` for media files. |
| `CONFIG_PATH` | `./data/config` | Path within `BASE_STORAGE_PATH` for configuration files. |
| `DOWNLOADS_PATH` | `./data/downloads` | Path within `BASE_STORAGE_PATH` for downloaded files. |
| **Traefik** | | |
| `DOMAIN` | `localhost` | The primary domain for M3TAL services. **Setting this enables `dash.DOMAIN` and `api.DOMAIN` routes.** |
| `TRAEFIK_WEB_PORT` | `80` | The host port Traefik listens on for HTTP traffic. |
| `TRAEFIK_WEBHTTPS_PORT` | `443` | The host port Traefik listens on for HTTPS traffic. |
| `TRAEFIK_DASHBOARD_PORT` | `8080` | The internal port Traefik uses for its own dashboard. |
| **VPN** | | |
| `VPN_USER` | `user` | Username for the VPN connection. |
| `VPN_PASSWORD` | `password` | Password for the VPN connection. |
| **System** | | |
| `DASHBOARD_PORT` | `8082` | The port the M3TAL Dashboard container listens on internally. |
| `DASHBOARD_EXPOSE_MODE` | `local` | Controls how the dashboard is exposed: `local` (direct port) or `traefik` (via Traefik). |
| `PUID` | `1000` | The User ID to run containers with. |
| `PGID` | `1000` | The Group ID to run containers with. |

---

## Detailed Environment Variable Reference

All environment variables are read from `/etc/m3tal/.env`.

### Core

These variables control fundamental aspects of M3TAL's operation.

| Name | Description | Default Value | Example Value | Used By |
|---|---|---|---|---|
| `STATE_DIR` | Directory for storing M3TAL's state database and configuration. This is where `/var/lib/m3tal/state.db` will be located. | `./state` | `/mnt/m3tal/state` | CLI, API daemon |
| `LOG_LEVEL` | Controls the verbosity of M3TAL's logs. Accepted values are `debug`, `info`, `warn`, `error`. | `info` | `debug` | CLI, API daemon |
| `DASHBOARD_SECRET` | Secret key for the M3TAL Dashboard. This is used for session management and other security-related functions. **This variable is auto-generated on the first `m3tal init` and should not be set manually unless rotating.** | `change_me_immediately` | `your_super_secret_dashboard_key_12345` | Dashboard container |
| `API_TOKEN` | API token for authentication with the M3TAL API. This is used by the dashboard and potentially other clients to authenticate requests. **This variable is auto-generated on the first `m3tal init` and should not be set manually unless rotating.** | `change_me_api_token` | `your_api_token_abcde12345` | CLI, API daemon, Dashboard container |
| `ADMIN_PASSWORD` | Password for the default admin user of the M3TAL Dashboard. | `admin_pass` | `a_strong_new_password` | Dashboard container |
| `TZ` | Timezone for M3TAL services. This affects log timestamps and any time-sensitive operations. | `America/Denver` | `UTC` | Dashboard container, All containers |
| `DEBUG_MODE` | Enables debug logging and potentially other development-specific features within M3TAL services. | `false` | `true` | CLI, API daemon, Dashboard container |
| `METRICS_ENABLED` | Enables the collection and exposition of metrics from M3TAL services, typically for monitoring purposes. | `true` | `false` | API daemon |

### Auth

These variables are related to authentication and security.

| Name | Description | Default Value | Example Value | Used By |
|---|---|---|---|---|
| `DASHBOARD_SECRET` | Secret key for the M3TAL Dashboard. This is used for session management and other security-related functions. **This variable is auto-generated on the first `m3tal init` and should not be set manually unless rotating.** | `change_me_immediately` | `your_super_secret_dashboard_key_12345` | Dashboard container |
| `API_TOKEN` | API token for authentication with the M3TAL API. This is used by the dashboard and potentially other clients to authenticate requests. **This variable is auto-generated on the first `m3tal init` and should not be set manually unless rotating.** | `change_me_api_token` | `your_api_token_abcde12345` | CLI, API daemon, Dashboard container |
| `ADMIN_PASSWORD` | Password for the default admin user of the M3TAL Dashboard. | `admin_pass` | `a_strong_new_password` | Dashboard container |

### Network

These variables configure M3TAL's network settings and how services are exposed.

| Name | Description | Default Value | Example Value | Used By |
|---|---|---|---|---|
| `HTTP_PORT` | The port the M3TAL API daemon listens on. This is the internal port for API communication. | `8080` | `5050` | API daemon |
| `NETWORK_NAME` | The name of the Docker network M3TAL services will join. This allows services to communicate with each other. | `m3tal` | `m3tal-network` | CLI, API daemon, All containers |
| `LOCAL_IP` | The IP address M3TAL should bind to locally. This is primarily used for internal service communication. | `127.0.0.1` | `192.168.1.100` | API daemon |
| `DOMAIN` | The primary domain for M3TAL services. **Setting this variable enables Traefik routing rules for `dash.${DOMAIN}` and `api.${DOMAIN}`.** | `localhost` | `m3tal.example.com` | Traefik, API daemon |
| `TRAEFIK_WEB_PORT` | The host port Traefik listens on for HTTP traffic. This is the primary entry point for external HTTP requests. | `80` | `80` | Traefik |
| `TRAEFIK_WEBHTTPS_PORT` | The host port Traefik listens on for HTTPS traffic. | `443` | `443` | Traefik |
| `TRAEFIK_DASHBOARD_PORT` | The internal port Traefik uses for its own dashboard. This is accessible via `http://127.0.0.1:8080` (or the configured `TRAEFIK_WEB_PORT` if Traefik is exposed differently). | `8080` | `8081` | Traefik |

### Storage

These variables define the storage locations for M3TAL data.

| Name | Description | Default Value | Example Value | Used By |
|---|---|---|---|---|
| `BASE_STORAGE_PATH` | **This variable controls where M3TAL media data is stored.** It acts as the root for all other storage-related paths. In production deployments, this defaults to `/mnt`, not `./data` as seen in template configurations. | `./data` | `/mnt` | CLI, API daemon, Dashboard container |
| `MEDIA_PATH` | Path within `BASE_STORAGE_PATH` for media files. | `./data/media` | `/mnt/m3tal/media` | CLI, API daemon, Dashboard container |
| `CONFIG_PATH` | Path within `BASE_STORAGE_PATH` for configuration files. | `./data/config` | `/mnt/m3tal/config` | CLI, API daemon, Dashboard container |
| `DOWNLOADS_PATH` | Path within `BASE_STORAGE_PATH` for downloaded files. | `./data/downloads` | `/mnt/m3tal/downloads` | CLI, API daemon, Dashboard container |

### Traefik

These variables are specific to the Traefik reverse proxy configuration.

| Name | Description | Default Value | Example Value | Used By |
|---|---|---|---|---|
| `DOMAIN` | The primary domain for M3TAL services. **Setting this variable enables Traefik routing rules for `dash.${DOMAIN}` and `api.${DOMAIN}`.** | `localhost` | `m3tal.example.com` | Traefik, API daemon |
| `TRAEFIK_WEB_PORT` | The host port Traefik listens on for HTTP traffic. This is the primary entry point for external HTTP requests. | `80` | `80` | Traefik |
| `TRAEFIK_WEBHTTPS_PORT` | The host port Traefik listens on for HTTPS traffic. | `443` | `443` | Traefik |
| `TRAEFIK_DASHBOARD_PORT` | The internal port Traefik uses for its own dashboard. This is accessible via `http://127.0.0.1:8080` (or the configured `TRAEFIK_WEB_PORT` if Traefik is exposed differently). | `8080` | `8081` | Traefik |

### VPN

These variables are used for configuring VPN connections.

| Name | Description | Default Value | Example Value | Used By |
|---|---|---|---|---|
| `VPN_USER` | Username for the VPN connection. | `user` | `myvpnuser` | All VPN-enabled containers |
| `VPN_PASSWORD` | Password for the VPN connection. | `password` | `mysecretvpnpassword` | All VPN-enabled containers |

### System

These variables control system-level configurations and container behaviors.

| Name | Description | Default Value | Example Value | Used By |
|---|---|---|---|---|
| `DASHBOARD_PORT` | The port the M3TAL Dashboard container listens on internally. | `8082` | `8082` | Dashboard container, API daemon |
| `DASHBOARD_EXPOSE_MODE` | Controls how the dashboard is exposed. <br> **`local` (default):** Uses `m3tal-compose.local.yml` to add a direct port binding (`${DASHBOARD_PORT:-8082}:8082`). Access via `http://HOST_IP:8082` or `http://localhost:8082`. No Traefik required. <br> **`traefik`:** Uses `m3tal-compose.traefik.yml` and Traefik labels to route traffic to `dash.${DOMAIN}`. Requires Traefik to be running. | `local` | `traefik` | Dashboard container, CLI |
| `PUID` | The User ID to run containers with. This is important for file permissions within volumes. | `1000` | `1001` | Dashboard container, All containers |
| `PGID` | The Group ID to run containers with. This is important for file permissions within volumes. | `1000` | `1001` | Dashboard container, All containers |