FROM golang:1.21.6-alpine

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY cmd cmd
COPY internal internal
COPY pkg pkg
RUN CGO_ENABLED=0 go build -o /pb_server ./main.go

CMD ["/pb_server"]