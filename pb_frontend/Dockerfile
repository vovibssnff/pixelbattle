# Build Vue app; runtime copies artifacts into shared volume for Caddy (see docker-compose).
FROM node:alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM alpine:3.19
COPY --from=build /app/dist /static
# caddy_html is a named volume: merge-only cp leaves old hashed JS/CSS. Wipe /srv then copy so deploys always match the image.
CMD ["sh", "-c", "rm -rf /srv/* && cp -a /static/. /srv/ && exec sleep infinity"]
