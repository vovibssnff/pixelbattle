FROM node:24-alpine3.20 AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM node:24-alpine3.20 AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM alpine:3.23 AS runtime
# Numeric UID/GID — no /etc/passwd entry needed; satisfies Trivy DS-0002 (no root).
# The named docker volume mounted at /srv inherits this ownership on first creation,
# so the unprivileged runtime can still seed the static assets via cp/find.
RUN apk upgrade --no-cache \
 && mkdir -p /static /srv \
 && chown -R 10001:10001 /static /srv
COPY --chown=10001:10001 --from=builder /app/dist /static
USER 10001:10001
ENTRYPOINT ["sh","-c","find /srv -mindepth 1 -delete; cp -a /static/. /srv/ && chmod -R a+rX /srv && exec sleep infinity"]

FROM runtime AS final
