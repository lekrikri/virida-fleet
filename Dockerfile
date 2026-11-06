FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
ARG PROGRAMME=simulateur
RUN CGO_ENABLED=0 go build -o /app ./cmd/${PROGRAMME}
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app /app
ENTRYPOINT ["/app"]




