package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Структуры данных
type UserData struct {
	Email          string
	HashedPassword string
	Username       string // Используем как уникальный ключ
}

type UserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// Контекст для передачи данных пользователя через middleware
type contextKey string

const userContextKey contextKey = "username"

// Имитация базы данных в памяти
var (
	// users - Ключ: Username (уникальный), Значение: UserData
	users = make(map[string]UserData)
	mu    sync.Mutex
)

// -------------------------
// Middleware
// -------------------------

// authMiddleware проверяет наличие куки сессии и передает имя пользователя в контекст.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_username")
		if err != nil {
			// Если куки нет, возвращаем ошибку 401
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Не авторизован", "status": "error"})
			return
		}

		username := cookie.Value

		// Передаем имя пользователя в контекст запроса
		ctx := context.WithValue(r.Context(), userContextKey, username)

		// Продолжаем обработку
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// -------------------------
// Вспомогательные функции
// -------------------------

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// -------------------------
// Обработчики маршрутов
// -------------------------

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Допустим только метод POST", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var creds UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Неверный формат JSON в теле запроса", http.StatusBadRequest)
		return
	}

	if creds.Username == "" || creds.Password == "" || creds.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Все поля обязательны", "status": "error"})
		return
	}

	mu.Lock()
	defer mu.Unlock() // Гарантируем разблокировку

	if _, exists := users[creds.Username]; exists {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"message": "Имя пользователя уже занято", "status": "error"})
		return
	}

	// 1. Хеширование пароля
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ Ошибка хеширования пароля: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// 2. Сохранение пользователя
	users[creds.Username] = UserData{
		Username:       creds.Username,
		Email:          creds.Email,
		HashedPassword: string(hashedPasswordBytes),
	}

	log.Printf("✅ НОВЫЙ ПОЛЬЗОВАТЕЛЬ ДОБАВЛЕН: %s", creds.Username)

	response := map[string]string{
		"message": "Регистрация прошла успешно! Теперь войдите.",
		"status":  "success",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Допустим только метод POST", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var creds UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Неверный формат JSON в теле запроса", http.StatusBadRequest)
		return
	}

	if creds.Username == "" || creds.Password == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Введите имя пользователя и пароль", "status": "error"})
		return
	}

	mu.Lock()
	userData, exists := users[creds.Username]
	mu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Неверное имя пользователя или пароль", "status": "error"})
		return
	}

	// 1. Сравнение пароля
	err := bcrypt.CompareHashAndPassword([]byte(userData.HashedPassword), []byte(creds.Password))
	if err != nil {
		log.Printf("❌ Неудачная попытка входа для %s", creds.Username)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Неверное имя пользователя или пароль", "status": "error"})
		return
	}

	// 2. Успешный вход! Устанавливаем куки сессии
	cookie := http.Cookie{
		Name:     "session_username",
		Value:    creds.Username,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, &cookie)

	log.Printf("✅ Успешный вход пользователя: %s. Установлена куки.", creds.Username)

	response := map[string]string{
		"message": "Вход выполнен успешно!",
		"status":  "success",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	// Имя пользователя уже получено из куки и передано в контекст через authMiddleware
	username := r.Context().Value(userContextKey).(string)

	mu.Lock()
	userData, exists := users[username]
	mu.Unlock()

	if !exists {
		// Теоретически невозможно, если куки верный, но на всякий случай
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь не найден", "status": "error"})
		return
	}

	// Отправляем данные (только безопасные!)
	response := map[string]string{
		"username": userData.Username,
		"email":    userData.Email,
		"status":   "success",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Допустим только метод POST", http.StatusMethodNotAllowed)
		return
	}
	// Устанавливаем куки с истекшим сроком действия
	expiredCookie := http.Cookie{
		Name:     "session_username",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, &expiredCookie)

	log.Printf("🚫 Пользователь вышел.")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Выход выполнен", "status": "success"})
}

// -------------------------
// Главная функция (main)
// -------------------------

func main() {
	// --- Обслуживание статических файлов ---
	// Учитывая, что main.go находится в 'cmd/', нужно подняться на уровень выше: '../'

	// Обслуживаем корневой маршрут (/), чтобы http://localhost:8080/ открывал templates/index.html
	http.Handle("/", http.FileServer(http.Dir("../templates")))

	// Обслуживаем папку /templates/ для ссылок типа <a href="/templates/reg.html">
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("../templates"))))

	// Обслуживаем папку /js/ для <script src="/js/app.js">
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("../js"))))

	// --- Обработчики API ---
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)

	// ✨ Применяем middleware для защиты маршрута профиля
	http.HandleFunc("/user", authMiddleware(userHandler))

	// Запуск веб-сервера
	fmt.Println("🚀 Сервер запущен на http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("❌ Ошибка запуска сервера: ", err)
	}
}
