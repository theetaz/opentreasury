# Build
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/generic-file-connector ./cmd/generic-file-connector

# Run
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/generic-file-connector /generic-file-connector
COPY profiles/itmis-payments.yaml /profiles/itmis-payments.yaml
USER nonroot
ENTRYPOINT ["/generic-file-connector"]
