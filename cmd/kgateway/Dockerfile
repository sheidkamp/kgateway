FROM cgr.dev/chainguard/static@sha256:b2e1c3d3627093e54f6805823e73edd17ab93d6c7202e672988080c863e0412b AS distroless_source

ARG GOARCH=amd64

COPY kgateway-linux-$GOARCH /usr/local/bin/kgateway

USER 10101

ENTRYPOINT ["/usr/local/bin/kgateway"]
