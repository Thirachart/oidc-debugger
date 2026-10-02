# Stage 1: Build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go mod download
# คอมไพล์เป็น Static Binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o debugger main.go

# Stage 2: Final Image (ใช้ Alpine เพื่อให้มี shell ไว้ debug หรือใช้ scratch เพื่อความเล็กสุด)
FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/debugger .
EXPOSE 5000
CMD ["./debugger"]