FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 go build -o simulateur ./cmd/simulateur
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /src/simulateur /simulateur
ENTRYPOINT ["/simulateur"]




