# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang
WORKDIR /go/src/app
COPY . .
# Copy the local package files to the container's workspace.
#ADD . /go/src/github.com/ericbreyer/riceServery

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
# RUN go get github.com/ericbreyer/riceServery
RUN go install github.com/ericbreyer/riceServery
RUN GOOS=linux go build -ldflags="-s -w" .

# Run the outyet command by default when the container starts.
ENTRYPOINT /go/bin/riceServery

# Document that the service listens on port 8080.
EXPOSE 80
EXPOSE 8080
