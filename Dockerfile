FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN go build -o /out/spotself ./cmd/spotself
RUN go build -o /out/spotselfctl ./cmd/spotselfctl

FROM alpine:3.20

WORKDIR /app
COPY --from=build /out/spotself /usr/local/bin/spotself
COPY --from=build /out/spotselfctl /usr/local/bin/spotselfctl
COPY web ./web
ENV SPOTSELF_ADDR=:8080
ENV SPOTSELF_DATA_DIR=/app/data
EXPOSE 8080
VOLUME ["/app/data"]
CMD ["spotself"]
