FROM golang:1.22 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -v -o /usr/src/app/goBotter


FROM alpine:3
RUN apk add --no-cache ca-certificates

WORKDIR /opt
COPY --from=builder /usr/src/app/goBotter /opt/goBotter

CMD [ "./goBotter" ]
