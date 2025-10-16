package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const MAX_UPLOAD_SIZE = 10 << 20 // 10 MB

// --- Вспомогательные функции ---

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// --- Обработчики API ---

// registerHandler обрабатывает регистрацию пользователя и загрузку фото.
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Допустим только метод POST", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// 1. Парсинг формы
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		log.Printf("❌ Ошибка парсинга формы: %v", err)
		http.Error(w, fmt.Sprintf("Максимальный размер запроса %d MB", MAX_UPLOAD_SIZE/1048576), http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	email := r.FormValue("email")

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

	// 2. Обработка загруженного файла
	var photoPath string
	file, handler, err := r.FormFile("profile_photo") // Имя поля в форме

	if err == nil {
		defer file.Close()

		uploadDir := filepath.Join("static", "uploads")
		// Генерируем уникальное имя файла для безопасности
		uniqueFileName := fmt.Sprintf("%s_%d%s", username, time.Now().Unix(), filepath.Ext(handler.Filename))
		fullPath := filepath.Join(uploadDir, uniqueFileName)
		photoPath = "/uploads/" + uniqueFileName // Путь, доступный через HTTP

		// Сохраняем файл на диск
		dst, createErr := os.Create(fullPath)
		if createErr != nil {
			log.Printf("❌ Ошибка создания файла на диске: %v. Продолжаем без фото.", createErr)
		} else {
			defer dst.Close()
			if _, copyErr := io.Copy(dst, file); copyErr != nil {
				log.Printf("❌ Ошибка копирования файла: %v. Продолжаем без фото.", copyErr)
			} else {
				log.Printf("✅ Файл успешно сохранен: %s", fullPath)
			}
		}

	} else if err != http.ErrMissingFile {
		log.Printf("❌ Ошибка при получении файла: %v", err)
		mu.Unlock()
		http.Error(w, "Ошибка при обработке файла", http.StatusInternalServerError)
		return
	}

	// 3. Хеширование и сохранение пользователя
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ Ошибка хеширования пароля: %v", err)
		mu.Unlock()
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	users[username] = UserData{
		Username:       username,
		Email:          email,
		HashedPassword: string(hashedPasswordBytes),
		PhotoPath:      photoPath,
	}
	mu.Unlock()

	log.Printf("✅ НОВЫЙ ПОЛЬЗОВАТЕЛЬ ДОБАВЛЕН: %s (Фото: %s)", username, photoPath)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Регистрация прошла успешно! Теперь войдите.", "status": "success"})
}

// loginHandler обрабатывает вход пользователя, устанавливая сессионную куки.
func loginHandler(w http.ResponseWriter, r *http.Request) {
	// ... (Ваша существующая логика loginHandler без изменений) ...

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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Вход выполнен успешно!", "status": "success"})
}

// userHandler возвращает данные профиля (доступен только после аутентификации).
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

	// Отправляем данные, включая PhotoPath
	response := map[string]string{
		"username":  userData.Username,
		"email":     userData.Email,
		"photo_url": userData.PhotoPath,
		"status":    "success",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// logoutHandler сбрасывает сессионную куки.
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
