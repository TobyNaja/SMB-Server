FROM debian:bookworm
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    samba \
    samba-common \
    samba-common-bin \
    smbclient \
    winbind \
    libnss-winbind \
    libpam-winbind \
    krb5-user \
    ldap-utils \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Dedicated non-login owner for all share directories.
# Fixed uid/gid so on-disk ownership stays stable across image rebuilds
# (the /var/lib/samba + share data volumes persist).
RUN groupadd -g 1500 smbshare && \
    useradd -u 1500 -g 1500 -M -s /usr/sbin/nologin smbshare

COPY entrypoint.sh /entrypoint.sh
COPY smb.conf.template /smb.conf.template
COPY krb5.conf /etc/krb5.conf

RUN chmod +x /entrypoint.sh && \
    mkdir -p /mnt/shared && \
    chmod 777 /mnt/shared && \
    mkdir -p /var/log/samba && \
    chmod 755 /var/log/samba

EXPOSE 137/udp 138/udp 139/tcp 445/tcp

ENTRYPOINT ["/entrypoint.sh"]
