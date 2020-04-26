FROM golang
WORKDIR /app
COPY . /app

RUN go build ./cmd/proxy

EXPOSE 80

CMD ["/app/proxy"]
