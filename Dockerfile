FROM debian:12
LABEL maintainer="Andreas Peters <support@aventer.biz>"
LABEL org.opencontainers.image.title="mesos-firecracker-executor"
LABEL org.opencontainers.image.description="ClusterD/Apache Mesos firecracker executor"
LABEL org.opencontainers.image.vendor="AVENTER UG (haftungsbeschränkt)"
LABEL org.opencontainers.image.source="https://github.com/AVENTER-UG/"

RUN apt update && apt install -y wget iptables iproute2 curl
RUN update-alternatives --set iptables /usr/sbin/iptables-legacy

COPY resources/firecracker/firecracker /usr/bin/
COPY resources/firecracker/jailer /usr/bin/
COPY resources/firecracker/mesos-firecracker-executor /usr/bin/
COPY resources/firecracker/rebase-snap /usr/bin/
COPY resources/firecracker/seccomp-filter.json /usr/bin/
COPY resources/fcnet.conflist /etc/cni/conf.d/
COPY resources/tc-redirect-tap /opt/cni/bin/

RUN export ARCH=`dpkg --print-architecture` && \
    export MARCH=`uname -m` && \
    wget -O /cni.tgz https://github.com/containernetworking/plugins/releases/download/v1.8.0/cni-plugins-linux-${ARCH}-v1.8.0.tgz && \
    mkdir -p /etc/cni/conf.d/ && \
    mkdir -p /opt/cni/bin && \
    tar -xvf /cni.tgz --directory /opt/cni/bin/ && \
    rm /cni.tgz    

WORKDIR /tmp


