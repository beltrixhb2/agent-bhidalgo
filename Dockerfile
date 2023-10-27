FROM golang:latest AS build

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main_opensky .

FROM alpine:latest

COPY --from=build /app/main_opensky .

RUN chmod +x main_opensky

EXPOSE 80

CMD ["./main_opensky"]
