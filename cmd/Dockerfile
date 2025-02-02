FROM golang:1.23.5-alpine

WORKDIR /usr/share/openweather-dashboard

COPY go.mod .

COPY go.sum .

RUN go mod download

COPY ../. .

RUN go build -o main .

CMD ["./main"]