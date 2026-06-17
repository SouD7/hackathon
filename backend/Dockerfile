FROM golang:1.24-bookworm AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /api /api
COPY migrations /migrations
EXPOSE 8080
USER nonroot:nonroot
CMD ["/api"]
