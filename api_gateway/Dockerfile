FROM golang:1.24

WORKDIR /api_gateway

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main ./cmd

EXPOSE 8082

CMD ["./main"]
