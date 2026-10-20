import fs from "fs";
import path from "path";
import { GoogleGenerativeAI } from "@google/generative-ai";

const genAI = new GoogleGenerativeAI(process.env.GOOGLE_AI_KEY);

// Ensure docs directory exists
fs.mkdirSync("docs", { recursive: true });

// --- Detect M3TAL repo structure ---
function detectStructure() {
  const structure = {
    hasDocker: fs.existsSync("deploy/stack/m3tal-compose.yml"),
    hasGo: fs.existsSync("go.mod"),
    hasCLI: fs.existsSync("cmd/m3tal/main.go"),
    hasTraefik: fs.existsSync("deploy/stack/traefik.yml"),
    hasRouting: fs.existsSync("deploy/stack/routing-compose.yml"),
    hasCloudflared: fs.existsSync("deploy/stack/cloudflared-config.yml"),
    services: [],
  };

  if (fs.existsSync("deploy")) {
    const sourceDirs = fs.readdirSync("deploy", { withFileTypes: true })
      .filter(d => d.isDirectory())
      .map(d => d.name);
    structure.services = sourceDirs;
  }

  return structure;
}

const structure = detectStructure();

// Load dynamic parsed context from previous parser steps
const dockerState = fs.existsSync("docs/docker.json")
  ? fs.readFileSync("docs/docker.json", "utf-8")
  : "[]";

const envState = fs.existsSync("docs/env.json")
  ? fs.readFileSync("docs/env.json", "utf-8")
  : "[]";

// Read real Traefik config files for DocCritic context grounding
const traefikStatic = fs.existsSync("deploy/stack/traefik.yml")
  ? fs.readFileSync("deploy/stack/traefik.yml", "utf-8")
  : "(not found)";

const traefikDynamic = fs.existsSync("deploy/stack/dynamic/api.yml")
  ? fs.readFileSync("deploy/stack/dynamic/api.yml", "utf-8")
  : "(not found)";

const dashComposeBase = fs.existsSync("deploy/stack/m3tal-compose.yml")
  ? fs.readFileSync("deploy/stack/m3tal-compose.yml", "utf-8")
  : "(not found)";

const dashComposeLocal = fs.existsSync("deploy/stack/m3tal-compose.local.yml")
  ? fs.readFileSync("deploy/stack/m3tal-compose.local.yml", "utf-8")
  : "(not found)";

const dashComposeTraefik = fs.existsSync("deploy/stack/m3tal-compose.traefik.yml")
  ? fs.readFileSync("deploy/stack/m3tal-compose.traefik.yml", "utf-8")
  : "(not found)";

