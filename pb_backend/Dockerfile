FROM golang:latest

WORKDIR /usr/src/app
COPY go.* ./
RUN go mod download

COPY ava.png ava.png
COPY . .
# COPY pixels pixels
# COPY server server
RUN CGO_ENABLED=0 go build -o /pb_server ./server/main.go

CMD ["/pb_server"]
