ARG serverBinary="build/mealbot"
ARG siteRoot="./site"
FROM alpine:latest

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
ADD build/ /opt/mealbot/

EXPOSE 80
WORKDIR /opt/mealbot/
ENTRYPOINT [ "/opt/mealbot/mealbot" ]