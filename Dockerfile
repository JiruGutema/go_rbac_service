# --- dev: hot reload with Air (used by docker compose) ---
FROM golang:1.25-alpine AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@latest
COPY go.* ./
RUN go mod download
CMD ["air"]

FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/rbac-server .

FROM alpine:3.22
COPY --from=build /bin/rbac-server /usr/local/bin/rbac-server
EXPOSE 8080
ENTRYPOINT ["rbac-server"]
