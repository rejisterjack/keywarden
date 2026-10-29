FROM golang:1.27.3 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/keywarden ./cmd/keywarden

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/keywarden /keywarden
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/keywarden"]
CMD ["serve"]
