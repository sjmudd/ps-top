# severalnines/sysbench (ashraf-s9s/sysbench-docker), rebuilt on a current
# Debian release instead of the floating debian:latest tag. The upstream
# image's client library (libmariadb3, pulled in transitively by `apt-get
# install sysbench`) is missing the caching_sha2_password.so client auth
# plugin, so it can't connect to caching_sha2_password-only servers (e.g.
# MySQL 9.x). That's Debian package drift from whatever snapshot the
# upstream image last built against, not a deliberate upstream choice -
# rebuilding on current Debian pulls in a libmariadb3 that already bundles
# the plugin. Verified: full sysbench prepare+run cycle against
# caching_sha2_password succeeds unmodified otherwise.
FROM debian:13
MAINTAINER Severalnines <ashraf@severalnines.com>

RUN apt-get update && \
	apt-get -y install curl

RUN curl -s https://packagecloud.io/install/repositories/akopytov/sysbench/script.deb.sh | bash

RUN apt-get -y install sysbench git make gcc unzip wget lua5.1 lua5.1-dev && \
	apt-get clean

RUN wget https://luarocks.org/releases/luarocks-2.4.3.tar.gz && tar zxpf luarocks-2.4.3.tar.gz && cd luarocks-2.4.3 && ./configure && make bootstrap
