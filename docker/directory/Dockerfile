# Build stage
FROM golang:1.24.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY docker/directory/hidden_services.json /src/hidden_services.json
COPY docker/directory/relays.json /src/relays.json
RUN CGO_ENABLED=0 GOOS=linux go build -o /directory ./cmd/directory

# Runtime stage
FROM gcr.io/distroless/static
COPY --from=build /directory /directory
COPY --from=build /src/hidden_services.json /hidden_services.json
COPY --from=build /src/relays.json /relays.json
EXPOSE 8081
ENTRYPOINT ["/directory"]
