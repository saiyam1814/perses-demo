# Build
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app main.go

# Run (distroless scratch style)
FROM gcr.io/distroless/static:nonroot
WORKDIR /
USER nonroot:nonroot
COPY --from=build /out/app /app
EXPOSE 8080
ENTRYPOINT ["/app"]

