FROM golang:1.21.1

WORKDIR /app

COPY . .

RUN go build -o main_opensky .

EXPOSE 80

CMD ["./main_opensky"]
