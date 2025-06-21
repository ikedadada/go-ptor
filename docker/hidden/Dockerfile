# Build stage
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY docker/hidden/hidden.pem /src/hidden.pem
RUN CGO_ENABLED=0 GOOS=linux go build -o /hidden ./cmd/hidden

# Runtime stage
FROM gcr.io/distroless/static
COPY --from=build /hidden /hidden
COPY --from=build /src/hidden.pem /hidden.pem
EXPOSE 5000 8080
ENTRYPOINT ["/hidden"]
