FROM debian:11

RUN apt update && apt install -y wget iptables iproute2 curl
RUN update-alternatives --set iptables /usr/sbin/iptables-legacy

COPY build/firecracker /usr/bin/
COPY build/jailer /usr/bin/
COPY build/mesos-firecracker-executor /usr/bin/
COPY build/rebase-snap /usr/bin/
COPY build/seccomp-filter.json /usr/bin/
COPY resources/fcnet.conflist /etc/cni/conf.d/
COPY resources/tc-redirect-tap /opt/cni/bin/

RUN export ARCH=`dpkg --print-architecture` && \
    export MARCH=`uname -m` && \
    wget -O /cni.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.3/cni-plugins-linux-${ARCH}-v0.8.3.tgz && \
    mkdir -p /etc/cni/conf.d/ && \
    mkdir -p /opt/cni/bin && \
    tar -xvf /cni.tgz --directory /opt/cni/bin/ && \
    rm /cni.tgz    

WORKDIR /tmp


