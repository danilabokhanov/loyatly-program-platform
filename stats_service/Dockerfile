FROM golang:1.24

WORKDIR /stats_service

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main ./cmd

EXPOSE 8085

CMD ["./main"]


