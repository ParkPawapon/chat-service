FROM golang:1.25.9-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/chat-service ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/chat-service /app/chat-service

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/chat-service"]
