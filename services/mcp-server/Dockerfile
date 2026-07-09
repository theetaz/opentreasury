# Build
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mcp-server ./cmd/mcp-server

# Run
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mcp-server /mcp-server
EXPOSE 8090
USER nonroot
ENTRYPOINT ["/mcp-server"]
