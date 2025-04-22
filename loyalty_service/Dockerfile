FROM golang:1.24

WORKDIR /loyalty_service

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main ./cmd

EXPOSE 8083

CMD ["./main"]
