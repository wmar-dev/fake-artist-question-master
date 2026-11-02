FROM golang:1.27-alpine AS base
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Run the tests without installing Go: docker build --target test .
FROM base AS test
RUN go test ./...

FROM base AS build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM scratch
COPY --from=build /server /server
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/server"]
