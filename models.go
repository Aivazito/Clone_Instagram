package main

import (
	"sync"
)

// --- Структуры Данных ---

// UserData хранит все данные о пользователе, включая хеш пароля и путь к фото.
type UserData struct {
	Email          string
	HashedPassword string
	Username       string // Используется как уникальный ключ в карте
	PhotoPath      string // Путь к файлу фотографии (например, /uploads/user_12345.jpg)
}

// UserCredentials используется для декодирования JSON-запросов (входа).
type UserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"` // Не используется для входа, но полезно держать вместе
}

// --- Глобальное Состояние (Имитация DB) ---

var (
	// users - это наша карта "базы данных" в памяти: [username]UserData
	users = make(map[string]UserData)
	// mu - Mutex для защиты доступа к карте users от гонок данных.
	mu sync.Mutex
)

// --- Контекст (для Middleware) ---

// contextKey используется как тип ключа для избежания коллизий в контексте.
type contextKey string

const userContextKey contextKey = "username"