// ─────────────────────────────────────────────────────────────────────────────
// SYSTEM CONTEXT — complete picture of the REAL M3TAL architecture
// This context is injected into every prompt so Gemini writes accurate docs.
// ─────────────────────────────────────────────────────────────────────────────
const context = `
## M3TAL System Architecture (Ground Truth)

### Components
- **CLI binary** (\`/usr/bin/m3tal\`): Unified Go binary installed via APT. Single entrypoint for all operations.
- **API daemon** (\`m3tal-api.service\`): Go binary running as a systemd service on port 8080. Manages Docker, state DB, and API routes.
- **Dashboard container** (\`m3tal-dashboard\`): Python/Flask container running internally on port 8082. Communicates with the API daemon at \`http://host.docker.internal:8080\`.
- **Traefik gateway** (\`routing-compose.yml\`): Reverse proxy container exposing services by domain name on port 80. Uses file provider for dynamic routing.
- **Cloudflared** (\`routing-compose.yml\`): Optional Cloudflare tunnel container for zero-config internet access.

### Filesystem Contract (MUST document explicitly)
| Path | Purpose |
|------|--------|
| \`/etc/m3tal/.env\` | Primary configuration file. Managed by \`m3tal config wizard\`. |
| \`/var/lib/m3tal/state.db\` | SQLite state database. Auto-created by the API daemon. |
| \`/opt/m3tal/stack/\` | Canonical stack directory. Contains compose files and Traefik config. |
| \`/docker\` | Symlink → \`/opt/m3tal/stack/\`. This is the user-facing path for all stack operations. |
| \`/docker/users.json\` | Dashboard credential store. Managed by \`m3tal dashpass\`. |

### Dashboard Access — TWO MODES (critical — MUST document both clearly)

The dashboard has two access modes, controlled by \`DASHBOARD_EXPOSE_MODE\` in \`/etc/m3tal/.env\`:

**Mode 1: local (default)**
- \`DASHBOARD_EXPOSE_MODE=local\`
- Uses override: \`m3tal-compose.local.yml\`
- Adds a direct port binding: \`\${DASHBOARD_PORT:-8082}:8082\`
- Access via: \`http://HOST_IP:8082\` or \`http://localhost:8082\`
- No Traefik required. Works out of the box on a home server.
- Best for: LAN-only setups, first-time users, local testing.

**Mode 2: traefik**
- \`DASHBOARD_EXPOSE_MODE=traefik\`
- Uses override: \`m3tal-compose.traefik.yml\`
- Adds Traefik labels so Traefik routes \`dash.\${DOMAIN}\` → dashboard on port 8082.
- Access via: \`http://dash.DOMAIN\` (Traefik must be running via \`m3tal up\`)
- Best for: domain-based setups, multiple services behind a reverse proxy.

Base dashboard compose (\`m3tal-compose.yml\`):
\`\`\`yaml
${dashComposeBase}
\`\`\`

Local mode override (\`m3tal-compose.local.yml\`):
\`\`\`yaml
${dashComposeLocal}
\`\`\`

Traefik mode override (\`m3tal-compose.traefik.yml\`):
\`\`\`yaml
${dashComposeTraefik}
\`\`\`

### Docker / Compose Runtime (MUST explain clearly)
- M3TAL uses **Docker Engine + Docker Compose V2** under the hood. These are hard dependencies.
- The \`m3tal up\` command runs \`docker compose\` across all \`*-compose.yml\` files found in \`/docker/\`.
- The \`m3tal dash up\` command specifically manages the dashboard container. It:
  1. Downloads the latest \`m3tal-compose.yml\`, \`m3tal-compose.local.yml\`, \`m3tal-compose.traefik.yml\` from GitHub.
  2. Reads \`DASHBOARD_EXPOSE_MODE\` from \`/etc/m3tal/.env\`.
  3. Starts the dashboard with the appropriate compose override file.
- User stacks live in \`/docker/\`. Adding a new stack means placing a \`*-compose.yml\` file there.

### Deployment Lifecycle — Day 2 Operations
Installing a new stack:
1. Place your compose file in \`/docker/my-stack-compose.yml\`
2. Ensure required variables are set in \`/etc/m3tal/.env\` (use \`m3tal config wizard\` or \`m3tal config set KEY value\`)
3. Run \`m3tal up\` to start all stacks.

### Traefik Routing Architecture
Traefik is deployed as a container via \`routing-compose.yml\`. It:
- Binds port 80 on the host as the HTTP entry point.
- Discovers services automatically via Docker labels.
- Loads dynamic config from \`/docker/dynamic/\` (file provider, hot-reload).
- Routes \`api.DOMAIN\` → \`http://host.docker.internal:8080\` (the Go API daemon) via \`dynamic/api.yml\`.
- Routes \`dash.DOMAIN\` → the dashboard container via Traefik labels in \`m3tal-compose.traefik.yml\` (only when \`DASHBOARD_EXPOSE_MODE=traefik\`).

Traefik static config (traefik.yml):
\`\`\`yaml
${traefikStatic}
\`\`\`

Dynamic routing example (dynamic/api.yml):
\`\`\`yaml
${traefikDynamic}
\`\`\`

### Service Management — systemd
- The API daemon is managed by systemd as \`m3tal-api.service\`.
- Commands: \`systemctl status m3tal-api\`, \`systemctl restart m3tal-api\`, \`journalctl -u m3tal-api -f\`

### Port Map
| Port | Service | Access |
|------|---------|--------|
| 80 | Traefik HTTP entry point | Public (traefik mode) |
| 8080 | M3TAL API daemon (Go) | Host-local |
| 8081 | Traefik dashboard | Host-local only |
| 8082 | M3TAL Dashboard | Direct port (local mode) or via Traefik (traefik mode) |

### APT Installation (ALWAYS include this exact block)
\`\`\`bash
# 1. Add the GPG signing key
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/public.key | sudo gpg --dearmor -o /usr/share/keyrings/m3tal-archive-keyring.gpg

# 2. Add the APT repository
echo "deb [signed-by=/usr/share/keyrings/m3tal-archive-keyring.gpg] https://jakej985-rgb.github.io/m3tal-apt-key stable main" | sudo tee /etc/apt/sources.list.d/m3tal.list

# 3. Install
sudo apt update && sudo apt install -y m3tal
\`\`\`

### Docker Services State JSON:
${dockerState}

### Environment Variables State JSON:
${envState}
`;

