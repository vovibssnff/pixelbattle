FROM node:24-alpine3.20 AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM node:24-alpine3.20 AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# Ship static files baked into Caddy — no shared volume needed
FROM caddy:2.8.4-alpine AS final
COPY --from=builder /app/dist /srv
