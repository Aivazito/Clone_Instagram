package main

import (
	"context"
	"encoding/json"
	"log" // 💡 Добавил log для отладки
	"net/http"
)

// authMiddleware защищает маршруты, проверяя, установлена ли сессионная куки и существует ли пользователь.
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

		username := cookie.Value

		// 🔑 КЛЮЧЕВОЕ ИСПРАВЛЕНИЕ: Проверяем, существует ли пользователь с этим именем в системе.
		mu.Lock()
		_, exists := users[username]
		mu.Unlock()

		if !exists {
			// Пользователь не найден в текущей "базе данных" (map users)
			log.Printf("❌ Неудачная аутентификация: Пользователь %s не найден в базе данных.", username)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь не найден", "status": "error"})
			return
		}

		// Аутентификация успешна. Передаем имя пользователя в контексте.
		log.Printf("✅ Успешная аутентификация: %s", username)
		ctx := context.WithValue(r.Context(), userContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// ⚠️ ВАЖНО: Убедитесь, что 'mu', 'users' и 'userContextKey'
// доступны и правильно определены в вашем пакете 'main'.
// (Предполагается, что они определены в main.go)
