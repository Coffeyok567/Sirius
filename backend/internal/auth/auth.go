package auth

import (
    "crypto/rand"
    "encoding/base64"
    "errors"
	"fmt"
    "os"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/argon2"
    "gorm.io/gorm"

    "messenger/internal/models"
)

type AuthService struct {
    db       *gorm.DB
    jwtSecret []byte
}

type Claims struct {
    UserID string `json:"userId"`
    jwt.RegisteredClaims
}

func NewAuthService(db *gorm.DB) *AuthService {
    secret := os.Getenv("MESSENGER_JWT_SECRET")
    if secret == "" {
        secret = "your-secret-key-change-in-production"
    }
    return &AuthService{
        db:        db,
        jwtSecret: []byte(secret),
    }
}

func (s *AuthService) Register(username, email, password, publicKey string) (*models.User, error) {
    // Check if user exists
    var existingUser models.User
	if err := s.db.Where("email = ?", email).First(&existingUser).Error; err == nil {
        return nil, errors.New("user already exists")
    }
    
    // Hash password with Argon2
    salt := make([]byte, 16)
    rand.Read(salt)
    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    
    // Store salt + hash
    passwordHash := base64.StdEncoding.EncodeToString(salt) + ":" + base64.StdEncoding.EncodeToString(hash)
    
	var disc string
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		d, err := allocateDiscriminator(tx, username)
		if err != nil {
			return err
		}
		disc = d
		return nil
	}); err != nil {
		return nil, err
	}

	user := &models.User{
        ID:                  generateID(),
        Username:            username,
		Discriminator:       disc,
        Email:               email,
        PasswordHash:        passwordHash,
        PublicKey:           publicKey,
        PrivateKeyEncrypted: "",
        CreatedAt:           time.Now(),
        UpdatedAt:           time.Now(),
    }
    
    if err := s.db.Create(user).Error; err != nil {
        return nil, err
    }
    
    return user, nil
}

func allocateDiscriminator(tx *gorm.DB, username string) (string, error) {
	// Pick the lowest available 0001..9999 for this username.
	var used []string
	if err := tx.Model(&models.User{}).Where("username = ?", username).Pluck("discriminator", &used).Error; err != nil {
		return "", err
	}
	seen := map[string]bool{}
	for _, u := range used {
		if len(u) == 4 {
			seen[u] = true
		}
	}
	for i := 1; i <= 9999; i++ {
		d := fmt.Sprintf("%04d", i)
		if !seen[d] {
			return d, nil
		}
	}
	return "", errors.New("no discriminator slots available for this username")
}

// BackfillDiscriminators assigns discriminators to existing users missing them.
func (s *AuthService) BackfillDiscriminators() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var users []models.User
		if err := tx.Where("discriminator = '' OR discriminator IS NULL").Order("created_at ASC").Find(&users).Error; err != nil {
			return err
		}
		for _, u := range users {
			d, err := allocateDiscriminator(tx, u.Username)
			if err != nil {
				return err
			}
			if err := tx.Model(&models.User{}).Where("id = ?", u.ID).Update("discriminator", d).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *AuthService) Login(email, password string) (string, string, error) {
    var user models.User
    if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
        return "", "", errors.New("invalid credentials")
    }
    
    // Verify password
    parts := splitHash(user.PasswordHash)
    if len(parts) != 2 {
        return "", "", errors.New("invalid password hash")
    }
    
    salt, _ := base64.StdEncoding.DecodeString(parts[0])
    storedHash, _ := base64.StdEncoding.DecodeString(parts[1])
    
    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    if !compareHash(hash, storedHash) {
        return "", "", errors.New("invalid credentials")
    }
    
    // Generate tokens
    accessToken, err := s.generateToken(user.ID, 15*time.Minute)
    if err != nil {
        return "", "", err
    }
    
    refreshToken, err := s.generateToken(user.ID, 7*24*time.Hour)
    if err != nil {
        return "", "", err
    }
    
    // Update user status
    s.db.Model(&user).Update("online", true)
    
    return accessToken, refreshToken, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return s.jwtSecret, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, errors.New("invalid token")
}

func (s *AuthService) generateToken(userID string, expiration time.Duration) (string, error) {
    claims := &Claims{
        UserID: userID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}

func (s *AuthService) GetUserByID(id string) (*models.User, error) {
    var user models.User
    if err := s.db.First(&user, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
    var user models.User
    if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

func (s *AuthService) SetUserOffline(userID string) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("online", false).Error
}

// UpdateProfile updates username and/or avatar for the given user. Empty avatar string clears it.
func (s *AuthService) UpdateProfile(userID string, username *string, avatar *string) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if username != nil {
		u := strings.TrimSpace(*username)
		if u == "" {
			return nil, errors.New("username cannot be empty")
		}
		if u != user.Username {
			// Username is no longer globally unique; allocate a new discriminator within the new username.
			d, err := allocateDiscriminator(s.db, u)
			if err != nil {
				return nil, err
			}
			updates["username"] = u
			updates["discriminator"] = d
		}
	}
	if avatar != nil {
		updates["avatar"] = *avatar
	}
	if len(updates) == 0 {
		return &user, nil
	}
	updates["updated_at"] = time.Now()
	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// RefreshAccessToken issues a new access token from a valid refresh JWT.
func (s *AuthService) RefreshAccessToken(refreshToken string) (string, error) {
    claims, err := s.ValidateToken(refreshToken)
    if err != nil {
        return "", err
    }
    return s.generateToken(claims.UserID, 15*time.Minute)
}

// AuthMiddleware validates Bearer JWT and sets userId on the Gin context.
func AuthMiddleware(s *AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        h := c.GetHeader("Authorization")
        if h == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing authorization header"})
            return
        }
        const prefix = "Bearer "
        if !strings.HasPrefix(h, prefix) {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid authorization header"})
            return
        }
        token := strings.TrimSpace(strings.TrimPrefix(h, prefix))
        claims, err := s.ValidateToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
            return
        }
        c.Set("userId", claims.UserID)
        c.Next()
    }
}

func generateID() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return base64.RawURLEncoding.EncodeToString(bytes)
}

func splitHash(hash string) []string {
    for i := 0; i < len(hash); i++ {
        if hash[i] == ':' {
            return []string{hash[:i], hash[i+1:]}
        }
    }
    return nil
}

func compareHash(a, b []byte) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}