package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io" // 🚀 НОВЫЙ ИМПОРТ для работы с файлами
	"log"
	"net/http"
	"os"            // 🚀 НОВЫЙ ИМПОРТ для работы с файлами и директориями
	"path/filepath" // 🚀 НОВЫЙ ИМПОРТ для безопасной работы с путями
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Структуры данных
type UserData struct {
	Email          string
	HashedPassword string
	Username       string // Используем как уникальный ключ
	PhotoPath      string // ✅ НОВОЕ ПОЛЕ: Путь к файлу фотографии
}

// UserCredentials остается для login, но не используется для register
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
	users = make(map[string]UserData)
	mu    sync.Mutex
)

// -------------------------
// Middleware и Вспомогательные функции (без изменений)
// -------------------------

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_username")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Не авторизован", "status": "error"})
			return
		}

		username := cookie.Value
		ctx := context.WithValue(r.Context(), userContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

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

	// 1. Парсинг формы и установка лимита размера (10 MB)
	const MAX_UPLOAD_SIZE = 10 << 20 // 10 MB
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		log.Printf("❌ Ошибка парсинга формы: %v", err)
		http.Error(w, fmt.Sprintf("Максимальный размер запроса %d MB", MAX_UPLOAD_SIZE/1048576), http.StatusBadRequest)
		return
	}

	// 2. Извлечение текстовых полей через r.FormValue()
	username := r.FormValue("username")
	password := r.FormValue("password")
	email := r.FormValue("email")

	// Проверка обязательных полей
	if username == "" || password == "" || email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Все обязательные поля должны быть заполнены", "status": "error"})
		return
	}

	mu.Lock()
	if _, exists := users[username]; exists {
		mu.Unlock()
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"message": "Имя пользователя уже занято", "status": "error"})
		return
	}
	// Не разблокируем здесь, чтобы сохранить блокировку для записи данных пользователя

	// 3. Обработка загруженного файла
	var photoPath string
	file, handler, err := r.FormFile("profile_photo") // "profile_photo" - атрибут 'name' из HTML

	if err == nil {
		// Файл был предоставлен
		defer file.Close()

		// 3.1. Создаем уникальный путь для сохранения
		// ВАЖНО: 'uploads' должно быть относительно места запуска сервера (../uploads)
		uploadDir := filepath.Join("..", "uploads")
		// Генерируем уникальное имя файла для безопасности
		uniqueFileName := fmt.Sprintf("%s_%d%s", username, time.Now().Unix(), filepath.Ext(handler.Filename))
		fullPath := filepath.Join(uploadDir, uniqueFileName)
		photoPath = "/uploads/" + uniqueFileName // Путь, который будет доступен через HTTP

		// 3.2. Сохраняем файл на диск
		dst, createErr := os.Create(fullPath)
		if createErr != nil {
			log.Printf("❌ Ошибка создания файла на диске: %v", createErr)
			// Продолжаем регистрацию, но без фото
		} else {
			defer dst.Close()
			if _, copyErr := io.Copy(dst, file); copyErr != nil {
				log.Printf("❌ Ошибка копирования файла: %v", copyErr)
			} else {
				log.Printf("✅ Файл успешно сохранен: %s", fullPath)
			}
		}

	} else if err != http.ErrMissingFile {
		// Ошибка, отличная от отсутствия файла (например, слишком большой размер)
		log.Printf("❌ Ошибка при получении файла: %v", err)
		mu.Unlock()
		http.Error(w, "Ошибка при обработке файла", http.StatusInternalServerError)
		return
	}

	// 4. Хеширование пароля
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ Ошибка хеширования пароля: %v", err)
		mu.Unlock()
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// 5. Сохранение пользователя в "базе данных"
	users[username] = UserData{
		Username:       username,
		Email:          email,
		HashedPassword: string(hashedPasswordBytes),
		PhotoPath:      photoPath, // ✅ Сохраняем путь к фото
	}
	mu.Unlock() // Разблокируем после записи

	log.Printf("✅ НОВЫЙ ПОЛЬЗОВАТЕЛЬ ДОБАВЛЕН: %s (Фото: %s)", username, photoPath)

	response := map[string]string{
		"message": "Регистрация прошла успешно! Теперь войдите.",
		"status":  "success",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// loginHandler (без изменений)
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

	err := bcrypt.CompareHashAndPassword([]byte(userData.HashedPassword), []byte(creds.Password))
	if err != nil {
		log.Printf("❌ Неудачная попытка входа для %s", creds.Username)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Неверное имя пользователя или пароль", "status": "error"})
		return
	}

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

// userHandler (обновлен для возврата пути к фото)
func userHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(userContextKey).(string)

	mu.Lock()
	userData, exists := users[username]
	mu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь не найден", "status": "error"})
		return
	}

	// Отправляем данные (включая PhotoPath!)
	response := map[string]string{
		"username":  userData.Username,
		"email":     userData.Email,
		"photo_url": userData.PhotoPath, // ✅ Возвращаем путь к фотографии
		"status":    "success",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// logoutHandler (без изменений)
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Допустим только метод POST", http.StatusMethodNotAllowed)
		return
	}
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
	// 1. Убеждаемся, что папка для загрузок существует
	uploadDir := filepath.Join("..", "uploads")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		log.Printf("Создание директории загрузок: %s", uploadDir)
		if err := os.Mkdir(uploadDir, 0755); err != nil {
			log.Fatalf("❌ Не удалось создать директорию загрузок: %v", err)
		}
	}

	// --- Обслуживание статических файлов ---
	// Обслуживаем корневой маршрут (/)
	http.Handle("/", http.FileServer(http.Dir("../templates")))
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("../templates"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("../js"))))

	// 2. ✅ НОВАЯ ДИРЕКТИВА: Обслуживаем папку /uploads/ для доступа к изображениям
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

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
