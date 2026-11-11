FROM golang:1.27.4 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=0.1.0
RUN ver="${VERSION#v}" && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X github.com/rejisterjack/keywarden/internal/version.Version=${ver}" -o /out/keywarden ./cmd/keywarden

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/keywarden /keywarden
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/keywarden"]
CMD ["serve"]
