FROM golang:1.22-alpine as builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /app
COPY . .

RUN go build -o bayes_sampling .

FROM alpine:latest

COPY --from=builder /app/bayes_sampling /usr/local/bin/bayes_sampling

ENTRYPOINT ["/usr/local/bin/bayes_sampling"]
EXPOSE 8080
