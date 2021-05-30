FROM golang:alpine

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o main .

WORKDIR /dist

RUN cp /build/main .
RUN cp -R /build/resources .
RUN cp /build/password.pwd .

CMD ["/dist/main"]


