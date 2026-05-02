FROM node:24-alpine3.20 AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM node:24-alpine3.20 AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM alpine:3.20 AS runtime
RUN apk upgrade --no-cache
COPY --from=builder /app/dist /static
ENTRYPOINT ["sh","-c","find /srv -mindepth 1 -delete; cp -a /static/. /srv/ && chmod -R a+rX /srv && exec sleep infinity"]

FROM runtime AS final
