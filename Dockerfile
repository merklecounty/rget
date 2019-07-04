# Start by building the application.
FROM golang:1.12 as build

WORKDIR /go/src/github.com/merklecounty/rget
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

# Now copy it into our base image.
FROM gcr.io/distroless/base
COPY --from=build /go/bin/rget /
