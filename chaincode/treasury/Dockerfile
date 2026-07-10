# Build
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/treasury-chaincode ./cmd/treasury-chaincode

# Run (chaincode-as-a-service)
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/treasury-chaincode /treasury-chaincode
USER nonroot
ENTRYPOINT ["/treasury-chaincode"]
