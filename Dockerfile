FROM golang:1.19-alpine as build
WORKDIR /build/terrastate
COPY . .
RUN go mod vendor && \
        CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o /terrastate cmd/terrastate/main.go && \
        apk add upx binutils && \
        strip /terrastate && \
        upx /terrastate && \
        ls -alh /terrastate

FROM scratch
LABEL org.opencontainers.image.source https://github.com/the-maldridge/terrastate
ENTRYPOINT ["/terrastate"]
ENV TS_AUTH=netauth \
        TS_BIND=0.0.0.0:8080 \
        TS_STORE=bitcask \
        TS_BITCASK_PATH=/data
COPY --from=build /terrastate /terrastate
