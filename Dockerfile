FROM debian:11.7-slim

RUN set -x \
 && sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list \
 && mkdir -p /opt/nautes/bin /opt/nautes/config /opt/nautes/keypair /opt/nautes/cert /opt/nautes/log \
 && apt update && apt install -y --no-install-recommends ca-certificates netbase curl git openssh-client \
 && rm -rf /var/lib/apt/lists/ \
 && apt autoremove -y && apt autoclean -y \
 && groupadd --gid 65532 nautes \
 && useradd --gid nautes --no-create-home --home /opt/nautes --comment "nautes user" --uid 65532 nautes

COPY bin /opt/nautes/bin

USER 65532:65532
WORKDIR /opt/nautes/
