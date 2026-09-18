# --- frontend build ---
FROM node:22-alpine AS webbuild
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# --- server build ---
FROM golang:1.26-alpine AS gobuild
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd
COPY internal/ ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /runeinsights ./cmd/server

# --- runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D app
USER app
WORKDIR /home/app
COPY --from=gobuild /runeinsights /usr/local/bin/runeinsights
COPY --from=webbuild /app/web/dist ./web/dist
VOLUME /home/app/data
ENV PORT=8080 DB_PATH=/home/app/data/rs.db STATIC_DIR=/home/app/web/dist
EXPOSE 8080
ENTRYPOINT ["runeinsights"]
