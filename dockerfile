ARG BUILD_IMAGE

FROM golang:latest as build

COPY . .

RUN unset GOPATH && \
      make build

RUN adduser --uid 10001 --disabled-password liteswitch-user

FROM scratch

USER liteswitch-user

COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /go/liteswitch /

ENTRYPOINT [ "./liteswitch" ]