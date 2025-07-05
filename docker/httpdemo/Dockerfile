# Build stage
FROM golang:1.24.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /httpdemo ./cmd/httpdemo

# Runtime stage
FROM gcr.io/distroless/static
COPY --from=build /httpdemo /httpdemo
EXPOSE 8080
ENTRYPOINT ["/httpdemo"]
