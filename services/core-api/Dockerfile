# Build
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/core-api ./cmd/core-api

# Run
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/core-api /core-api
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/core-api"]
