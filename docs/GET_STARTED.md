# GETTING STARTED WITH M3TAL

This guide will walk you through the installation and initial setup of the M3TAL ecosystem.

## 1. Prerequisites

Before proceeding, ensure you have the following installed on your system:

- **Docker Engine**: The containerization platform.
- **Docker Compose V2**: The orchestration tool for Docker.

You can verify your installation by running:

```bash
docker --version && docker compose version
```

## 2. Install M3TAL via APT

M3TAL is distributed via an APT repository for easy installation and updates.

```bash
# 1. Add the GPG signing key
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/public.key | sudo gpg --dearmor -o /usr/share/keyrings/m3tal-archive-keyring.gpg

# 2. Add the APT repository
echo "deb [signed-by=/usr/share/keyrings/m3tal-archive-keyring.gpg] https://jakej985-rgb.github.io/m3tal-apt-key stable main" | sudo tee /etc/apt/sources.list.d/m3tal.list

# 3. Install
sudo apt update && sudo apt install -y m3tal
```

## 3. Run the Configuration Wizard

The M3TAL configuration wizard will guide you through setting up essential environment variables.

```bash
sudo m3tal config wizard
```

You will be prompted with several questions. Here's what each prompt means:

*   **`DASHBOARD_PORT`**: The port on which the M3TAL dashboard will be accessible. The default is `8082`.
*   **`DASHBOARD_EXPOSE_MODE`**: Determines how the dashboard is exposed.
    *   `local` (default): Exposes the dashboard directly via a port on your host machine, ideal for LAN access or initial setup.
    *   `traefik`: Exposes the dashboard through Traefik (a reverse proxy), allowing access via a domain name. This requires Traefik to be running.
*   **`HTTP_PORT`**: The port on which the M3TAL API daemon will listen. The default is `8080`.
*   **`STATE_DIR`**: The directory where M3TAL stores its state, such as the SQLite database. The default is `./state`.
*   **`LOG_LEVEL`**: Sets the verbosity of M3TAL's logging. Options include `debug`, `info`, `warn`, `error`.
*   **`DASHBOARD_SECRET`**: A secret key used for securing the dashboard. **It is highly recommended to change this from the default.**
*   **`API_TOKEN`**: An authentication token for the M3TAL API. **It is highly recommended to change this from the default.**
*   **`ADMIN_PASSWORD`**: The password for accessing the M3TAL dashboard. **It is highly recommended to change this from the default.**
*   **`DOMAIN`**: The primary domain used for accessing services through Traefik. Defaults to `localhost`.
*   **`BASE_STORAGE_PATH`**: The base directory for storing various data types. Defaults to `./data`.
*   **`MEDIA_PATH`**: The directory for media storage.
*   **`CONFIG_PATH`**: The directory for configuration files.
*   **`DOWNLOADS_PATH`**: The directory for downloaded files.
*   **`PUID`**: The user ID for running containers. Typically your user's ID.
*   **`PGID`**: The group ID for running containers. Typically your user's group ID.
*   **`TZ`**: Your local timezone, used for consistent time logging.

## 4. Start the Routing Stack (Traefik)

This command initializes and starts the Docker Compose stacks defined in the `/docker/` directory. This includes Traefik, which acts as the primary reverse proxy for your services.

```bash
m3tal up
```

This command reads all `*-compose.yml` files located in the `/docker/` directory and starts the services defined within them using Docker Compose.

## 5. Launch the Desktop App or Headless Mode

M3TAL can be launched either as a dedicated Linux desktop window or in headless mode for remote home servers.

### On Linux Desktop
Simply run `m3tal` or open **M3TAL Control Center** from your desktop applications menu:

```bash
m3tal
```

This starts the embedded API server and opens the M3TAL control center in an isolated desktop window.

### On Headless Home Servers
If your server is headless without a graphical display:

```bash
m3tal headless
```

This starts the background API and serves the Web GUI on port `5050` (or pass a port like `m3tal headless 8080`).

## 6. Access the Web GUI (Remote or Local)

On your desktop, the window opens automatically. For headless home servers, open your browser and navigate to:

`http://<SERVER_IP>:5050`

(Replace `<SERVER_IP>` with your home server's LAN IP address.)

## 7. Log In

Upon first access, you will be presented with a login screen.

*   **Username:** `admin`
*   **Password:** The password you set for `ADMIN_PASSWORD` during the configuration wizard.

If you need to change the dashboard password after initial setup, use the following command:

```bash
sudo m3tal dashpass
```

This command will prompt you for the new password and update the `users.json` file.

## Filesystem Contract

M3TAL utilizes a structured filesystem for configuration and data storage.

| Path                      | Purpose                                                                                              |
| :------------------------ | :--------------------------------------------------------------------------------------------------- |
| `/etc/m3tal/.env`         | Primary environment configuration file. Managed by `m3tal config wizard` and `m3tal config set`.     |
| `/var/lib/m3tal/state.db` | SQLite state database for M3TAL. Auto-created and managed by the API daemon.                       |
| `/opt/m3tal/stack/`       | The canonical directory containing M3TAL's core Docker Compose files and Traefik configuration.    |
| `/docker`                 | A symbolic link pointing to `/opt/m3tal/stack/`. This is the user-facing path for managing stacks. |
| `/docker/users.json`      | Stores dashboard user credentials. Managed by `m3tal dashpass`.                                      |

## Port Map

M3TAL and its components utilize the following ports:

| Port   | Service            | Access                                     |
| :----- | :----------------- | :----------------------------------------- |
| 80     | Traefik (HTTP)     | Public (if exposed and in `traefik` mode)  |
| 8080   | M3TAL API daemon   | Host-local only                            |
| 8081   | Traefik Dashboard  | Host-local only (admin access to Traefik)  |
| 8082   | M3TAL Dashboard    | Direct port (local mode) or via Traefik    |

## Firewall Note

If you intend for Traefik to be accessible from outside your local network, ensure that port 80 (and potentially 443 for HTTPS) is open in your firewall. For example, using `ufw`:

```bash
sudo ufw allow 80
```

## Service Management

The M3TAL API daemon runs as a systemd service. You can manage and monitor it using `systemctl` and `journalctl`.

*   **Check status:**
    ```bash
    systemctl status m3tal-api
    ```
*   **View logs in real-time:**
    ```bash
    journalctl -u m3tal-api -f
    ```
*   **Restart the API daemon:**
    ```bash
    sudo systemctl restart m3tal-api
    ```