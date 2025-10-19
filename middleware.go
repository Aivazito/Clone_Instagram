package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

// authMiddleware защищает маршруты, проверяя, установлена ли сессионная куки и существует ли пользователь по Email.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_username")
		if err != nil {
			// Кука отсутствует или истекла
			log.Printf("❌ Неудачная аутентификация: Кука 'session_username' отсутствует.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Не авторизован", "status": "error"})
			return
		}

		// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Значение куки - это теперь email
		email := cookie.Value

		// Проверяем, существует ли пользователь с этим email в системе.
		mu.Lock()
		_, exists := users[email] // Ищем по email
		mu.Unlock()

		if !exists {
			// Пользователь не найден в текущей "базе данных" (map users)
			log.Printf("❌ Неудачная аутентификация: Пользователь с email %s не найден в базе данных.", email)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь не найден", "status": "error"})
			return
		}

		// Аутентификация успешна. Передаем email в контексте.
		log.Printf("✅ Успешная аутентификация по Email: %s", email)
		// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Сохраняем email в контексте
		ctx := context.WithValue(r.Context(), userContextKey, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
