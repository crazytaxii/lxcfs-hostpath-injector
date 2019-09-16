FROM alpine:latest

RUN mkdir -p /etc/webhook/certs
COPY _output/lxcfs-sidecar-injector /app/
WORKDIR /app
VOLUME [ "/etc/webhook/certs" ]
EXPOSE 443
ENTRYPOINT [ "./lxcfs-sidecar-injector" ]
