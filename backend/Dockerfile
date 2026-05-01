FROM golang:1.24-alpine AS build

ARG TARGET=api

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/. .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/${TARGET} ./cmd/${TARGET}

FROM alpine:3.20

ARG TARGET=api

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=build /out/${TARGET} /app/${TARGET}

USER app

EXPOSE 8080

ENTRYPOINT ["/bin/sh", "-c", "/app/${TARGET}"]
