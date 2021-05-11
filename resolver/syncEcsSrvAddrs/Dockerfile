FROM alpine:latest
MAINTAINER Samuel <zb@letabc.com>

# ca-certificates fix "x509: failed to load system roots and no roots provided"
# tzdata fix "time: missing Location in call to Time.In"
RUN apk update \
  && apk add ca-certificates tzdata \
  && rm -rf /var/cache/apk/* \
  && adduser -D -g '' ecs-user

WORKDIR /opt/app/go/bin
USER ecs-user

COPY syncEcsSrvAddrs syncEcsSrvAddrs
COPY config.toml config.toml
ENTRYPOINT ["./syncEcsSrvAddrs"]
