# Stage 1: build SvelteKit static SPA
FROM node:22-alpine AS node-builder
ENV CI=true
WORKDIR /src
RUN npm install -g pnpm
COPY frontend/ ./
RUN pnpm install --frozen-lockfile && pnpm build

# Stage 2: compile Go binary (tests must pass first)
FROM golang:1.25-alpine AS go-builder
WORKDIR /src
# Cache deps before copying source
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# All tests must pass — build is blocked if any test fails
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# Stage 3: minimal runtime image
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/server ./server
COPY --from=node-builder /src/build ./frontend/build
EXPOSE 8080
ENTRYPOINT ["/app/server"]
