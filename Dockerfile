# Start by building the application.
FROM golang:1.12 as build

WORKDIR /go/src/go.merklecounty.com/rget
COPY . .

ENV GO111MODULE=on
ENV GOFLAGS=-mod=vendor

RUN go test -v ./...
RUN go install -v ./...

# Now copy it into our base image.
FROM gcr.io/distroless/base
COPY --from=build /go/bin/rget /
