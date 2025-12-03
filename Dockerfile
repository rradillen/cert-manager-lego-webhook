FROM golang:1.23.5 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GO11MODULE=on go build -ldflags="-s -w" -o webhook .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder --chown=nonroot:nonroot /app/webhook /webhook

USER nonroot

ENTRYPOINT ["/webhook"]

FROM alpine:3.19 AS debug
RUN apk add --no-cache bash curl bind-tools iputils
WORKDIR /app
COPY --from=builder /app/webhook /webhook
ENTRYPOINT ["/webhook"]
