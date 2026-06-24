# ---- build stage ----
FROM golang:1.26 AS build
WORKDIR /src

# ก๊อป go.mod/go.sum ก่อน → cache layer ของ dependency (เร็วเวลา build ซ้ำ)
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# build แบบ static (CGO ปิด) สำหรับ linux/amd64 — entrypoint อยู่ที่ ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /app ./cmd/api/...

# ---- runtime stage ----
# distroless = image เล็ก (~8MB), ไม่มี shell, รันด้วย non-root
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app /app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
