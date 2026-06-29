FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY services ./services
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /xcontext ./services/api/cmd/xcontext

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /xcontext /xcontext
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/xcontext"]
