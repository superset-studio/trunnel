FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY api/go.mod api/go.sum ./api/
RUN cd api && go mod download

COPY api/ ./api/

RUN cd api && CGO_ENABLED=0 go build -o /kapstan ./cmd/kapstan

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /kapstan /usr/local/bin/kapstan
COPY api/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["kapstan"]
CMD ["server"]
