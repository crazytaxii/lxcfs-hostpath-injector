FROM golang:alpine AS builder

RUN mkdir /app
WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o lxcfs-sidecar-injector ./cmd/injector

FROM alpine:latest
ARG REPO_URL
ARG BRANCH
ARG COMMIT_REF
LABEL repo-url=$REPO_URL
LABEL branch=$BRANCH
LABEL commit-ref=$COMMIT_REF
RUN mkdir -p /etc/webhook/certs
COPY --from=builder /app/lxcfs-sidecar-injector /app/
WORKDIR /app
VOLUME [ "/etc/webhook/certs" ]
EXPOSE 443
ENTRYPOINT [ "./lxcfs-sidecar-injector" ]
