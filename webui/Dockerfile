# M3TAL Dashboard - Pure Debian Hardened Base
FROM debian:bookworm-slim

# Prevent debris and speed up installs
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

# 1. System Hardening & Resilient Tooling (Apt-First Python)
# Using Apt for core Python libs to bypass Pip corruption on unstable networks
RUN for i in {1..5}; do apt-get update && apt-get upgrade -y && \
    apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    curl \
    procps \
    python3-flask \
    python3-requests \
    python3-yaml \
    python3-psutil \
    python3-bcrypt \
    python3-eventlet \
    python3-flask-socketio \
    python3-pytest \
    python3-flake8 && break || sleep 5; done \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# 2. Python Toolchain Hardening (Using stable base-image pip)
# Skipping upgrade to avoid hash-mismatch errors on unstable networks

# 3. Application Setup
WORKDIR /docker/dashboard
COPY requirements.txt .
# Resilient install for any missing minor dependencies
RUN python3 -m pip install --no-cache-dir --default-timeout=1000 --retries 10 -r requirements.txt || true

# Copy source code
COPY . .

# 4. Non-Root Execution Enforcement
RUN groupadd -g 1000 appuser && \
    useradd -m -u 1000 -g 1000 appuser && \
    chown -R appuser:appuser /docker/dashboard && \
    mkdir -p /docker/state/logs && \
    chown -R appuser:appuser /docker/state/logs

USER appuser

# OCI Image Labels
LABEL org.opencontainers.image.title="M3TAL Dashboard" \
      org.opencontainers.image.description="AI-Powered Docker Control Plane Dashboard" \
      org.opencontainers.image.vendor="M3TAL" \
      org.opencontainers.image.logo="https://raw.githubusercontent.com/jakej985-rgb/M3tal-Media-Server/main/docs/godash_logo.svg"

# Startup command
CMD ["python3", "server.py"]
