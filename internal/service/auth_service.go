package service

import (
	"context"
	"errors"
	"fmt"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"time"

	"github.com/golang-jwt/jwt/v5" // Cần go get github.com/golang-jwt/jwt/v5
	"golang.org/x/crypto/bcrypt"   // Cần go get golang.org/x/crypto/bcrypt
)

var ErrInvalidCredentials = errors.New("tên đăng nhập hoặc mật khẩu không đúng")
var ErrUserAlreadyExists = errors.New("tên người dùng đã tồn tại")
var ErrTokenInvalid = errors.New("token không hợp lệ hoặc đã hết hạn")

type AuthService struct {
	userRepo           repository.UserRepository
	jwtSecret          string
	jwtExpirationHours time.Duration
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string, jwtExpHours time.Duration) *AuthService {
	return &AuthService{
		userRepo:           userRepo,
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpHours,
	}
}

func (s *AuthService) Register(ctx context.Context, dto domain.RegisterUserDTO) (*domain.User, error) {
	// Kiểm tra username đã tồn tại chưa
	existingUser, err := s.userRepo.FindByUsername(ctx, dto.Username)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("lỗi khi kiểm tra người dùng: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Hash mật khẩu
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("lỗi hash mật khẩu: %w", err)
	}

	userRole := "operator" // Hoặc lấy từ dto nếu cần

	user := &domain.User{
		Username: dto.Username,
		Password: string(hashedPassword), // Lưu password đã hash
		Role:     userRole,
	}

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi tạo người dùng: %w", err)
	}
	createdUser.Password = "" // Không trả về password hash
	return createdUser, nil
}

func (s *AuthService) Login(ctx context.Context, dto domain.LoginUserDTO) (*domain.AuthResponseDTO, error) {
	user, err := s.userRepo.FindByUsername(ctx, dto.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("lỗi khi tìm người dùng: %w", err)
	}

	// So sánh mật khẩu đã hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(dto.Password))
	if err != nil {
		// Lỗi có thể là do mật khẩu không khớp hoặc lỗi bcrypt khác
		return nil, ErrInvalidCredentials
	}

	// Tạo JWT token
	expirationTime := time.Now().Add(s.jwtExpirationHours)
	//claims := &jwt.RegisteredClaims{
	//	Subject:   fmt.Sprintf("%d", user.ID), // Hoặc user.Username
	//	ExpiresAt: jwt.NewNumericDate(expirationTime),
	//	IssuedAt:  jwt.NewNumericDate(time.Now()),
	//	// Thêm các claims tùy chỉnh nếu cần, ví dụ: role
	//	// "role": user.Role,
	//}
	// Thêm role vào custom claims
	customClaims := jwt.MapClaims{
		"sub":      fmt.Sprintf("%d", user.ID),
		"exp":      expirationTime.Unix(),
		"iat":      time.Now().Unix(),
		"role":     user.Role,
		"username": user.Username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("lỗi tạo token: %w", err)
	}

	return &domain.AuthResponseDTO{
		Token:    tokenString,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

// ValidateToken dùng cho middleware
func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("phương thức ký không mong muốn: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		// Phân tích lỗi cụ thể từ jwt.ParseWithClaims
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, nil, fmt.Errorf("%w: token có định dạng sai", ErrTokenInvalid)
		} else if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, nil, fmt.Errorf("%w: token đã hết hạn", ErrTokenInvalid)
		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, nil, fmt.Errorf("%w: token chưa hợp lệ", ErrTokenInvalid)
		}
		return nil, nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	if !token.Valid {
		return nil, nil, ErrTokenInvalid
	}
	return token, claims, nil
}