// ─────────────────────────────────────────────────────────────────────────────
// DOCUMENTATION TARGETS
// ─────────────────────────────────────────────────────────────────────────────
const targets = [
  {
    path: "README.md",
    name: "README",
    prompt: `You are DocSmith, the M3TAL Ecosystem Documentation Architect.
Generate the primary README.md based on the REAL system architecture documented below.

STRICT RULES:
- Title must be "M3TAL System Documentation".
- Rephrase the "Overview" section to be strictly direct and technical: "This document provides technical details and operational procedures for the M3TAL system." Avoid all marketing copy and terms like "cohesive", "robust", "ecosystem".
- NO placeholders. Every section must contain real, actionable content.
- MUST include a dedicated "Prerequisites" section at the very beginning of the document explicitly stating:
  "Docker Engine and Docker Compose V2 are strictly REQUIRED and must be installed prior to M3TAL installation and operation. M3TAL internally orchestrates Docker containers using Docker Engine and Docker Compose V2."
- MUST include the exact APT installation block from the context below. Do not invent a different install method.
- MUST include a "Deployment Lifecycle" section explaining:
  "M3TAL orchestrates Docker containers using Docker Compose V2. The \`m3tal up\` command is a wrapper around \`docker compose\` that operates on all \`*-compose.yml\` files located within the \`/docker/\` directory, effectively deploying each as an independent stack."
  - MUST explicitly clarify the relationship between \`/docker\` and \`/opt/m3tal/stack/\` to emphasize that \`/opt/m3tal/stack/\` is the canonical source of truth directory where all stack files reside, and that \`/docker\` is the user-facing symlink alias for all stack operations.
  - In the "Adding a New Stack" sub-section, MUST explicitly state:
    "To deploy a new Docker Compose stack:
    1. Place your Docker Compose file (e.g., \`my-stack-compose.yml\`) directly into the \`/docker/\` directory. This file will be automatically included by \`m3tal up\`."
- MUST include a "Dashboard Access" section clearly explaining BOTH modes:
  - **Local mode** (default, \`DASHBOARD_EXPOSE_MODE=local\`): Direct port binding at \`http://HOST_IP:8082\`. No Traefik needed. Explicitly state that a new user performing a default installation will access the dashboard directly via port 8082, and link this "direct access via port 8082" behavior to the default setting \`DASHBOARD_EXPOSE_MODE=local\`.
  - **Traefik mode** (\`DASHBOARD_EXPOSE_MODE=traefik\`): Domain routing at \`http://dash.DOMAIN\` via Traefik labels. Requires Traefik running.
  - Make it crystal clear that a new user using the default install gets access at port 8082 directly, NOT via a domain.
- MUST include a "Traefik Gateway" section explaining:
  "Traefik automatically discovers and routes traffic to Docker services by interpreting Traefik labels defined within their Docker Compose service definitions. Crucially, services are not exposed by Traefik by default and require \`traefik.enable=true\` along with other relevant labels to be discoverable and routable."
  - MUST explain the role of dynamic configuration files (such as \`dynamic/api.yml\`) in routing requests to services listening on host-local ports (specifically, routing \`api.DOMAIN\` to the Go API daemon on host-local port \`8080\` via \`http://host.docker.internal:8080\`).
  - \`dash.DOMAIN\` routes to the dashboard container
  - MUST show a concrete YAML example of how to expose a custom user service via Traefik labels. Show a snippet for a hypothetical \`my-app-compose.yml\` with labels like:
    \`\`\`yaml
    services:
      my-app:
        image: nginx:alpine
        labels:
          - "traefik.enable=true"
          - "traefik.http.routers.myapp.rule=Host(\`app.DOMAIN\`)\"
          - "traefik.http.services.myapp.loadbalancer.server.port=80"
          - "traefik.http.routers.myapp.entrypoints=web"
        networks:
          - proxy
    \`\`\`
- MUST include a "Service Management" section showing systemctl commands for m3tal-api.service
- MUST include a Quick Demo section explaining:
  - How to run \`m3tal dash up\` to start the dashboard container specifically.
  - Clarify that \`m3tal up\` orchestrates and deploys all other stacks in the directory, including any user-defined compose files.
- MUST include a "Port Map" section with exactly this table and note:
  "The following table lists the primary network ports utilized by the M3TAL system:

  | Port | Service | Access | Description |
  |------|---------|--------|-------------|
  | 80 | Traefik HTTP entry point | Public | The public-facing HTTP port for services exposed via Traefik. |
  | 8080 | M3TAL API daemon (Go) | Host-local | The internal port the M3TAL API daemon listens on. |
  | 8081 | Traefik dashboard | Host-local only | The internal Traefik dashboard port, accessible only from the host machine. |
  | 8082 | M3TAL Dashboard | Direct port (local mode) or via Traefik (traefik mode) | The port the M3TAL Dashboard container listens on internally. Access method depends on \`DASHBOARD_EXPOSE_MODE\`. |

  Note: These are the primary M3TAL-managed ports. User-added Docker Compose stacks may expose additional ports on their own, depending on their configurations."
- MUST include a "Firewall Considerations" section placed prominently near the Traefik Gateway or Installation section:
  "If you are using Traefik for public-facing access (i.e., \`DASHBOARD_EXPOSE_MODE=traefik\` or exposing other services via Traefik), ensure that host port \`80\` (and \`443\` if HTTPS is configured) is open in your firewall (e.g., \`ufw allow 80/tcp\`)."

${context}
`
  },
  {
    path: "docs/GET_STARTED.md",
    name: "Getting Started Guide",
    prompt: `You are DocSmith, the M3TAL Ecosystem Documentation Architect.
Generate docs/GET_STARTED.md — a complete, newbie-friendly setup guide for first-time users.

STRICT RULES:
- NO marketing copy. Write clear operational steps only.
- Step 1: Prerequisites — explicitly state: "Docker Engine and Docker Compose V2 must be installed."
  Show: \`docker --version && docker compose version\`
- Step 2: Install M3TAL via APT (include the exact 3-command APT block from context).
- Step 3: Run the configuration wizard: \`sudo m3tal config wizard\` — explain what each prompt means.
- Step 4: Start the routing stack (Traefik): \`m3tal up\` — explain this starts all compose files in /docker/.
- Step 5: Start the dashboard: \`m3tal dash up\` — explain it pulls the image and starts the container.
- Step 6: Open browser at \`http://YOUR_IP:8082\` (or \`http://dash.DOMAIN\` if Traefik is configured).
- Step 7: Log in — default credentials, how to change with \`sudo m3tal dashpass\`.
- Filesystem Contract section: document /etc/m3tal/.env, /var/lib/m3tal/state.db, /docker symlink.
- Port table: 80 (Traefik), 8080 (API), 8082 (Dashboard).
- Firewall note: \`sudo ufw allow 80\` if Traefik is exposed.
- Service management: show \`systemctl status m3tal-api\`, \`journalctl -u m3tal-api -f\`.

${context}
`
  },
  {
    path: "docs/COMMAND_REFERENCE.md",
    name: "Command Line Reference",
    prompt: `You are DocSmith, the M3TAL Ecosystem Documentation Architect.
Generate docs/COMMAND_REFERENCE.md — a complete CLI cheat-sheet.

STRICT RULES:
- Document every command with a real usage example. No placeholders.
  - \`m3tal\` (or \`m3tal ui\`) → Launches the M3TAL Linux Desktop Application.
  - \`m3tal headless [port]\` → Starts M3TAL in headless mode (ideal for home servers).
  - \`m3tal stats\` → Prints instant host system, GPU, thermal, and storage metrics to the terminal.
  - \`m3tal doctor\` → Pre-flight health check: Docker connectivity, .env validity, port availability.
  - \`m3tal config wizard\` → Interactive wizard to configure /etc/m3tal/.env.
  - \`m3tal config set KEY VALUE\` → Set a single env var.
  - \`m3tal config get KEY\` → Read a single env var.
  - \`m3tal config scan\` → List all env vars across all stacks.
  - \`m3tal config list\` → List current .env file contents.
  - \`m3tal dashpass [username] [password]\` → Update dashboard user password. Interactive if args omitted.
  - \`m3tal dash up\` → Pull latest dashboard compose config from GitHub, then start the dashboard container.
  - \`m3tal dash down\` → Stop the dashboard container.
  - \`m3tal dash restart\` → Restart the dashboard container.
  - \`m3tal dash logs\` → Stream dashboard container logs.
  - \`m3tal dash status\` → Show dashboard container status.
  - \`m3tal up\` → Run docker compose up across all *-compose.yml files in /docker/.
  - \`m3tal down\` → Run docker compose down across all stacks.
  - \`m3tal logs\` → Stream aggregated logs from all running stacks.
- Include a section on systemd service management: \`systemctl status m3tal-api\`, \`journalctl -u m3tal-api -f\`.
- Include a Docker section showing direct compose commands as fallback.

${context}
`
  },
  {
    path: "docs/ENVIRONMENT_VARIABLES.md",
    name: "Environment Variables Reference",
    prompt: `You are DocSmith, the M3TAL Ecosystem Documentation Architect.
Generate docs/ENVIRONMENT_VARIABLES.md — a complete environment variable reference.

STRICT RULES:
- Use the Environment Variables State JSON from the context as your source of truth.
- For EVERY variable in the JSON, document:
  - Name, description, default value, example value, which component uses it.
- Note that DASHBOARD_SECRET and API_TOKEN are auto-generated on first \`m3tal init\` — users should NOT set them manually unless rotating.
- Note that BASE_STORAGE_PATH controls where media data is stored and defaults to /mnt in production deployments (not ./data as in the template).
- Note that DOMAIN controls Traefik routing rules — setting it enables \`dash.DOMAIN\` and \`api.DOMAIN\` routes.
- Explain that all variables are read from \`/etc/m3tal/.env\` by both the CLI and all compose stacks via --env-file.
- Group variables logically: Core, Auth, Network, Storage, Traefik, VPN, System.
- Include a quick-reference table at the top.

${context}
`
  }
];

