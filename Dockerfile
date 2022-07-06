FROM golang:1.17 AS builder
WORKDIR /go/src/github.com/zhiweiyin318/addon-framework
COPY . .
ENV GO_PACKAGE github.com/zhiweiyin318/addon-framework

RUN make build --warn-undefined-variables

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
COPY --from=builder /go/src/github.com/zhiweiyin318/addon-framework/helloworld /
COPY --from=builder /go/src/github.com/zhiweiyin318/addon-framework/helloworld_helm /

RUN microdnf update && microdnf clean all
