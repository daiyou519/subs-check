FROM debian:bookworm

ARG TARGETPLATFORM
ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y ca-certificates tzdata curl && \
    ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata && \
    rm -rf /var/cache/apt/*  && \
    mkdir -p /opt

COPY docker/glibc/${TARGETPLATFORM}/bestsub /opt/bestsub
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]