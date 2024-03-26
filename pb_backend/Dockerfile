FROM golang:1.22.0-bullseye AS build-env
WORKDIR /app
COPY . .
COPY go.* ./
RUN go mod download

RUN go install github.com/go-delve/delve/cmd/dlv@latest

RUN go build -gcflags="all=-N -l" -o main ./main

FROM debian:buster
EXPOSE 8000 40000

WORKDIR /
COPY --from=build-env /go/bin/dlv /
COPY --from=build-env /app/main /

CMD ["/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/main"]


#WORKDIR /app
#COPY go.* ./
#RUN go mod download
#
#COPY . .
## COPY pixels pixels
## COPY server server
#CMD ["go", "run", "./main/main.go"]

