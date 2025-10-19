package main

import (
	"sync"
)

// --- Структуры Данных ---

// UserData хранит все данные о пользователе, включая хеш пароля и путь к фото.
type UserData struct {
	Email          string // Email теперь уникален и используется для входа
	HashedPassword string
	Username       string // Имя пользователя используется для отображения, но не для входа
	PhotoPath      string // Путь к файлу фотографии (например, /uploads/user_12345.jpg)
}

// UserCredentials используется для декодирования JSON-запросов.
// Внимание: для входа будем использовать Email и Password.
type UserCredentials struct {
	Username string `json:"username"` // По-прежнему нужно для registerHandler, но не для loginHandler
	Password string `json:"password"`
	Email    string `json:"email"` // КЛЮЧЕВОЕ ИЗМЕНЕНИЕ ДЛЯ ВХОДА
}

// --- Глобальное Состояние (Имитация DB) ---

var (
	// users - это наша карта "базы данных" в памяти: [email]UserData
	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: используем email как ключ
	users = make(map[string]UserData)
	// mu - Mutex для защиты доступа к карте users от гонок данных.
	mu sync.Mutex
)

// --- Контекст (для Middleware) ---

// contextKey используется как тип ключа для избежания коллизий в контексте.
type contextKey string

const userContextKey contextKey = "email" // КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Теперь контекстный ключ хранит email
