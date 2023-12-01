FROM public.ecr.aws/docker/library/golang:latest AS build 

WORKDIR /build

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o main_opensky .

FROM public.ecr.aws/docker/library/alpine:latest

WORKDIR /app  

COPY --from=build /build/main_opensky .

RUN chmod +x main_opensky

RUN pwd && find .

CMD ["./main_opensky"]
