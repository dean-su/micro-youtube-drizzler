# build stage
FROM golang:1.9-alpine AS build-env
RUN mkdir -p /go/src \
  && mkdir -p /go/bin \
  && mkdir -p /go/pkg
RUN apk update && apk add git

WORKDIR /go/src/myapp
ADD . .
RUN go build -v -o download-srv .

# final stage
FROM alpine:3.2

WORKDIR /app
COPY --from=build-env /go/src/myapp/download-srv download-srv
ENTRYPOINT [ "/app/download-srv" ]
