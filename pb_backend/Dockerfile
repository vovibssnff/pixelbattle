FROM golang:latest

WORKDIR app
COPY go.* ./
RUN go mod download

COPY . .
# COPY pixels pixels
# COPY server server
CMD ["go", "run", "./main/main.go"]