// ─────────────────────────────────────────────────────────────────────────────
// MODEL FALLBACK CHAIN
// ─────────────────────────────────────────────────────────────────────────────
const modelsToTry = [
  "gemini-2.5-flash",        // Primary
  "gemini-2.5-flash-lite",   // Fallback
  "gemini-1.5-flash",        // Legacy
  "gemini-pro"               // Universal
];

function postProcessOutput(content) {
  let cleaned = content;

  // Regex to match any variant of the echo "deb [signed-by=... m3tal repository line and replace it with the perfectly formed, full command
  const aptRepoRegex = /echo\s+"deb\s+\[signed-by=\/usr\/share\/keyrings\/m3tal-archive-keyring\.gpg\]\s+https:\/\/jakej985-rgb\.github\.io\/[^\n]*/gi;
  cleaned = cleaned.replace(aptRepoRegex, 'echo "deb [signed-by=/usr/share/keyrings/m3tal-archive-keyring.gpg] https://jakej985-rgb.github.io/m3tal-apt-key stable main" | sudo tee /etc/apt/sources.list.d/m3tal.list');

  // Ensure GPG key download command is perfectly formatted
  const gpgRegex = /curl\s+-fsSL\s+https:\/\/jakej985-rgb\.github\.io\/m3tal-apt-key\/public\.key\s*\|\s*sudo\s+gpg\s+--dearmor\s+-o\s+\/usr\/share\/keyrings\/m3tal-archive-keyring\.gpg/gi;
  cleaned = cleaned.replace(gpgRegex, 'curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/public.key | sudo gpg --dearmor -o /usr/share/keyrings/m3tal-archive-keyring.gpg');

  return cleaned;
}

async function generateWithFallback(target) {
  for (const modelName of modelsToTry) {
    try {
      console.log(`[${target.name}] Attempting to use model: ${modelName}`);
      const model = genAI.getGenerativeModel({ model: modelName });
      const result = await model.generateContent(target.prompt);
      const output = result.response.text();
      const processed = postProcessOutput(output);
      fs.writeFileSync(target.path, processed);
      console.log(`[${target.name}] ✅ ${target.path} updated using ${modelName}.`);
      return;
    } catch (error) {
      console.warn(`[${target.name}] Model ${modelName} failed: ${error.message}`);
    }
  }
  console.error(`[${target.name}] ❌ All Gemini models failed to generate content.`);
  if (target.path === "README.md" && fs.existsSync("README.generated.md")) {
    fs.copyFileSync("README.generated.md", target.path);
    console.log(`[${target.name}] ⚠️ Fallback to README.generated.md applied successfully.`);
  }
}

async function run() {
  for (const target of targets) {
    await generateWithFallback(target);
  }
  console.log("✅ All documentation targets updated.");
}

run();
