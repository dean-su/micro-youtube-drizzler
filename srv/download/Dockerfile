FROM golang:1.9.2-alpine3.6 AS build

# Install tools required to build the project
# We need to run `docker build --no-cache .` to update those dependencies
RUN apk add --no-cache git
RUN go get github.com/golang/dep/cmd/dep

# Gopkg.toml and Gopkg.lock lists project dependencies
# These layers are only re-built when Gopkg files are updated
COPY Gopkg.lock Gopkg.toml /go/src/download/
WORKDIR /go/src/download/
# Install library dependencies
RUN dep ensure -vendor-only

# Copy all project and build it
# This layer is rebuilt when ever a file has changed in the project directory
COPY . /go/src/download/
RUN go build -o /bin/download

# This results in a single layer image
FROM scratch
COPY --from=build /bin/download /bin/download
ENTRYPOINT ["/bin/download"]
CMD ["--help"]
