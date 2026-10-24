#!/usr/bin/env python3
"""
M3TAL External Plugin Catalog Generator
Decoupled standalone script to generate catalog.json without requiring the M3TAL CLI.
"""
import argparse
import glob
import json
import os
import sys

DEFAULT_CATALOG = [
    {
        "name": "docker-monitor",
        "title": "Containers",
        "process_name": "m3tal-docker",
        "package_name": "m3tal-plugin-docker-monitor",
        "category": "Container Management",
        "description": "Docker container fleet monitor, log inspector, container lifecycle controls, and image pruning.",
        "capabilities": ["collector", "actions", "ui_view", "websocket_stream"],
        "version": "0.1.0",
        "status": "available",
        "apt_command": "sudo apt install -y m3tal-plugin-docker-monitor"
    },
    {
        "name": "stack-orchestrator",
        "title": "Compose Stacks",
        "process_name": "m3tal-stacks",
        "package_name": "m3tal-plugin-stacks",
        "category": "Compose Stacks",
        "description": "Multi-service compose stack orchestration, live health checks, and lifecycle deployment.",
        "capabilities": ["orchestrator", "actions", "ui_view"],
        "version": "0.1.0",
        "status": "planned",
        "apt_command": "sudo apt install -y m3tal-plugin-stacks"
    },
    {
        "name": "ai-engine",
        "title": "Local AI & LLM",
        "process_name": "m3tal-ai",
        "package_name": "m3tal-plugin-ai",
        "category": "Local AI & LLM",
        "description": "Local LLM inference manager (Ollama, vLLM), job queue, and model catalog.",
        "capabilities": ["inference_runner", "job_queue", "ui_view"],
        "version": "0.1.0",
        "status": "planned",
        "apt_command": "sudo apt install -y m3tal-plugin-ai"
    },
    {
        "name": "smart-disk",
        "title": "Smart Disk Telemetry",
        "process_name": "m3tal-smart",
        "package_name": "m3tal-plugin-smart",
        "category": "Hardware Diagnostics",
        "description": "Drive health telemetry, NVMe wear life, S.M.A.R.T. self-tests, and temperature history.",
        "capabilities": ["collector", "ui_widget", "alerts"],
        "version": "0.1.0",
        "status": "planned",
        "apt_command": "sudo apt install -y m3tal-plugin-smart"
    },
    {
        "name": "traefik-gateway",
        "title": "Traefik Gateway",
        "process_name": "m3tal-traefik",
        "package_name": "m3tal-plugin-traefik",
        "category": "Networking & Proxy",
        "description": "Traefik v3 reverse proxy, automated Let's Encrypt SSL/TLS certificates, and dynamic routing.",
        "capabilities": ["proxy_generator", "route_provider"],
        "version": "0.1.0",
        "status": "planned",
        "apt_command": "sudo apt install -y m3tal-plugin-traefik"
    },
    {
        "name": "gluetun-vpn",
        "title": "VPN Tunnel",
        "process_name": "m3tal-vpn",
        "package_name": "m3tal-plugin-vpn",
        "category": "Network Privacy",
        "description": "WireGuard/OpenVPN tunnel connectivity with container egress routing and leak protection.",
        "capabilities": ["network_tunnel", "status_checker"],
        "version": "0.1.0",
        "status": "planned",
        "apt_command": "sudo apt install -y m3tal-plugin-vpn"
    },
    {
        "name": "samba-manager",
        "title": "Samba Manager",
        "process_name": "m3tal-samba",
        "package_name": "m3tal-plugin-samba",
        "category": "File Sharing",
        "description": "Graphical Samba/NFS share manager, guest permissions, user credentials, and active mounts.",
        "capabilities": ["actions", "ui_view", "system_admin"],
        "version": "0.1.0",
        "status": "planned",
        "apt_command": "sudo apt install -y m3tal-plugin-samba"
    },
    {
        "name": "system-bench",
        "title": "System Benchmarks",
        "process_name": "m3tal-bench",
        "package_name": "m3tal-plugin-bench",
        "category": "Diagnostics",
        "description": "One-click benchmarking for CPU multi-threading, GPU compute, and disk I/O throughput.",
        "capabilities": ["benchmark_runner", "ui_view"],
        "version": "0.1.0",
        "status": "planned",
        "apt_command": "sudo apt install -y m3tal-plugin-bench"
    }
]

def generate_catalog(output_path=None, search_dirs=None):
    entries = [dict(item) for item in DEFAULT_CATALOG]
    
    if search_dirs is None:
        search_dirs = [
            ".",
            "plugin-repo",
            "/home/m3tal/apps/m3tal-plugins",
            "/home/m3tal/apps/m3tal-plugins/dist/packages"
        ]
    
    for entry in entries:
        name = entry["name"]
        for sdir in search_dirs:
            if not os.path.isdir(sdir):
                continue
            debs = glob.glob(os.path.join(sdir, f"m3tal-plugin-{name}_*.deb"))
            if not debs:
                exact = os.path.join(sdir, f"m3tal-plugin-{name}.deb")
                if os.path.isfile(exact):
                    debs = [exact]
            if debs:
                deb_file = os.path.basename(debs[0])
                entry["has_deb"] = True
                entry["deb_filename"] = deb_file
                entry["download_url"] = f"/api/v2/plugins/{name}/download"
                entry["status"] = "available"
                break
    
    content = json.dumps(entries, indent=2) + "\n"
    
    if output_path:
        out_dir = os.path.dirname(output_path)
        if out_dir:
            os.makedirs(out_dir, exist_ok=True)
        with open(output_path, "w", encoding="utf-8") as f:
            f.write(content)
        print(f"✅ Generated plugin catalog with {len(entries)} items at {output_path}")
    else:
        sys.stdout.write(content)

def main():
    parser = argparse.ArgumentParser(description="Generate M3TAL external plugin catalog JSON")
    parser.add_argument("--output", "-o", help="Target output file path (e.g. plugin-repo/catalog.json)")
    args = parser.parse_args()
    generate_catalog(output_path=args.output)

if __name__ == "__main__":
    main()
