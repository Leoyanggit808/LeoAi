package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
	//jwt.RegisteredClaims：自动包含以下标准字段：
	//ExpiresAt（过期时间）
	//IssuedAt（签发时间）
	//NotBefore、Issuer、Subject 等（可选）
}

// GenerateToken 生成 JWT Token（24小时有效）
func GenerateToken(userID uint, secret string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret)) // SignedString：用 secret 对 claims 进行签名，生成最终的 Token 字符串
}

// ParseToken 解析并验证 Token
func ParseToken(tokenString, secret string) (*Claims, error) {
	//解析 Token 字符串 + 把解析后的数据填充到你自定义的 Claims 结构体中，同时验证 Token 的签名和有效性。
	token, err := jwt.ParseWithClaims(
		tokenString, //前端传来的 JWT 字符串
		&Claims{},   //自定义 Claims 结构体指针（必须传 &Claims{}）
		// 验证签名方法，防止算法篡改攻击
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
			//这里jwt.ParseWithClaims的第三个参数传的是keyfunc的类型，是jwt中定义的类型
			//我们只负责定义这个函数，并不需要自己去调用它。
			//JWT 库内部会自动调用这个函数，并自动传入一个 *jwt.Token 对象。
		})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的token")
	}

	return claims, nil
}
