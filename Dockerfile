# ---- build stage ----
FROM golang:1.26 AS build
WORKDIR /src

# Cloudflare Containers รันบน linux/amd64 เท่านั้น
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /app .

# ---- runtime stage ----
# distroless = image เล็ก, ไม่มี shell, รันด้วย non-root
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app /app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
