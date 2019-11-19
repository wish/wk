FROM golang:1.13

ENV GO111MODULE=on
WORKDIR /go/src/github.com/wish/wk

# Cache dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . /go/src/github.com/wish/wk
RUN CGO_ENABLED=0 GOOS=linux go build -o /wk -a -installsuffix cgo ./cmd/wk

FROM quay.io/wish/jsonnet-builder:v0.13.0
WORKDIR /
COPY --from=0 /wk /bin/wk
