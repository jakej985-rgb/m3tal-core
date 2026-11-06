#!/usr/bin/env python3
"""
M3TAL External Plugin Catalog Generator
Generates catalog.json for the m3tal-plugins repository.
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
        "capabilities": ["collector", "actions", "ui_view", "ui_widget", "settings"],
        "version": "0.8.1",
        "status": "available",
        "apt_command": "sudo apt install -y m3tal-plugin-docker-monitor",
    },
    {
        "name": "compose",
        "title": "Compose Editor",
        "process_name": "m3tal-compose",
        "package_name": "m3tal-plugin-compose",
        "category": "Container Management",
        "description": "Docker Compose visual editor, syntax validator, and multi-container stack lifecycle manager.",
        "capabilities": ["collector", "actions", "ui_view", "settings"],
        "version": "0.6.0",
        "status": "available",
        "apt_command": "sudo apt install -y m3tal-plugin-compose",
    },
    {
        "name": "mesh",
        "title": "Mesh Fleet",
        "process_name": "m3tal-mesh",
        "package_name": "m3tal-plugin-mesh",
        "category": "Networking & Fleet",
        "description": "Local network discovery, multi-node clustering, and telemetry federation across M3TAL workstations, servers, and NAS nodes.",
        "capabilities": ["collector", "actions", "ui_view", "settings"],
        "version": "0.5.0",
        "status": "available",
        "apt_command": "sudo apt install -y m3tal-plugin-mesh",
        "modes": [
            {
                "id": "cluster",
                "label": "Cluster",
                "sublabel": "Full Peer",
                "description": "Full peer (Send + Receive). Broadcasts local machine metrics AND renders the multi-node fleet dashboard. Recommended for primary workstations.",
                "package_name": "m3tal-plugin-mesh-cluster",
                "apt_command": "sudo apt install -y m3tal-plugin-mesh-cluster",
                "process_name": "m3tal-mesh",
            },
            {
                "id": "mesh",
                "label": "Mesh",
                "sublabel": "Viewer",
                "description": "Reader and dashboard UI. Scans LAN to view and monitor other connected M3TAL nodes from your desktop.",
                "package_name": "m3tal-plugin-mesh",
                "apt_command": "sudo apt install -y m3tal-plugin-mesh",
                "process_name": "m3tal-mesh",
            },
            {
                "id": "node",
                "label": "Node",
                "sublabel": "Sender",
                "description": "Headless sender daemon. Broadcasts CPU, GPU, and disk metrics without running a GUI. Ideal for servers and NAS.",
                "package_name": "m3tal-plugin-mesh-node",
                "apt_command": "sudo apt install -y m3tal-plugin-mesh-node",
                "process_name": "m3tal-mesh-node",
            },
        ],
    },
    {
        "name": "mesh-node",
        "title": "Mesh Node",
        "process_name": "m3tal-mesh-node",
        "package_name": "m3tal-plugin-mesh-node",
        "category": "Networking & Fleet",
        "description": "Headless telemetry broadcaster for servers, NAS devices, and secondary machines.",
        "capabilities": ["collector", "actions", "ui_view", "settings"],
        "version": "0.5.0",
        "status": "available",
        "apt_command": "sudo apt install -y m3tal-plugin-mesh-node",
    },
    {
        "name": "samba-nfs",
        "title": "File Shares",
        "process_name": "m3tal-samba-nfs",
        "package_name": "m3tal-plugin-samba-nfs",
        "category": "File Sharing",
        "description": "Graphical Samba (SMB/CIFS) and NFS share manager: share provisioning, user credentials, guest permissions, active client monitoring, and network mounts.",
        "capabilities": ["collector", "actions", "ui_view", "settings"],
        "version": "0.1.0",
        "status": "developing",
    },
    {
        "name": "ai-engine",
        "title": "Local AI Assistant",
        "process_name": "m3tal-ai",
        "package_name": "m3tal-plugin-ai",
        "category": "Local AI & LLM",
        "description": "In-progress M3TAL assistant/RAG prototype. Not yet a complete installable Core plugin.",
        "capabilities": ["collector", "actions", "ui_view"],
        "status": "developing",
    },
    {
        "name": "stack-orchestrator",
        "title": "Compose Template DB",
        "process_name": "m3tal-stacks",
        "package_name": "m3tal-plugin-stacks",
        "category": "Compose Stacks",
        "description": "Planned curated database of verified Docker Compose templates, argument presets, and community stacks.",
        "capabilities": ["actions", "ui_view"],
        "status": "planned",
    },
    {
        "name": "smart-disk",
        "title": "Smart Disk Telemetry",
        "process_name": "m3tal-smart",
        "package_name": "m3tal-plugin-smart",
        "category": "Hardware Diagnostics",
        "description": "Planned drive health telemetry, NVMe wear life, S.M.A.R.T. self-tests, and temperature history.",
        "capabilities": ["collector", "ui_widget"],
        "status": "planned",
    },
    {
        "name": "traefik-gateway",
        "title": "Traefik Gateway",
        "process_name": "m3tal-traefik",
        "package_name": "m3tal-plugin-traefik",
        "category": "Networking & Proxy",
        "description": "Planned Traefik reverse proxy, automated TLS certificates, and dynamic routing manager.",
        "capabilities": ["actions", "ui_view", "settings"],
        "status": "planned",
    },
    {
        "name": "gluetun-vpn",
        "title": "VPN Tunnel",
        "process_name": "m3tal-vpn",
        "package_name": "m3tal-plugin-vpn",
        "category": "Network Privacy",
        "description": "Planned WireGuard/OpenVPN tunnel connectivity with container egress routing and leak protection.",
        "capabilities": ["collector", "actions", "ui_view", "settings"],
        "status": "planned",
    },
    {
        "name": "system-bench",
        "title": "System Benchmarks",
        "process_name": "m3tal-bench",
        "package_name": "m3tal-plugin-bench",
        "category": "Diagnostics",
        "description": "Planned one-click benchmarking for CPU multi-threading, GPU compute, and disk I/O throughput.",
        "capabilities": ["actions", "ui_view"],
        "status": "planned",
    },
]


def generate_catalog(output_path=None, search_dirs=None):
    entries = [dict(item) for item in DEFAULT_CATALOG]
    if search_dirs is None:
        search_dirs = [
            ".",
            "plugin-repo",
            "/home/m3tal/apps/m3tal-plugins",
            "/home/m3tal/apps/m3tal-plugins/dist/packages",
        ]

    for entry in entries:
        package_name = entry.get("package_name", "")
        if not package_name:
            continue
        for sdir in search_dirs:
            if not os.path.isdir(sdir):
                continue
            debs = glob.glob(os.path.join(sdir, f"{package_name}_*.deb"))
            exact = os.path.join(sdir, f"{package_name}.deb")
            if not debs and os.path.isfile(exact):
                debs = [exact]
            if debs:
                entry["has_deb"] = True
                entry["deb_filename"] = os.path.basename(debs[0])
                entry["download_url"] = f"/api/v2/plugins/{entry['name']}/download"
                entry["status"] = "available"
                break

    content = json.dumps(entries, indent=2) + "\n"
    if output_path:
        out_dir = os.path.dirname(output_path)
        if out_dir:
            os.makedirs(out_dir, exist_ok=True)
        with open(output_path, "w", encoding="utf-8") as f:
            f.write(content)
        print(f"Generated plugin catalog with {len(entries)} items at {output_path}")
    else:
        sys.stdout.write(content)


def main():
    parser = argparse.ArgumentParser(description="Generate M3TAL external plugin catalog JSON")
    parser.add_argument("--output", "-o", help="Target output file path (e.g. plugin-repo/catalog.json)")
    args = parser.parse_args()
    generate_catalog(output_path=args.output)


if __name__ == "__main__":
    main()
