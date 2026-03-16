# Stage 1: Build CSS
FROM node:lts-alpine AS css
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci
COPY static/css/input.css static/css/
COPY templates/ templates/
RUN npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --minify

# Stage 2: Build Go binary
FROM golang:alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=css /build/static/css/styles.css static/css/
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o app ./cmd/server

# Stage 3: Minimal runtime
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /build/app /app
COPY --from=builder /build/static /static
COPY --from=builder /build/templates /templates
EXPOSE 8080
ENTRYPOINT ["/app"]
