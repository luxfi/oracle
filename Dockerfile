# syntax=docker/dockerfile:1.6
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/oracled ./cmd/oracled

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/oracled /usr/local/bin/oracled
ENV PORT=7800
EXPOSE 7800
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/oracled"]
