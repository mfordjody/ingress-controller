FROM golang:1.23 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
COPY . .

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o ingress-controller cmd/main.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /workspace/ingress-controller .

ENTRYPOINT ["./ingress-controller"]
