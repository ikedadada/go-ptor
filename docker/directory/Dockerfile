# Build stage
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY docker/directory/directory.json /src/directory.json
RUN CGO_ENABLED=0 GOOS=linux go build -o /directory ./cmd/directory

# Runtime stage
FROM gcr.io/distroless/static
COPY --from=build /directory /directory
COPY --from=build /src/directory.json /directory.json
EXPOSE 8081
ENTRYPOINT ["/directory"]
