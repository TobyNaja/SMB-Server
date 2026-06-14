FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    samba \
    samba-common \
    samba-common-bin \
    smbclient \
    winbind \
    curl \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
COPY smb.conf.template /smb.conf.template

RUN chmod +x /entrypoint.sh && \
    mkdir -p /mnt/shared && \
    chmod 777 /mnt/shared && \
    mkdir -p /var/log/samba && \
    chmod 755 /var/log/samba

EXPOSE 137/udp 138/udp 139/tcp 445/tcp

ENTRYPOINT ["/entrypoint.sh"]

