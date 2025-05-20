# Dockerfile

# --- Giai đoạn 1: Build ứng dụng Go ---
FROM golang:1.23.7-alpine AS builder

# Đặt thư mục làm việc
WORKDIR /app

# Sao chép file go.mod và go.sum để tải dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Sao chép toàn bộ mã nguồn của ứng dụng
COPY . .

# Build ứng dụng Go
# CGO_ENABLED=0 để tạo file thực thi tĩnh, không phụ thuộc vào thư viện C
# -o /app/main để đặt tên file thực thi là "main" và lưu vào thư mục /app
# ./main.go là đường dẫn tới file main của bạn
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main ./main.go

# --- Giai đoạn 2: Tạo image runtime cuối cùng ---
FROM alpine:latest

# Cài đặt các chứng chỉ CA cần thiết cho kết nối HTTPS, SSL/TLS (ví dụ: kết nối tới AWS)
RUN apk --no-cache add ca-certificates

# Đặt thư mục làm việc
WORKDIR /root/

# Sao chép file thực thi đã được build từ giai đoạn builder
COPY --from=builder /app/main .

# (Tùy chọn) Sao chép thư mục migrations nếu bạn muốn chạy migrations từ trong container
# COPY migrations ./migrations

# (Tùy chọn) Sao chép file .env.
# LƯU Ý: Trong môi trường production, cách tốt hơn là sử dụng biến môi trường của Docker/Kubernetes/dịch vụ hosting
# thay vì sao chép trực tiếp file .env vào image.
# Nếu bạn vẫn muốn sao chép, hãy đảm bảo file .env không chứa thông tin quá nhạy cảm khi build image công khai.
# COPY .env .

# Expose port mà ứng dụng của bạn lắng nghe (ví dụ: 8080 từ config)
# Giá trị này nên khớp với SERVER_PORT trong file .env hoặc biến môi trường
EXPOSE 8080

# Lệnh để chạy ứng dụng khi container khởi động
# File thực thi của chúng ta tên là "main" và nằm ở thư mục làm việc hiện tại (/root/)
CMD ["./main"]
