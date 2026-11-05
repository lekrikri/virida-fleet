FROM ubuntu:24.04
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends openssh-server python3 sudo \
 && rm -rf /var/lib/apt/lists/* && mkdir -p /run/sshd \
 && echo 'ubuntu ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/ubuntu
CMD ["/usr/sbin/sshd", "-D", "-e"]
