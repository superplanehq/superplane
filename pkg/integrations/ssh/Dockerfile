FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-server ca-certificates bash coreutils procps iproute2 util-linux \
 && rm -rf /var/lib/apt/lists/*

# SSH runtime dirs
RUN mkdir -p /var/run/sshd

# Hardening-ish defaults (still simple for local testing)
RUN sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config \
 && sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config \
 && sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config \
 && echo "AllowTcpForwarding no" >> /etc/ssh/sshd_config \
 && echo "X11Forwarding no" >> /etc/ssh/sshd_config \
 && echo "ClientAliveInterval 30" >> /etc/ssh/sshd_config \
 && echo "ClientAliveCountMax 3" >> /etc/ssh/sshd_config

# Add a test user
ARG USERNAME=testuser
RUN useradd -m -s /bin/bash ${USERNAME}

# Add entrypoint
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 22
ENTRYPOINT ["/entrypoint.sh"]
