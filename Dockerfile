FROM golang:1.22 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -v -o /usr/src/app/goBotter


FROM ubuntu:24.04
RUN apt update && apt install ca-certificates && apt upgrade -y

WORKDIR /opt
COPY --from=builder /usr/src/app/goBotter /opt/goBotter

CMD [ "./goBotter" ]
