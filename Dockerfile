FROM golang:1.22 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go build -v -o /usr/src/app/goBotter


FROM alpine:3
RUN apk add --no-cache ca-certificates

WORKDIR /the/workdir/path
COPY --from=builder /usr/src/app/goBotter /opt/goBotter

CMD [ "/opt/goBotter" ]
