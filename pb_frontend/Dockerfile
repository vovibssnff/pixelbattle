# Build Vue app; runtime copies artifacts into shared volume for Caddy (see docker-compose).
FROM node:alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM alpine:3.19
COPY --from=build /app/dist /static
CMD ["sh", "-c", "cp -r /static/. /srv/ && sleep infinity"]
