FROM docker.io/library/golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gocuts .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gocuts /gocuts
EXPOSE 8080
ENV GOCUTS_CONFIG=/gocuts.toml
ENTRYPOINT ["/gocuts"]
