package main

import (
	// Добавлен, если не было
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
	email := r.FormValue("email") // Используем email

	if username == "" || password == "" || email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Все обязательные поля должны быть заполнены", "status": "error"})
		return
	}

	mu.Lock()
	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Проверяем по email, так как он теперь ключ
	if _, exists := users[email]; exists {
		mu.Unlock()
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"message": "Email уже используется", "status": "error"})
		return
	}

	// 2. Обработка загруженного файла (логика не меняется)
	var photoPath string
	file, handler, err := r.FormFile("profile_photo")

	if err == nil {
		defer file.Close()

		uploadDir := filepath.Join("static", "uploads")
		// Генерируем уникальное имя файла
		uniqueFileName := fmt.Sprintf("%s_%d%s", username, time.Now().Unix(), filepath.Ext(handler.Filename))
		fullPath := filepath.Join(uploadDir, uniqueFileName)
		photoPath = "/uploads/" + uniqueFileName

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

	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Сохраняем данные по ключу email
	users[email] = UserData{
		Username:       username,
		Email:          email,
		HashedPassword: string(hashedPasswordBytes),
		PhotoPath:      photoPath,
	}
	mu.Unlock()

	log.Printf("✅ НОВЫЙ ПОЛЬЗОВАТЕЛЬ ДОБАВЛЕН: %s (Email: %s, Фото: %s)", username, email, photoPath)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Регистрация прошла успешно! Теперь войдите.", "status": "success"})
}

// loginHandler обрабатывает вход пользователя, устанавливая сессионную куки.
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

	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Проверяем наличие email, а не username
	if creds.Email == "" || creds.Password == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Введите email и пароль", "status": "error"})
		return
	}

	mu.Lock()
	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Ищем пользователя по Email
	userData, exists := users[creds.Email]
	mu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Неверный email или пароль", "status": "error"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(userData.HashedPassword), []byte(creds.Password))
	if err != nil {
		log.Printf("❌ Неудачная попытка входа для %s", creds.Email)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Неверный email или пароль", "status": "error"})
		return
	}

	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Сохраняем email в куке
	cookie := http.Cookie{
		Name:     "session_username", // Название куки оставляем для совместимости (но теперь хранит email)
		Value:    creds.Email,        // Храним Email
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, &cookie)

	log.Printf("✅ Успешный вход пользователя: %s. Установлена куки.", userData.Username)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Вход выполнен успешно!", "status": "success"})
}

// userHandler возвращает данные профиля (доступен только после аутентификации).
func userHandler(w http.ResponseWriter, r *http.Request) {
	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Получаем email из контекста
	email := r.Context().Value(userContextKey).(string)

	mu.Lock()
	// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Ищем пользователя по email
	userData, exists := users[email]
	mu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь не найден (по email)", "status": "error"})
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

// logoutHandler сбрасывает сессионную куки (логика не меняется).
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

// =======================================================================
// ✅ НОВЫЙ ОБРАБОТЧИК ДЛЯ ОБНОВЛЕНИЯ ПРОФИЛЯ
// =======================================================================

func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Допустим только метод POST", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// 1. Получаем email текущего пользователя из контекста
	oldEmail := r.Context().Value(userContextKey).(string)

	// 2. Парсинг формы (Максимальный размер запроса 10 MB)
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		log.Printf("❌ Ошибка парсинга формы обновления: %v", err)
		http.Error(w, "Слишком большой запрос", http.StatusBadRequest)
		return
	}

	// Получаем новые значения полей
	newUsername := r.FormValue("username") // Имя и Фамилия, объединенные в JS
	newEmail := r.FormValue("email")
	newPassword := r.FormValue("new_password") // Пароль

	if newUsername == "" || newEmail == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Имя и Email не могут быть пустыми", "status": "error"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	userData, exists := users[oldEmail]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь не найден", "status": "error"})
		return
	}

	// 3. Обработка загруженного файла (если есть)
	var newPhotoPath = userData.PhotoPath // Сохраняем старый путь по умолчанию
	file, handler, err := r.FormFile("profile_photo")

	if err == nil {
		defer file.Close()

		// Удаляем старое фото, если оно существует, для очистки диска
		if userData.PhotoPath != "" && userData.PhotoPath != newPhotoPath {
			oldFilePath := filepath.Join("static", userData.PhotoPath)
			if oldFilePath[0] == '/' {
				oldFilePath = oldFilePath[1:] // Убираем начальный слэш
			}
			os.Remove(oldFilePath)
		}

		// Сохраняем новое фото
		uploadDir := filepath.Join("static", "uploads")
		uniqueFileName := fmt.Sprintf("%s_%d%s", newUsername, time.Now().Unix(), filepath.Ext(handler.Filename))
		fullPath := filepath.Join(uploadDir, uniqueFileName)
		newPhotoPath = "/uploads/" + uniqueFileName

		dst, createErr := os.Create(fullPath)
		if createErr != nil {
			log.Printf("❌ Ошибка создания нового файла: %v", createErr)
		} else {
			defer dst.Close()
			if _, copyErr := io.Copy(dst, file); copyErr != nil {
				log.Printf("❌ Ошибка копирования нового файла: %v", copyErr)
			} else {
				log.Printf("✅ Новый файл успешно сохранен: %s", fullPath)
			}
		}
	} else if err != http.ErrMissingFile {
		log.Printf("❌ Ошибка при получении файла: %v", err)
		http.Error(w, "Ошибка при обработке файла", http.StatusInternalServerError)
		return
	}

	// 4. Обновление хеша пароля (если предоставлен)
	hashedPassword := userData.HashedPassword
	if newPassword != "" {
		hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("❌ Ошибка хеширования нового пароля: %v", err)
			http.Error(w, "Внутренняя ошибка сервера при хешировании", http.StatusInternalServerError)
			return
		}
		hashedPassword = string(hashedPasswordBytes)
	}

	// 5. Обновление структуры данных
	updatedData := UserData{
		Username:       newUsername,
		Email:          newEmail,
		HashedPassword: hashedPassword,
		PhotoPath:      newPhotoPath,
	}

	// 6. Обработка изменения Email (КЛЮЧЕВОЙ МОМЕНТ)
	if oldEmail != newEmail {
		// Проверяем, не занят ли новый email (другим пользователем)
		if _, exists := users[newEmail]; exists {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Новый Email уже занят другим пользователем", "status": "error"})
			return
		}

		// Удаляем старую запись и создаем новую с новым email
		delete(users, oldEmail)
		users[newEmail] = updatedData
		log.Printf("✅ Пользователь %s обновил Email с %s на %s", newUsername, oldEmail, newEmail)

		// 7. Если Email изменился, необходимо обновить сессионную куку
		cookie := http.Cookie{
			Name:     "session_username",
			Value:    newEmail,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		}
		http.SetCookie(w, &cookie)

	} else {
		// Email не изменился, просто обновляем текущую запись
		users[oldEmail] = updatedData
	}

	log.Printf("✅ Профиль пользователя %s успешно обновлен. (Email: %s)", updatedData.Username, updatedData.Email)

	if oldEmail != updatedData.Email || oldEmail == updatedData.Email {
		// Используем oldEmail, чтобы хаб мог найти клиента, у которого этот email в структуре
		hub.profileUpdate <- oldEmail
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":   "Настройки профиля успешно сохранены!",
		"status":    "success",
		"new_email": updatedData.Email, // Возвращаем новый email (для JS)
	})
}
