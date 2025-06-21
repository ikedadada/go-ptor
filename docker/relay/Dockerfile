# Build stage
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /relay ./cmd/relay

# Runtime stage
FROM gcr.io/distroless/static
COPY --from=build /relay /relay
EXPOSE 5000
ENTRYPOINT ["/relay"]
