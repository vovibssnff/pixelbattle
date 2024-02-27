FROM golang:latest

WORKDIR /usr/src/app
COPY go.* ./
RUN go mod download

COPY . .
# COPY pixels pixels
# COPY server server
RUN CGO_ENABLED=0 go build -o /pb_server ./main/main.go

CMD ["/pb_server"]
