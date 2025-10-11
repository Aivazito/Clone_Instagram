package cmd

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// В этой функции вам нужно получить username и password из запроса (r)
func registerHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Проверяем метод
	if r.Method != http.MethodPost {
		// Если не POST, перенаправляем на страницу регистрации
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	// 2. Получаем данные формы
	// Эти имена ("username", "password") должны совпадать с атрибутами name в вашей HTML-форме!
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Имя пользователя и пароль не могут быть пустыми.", http.StatusBadRequest)
		return
	}

	// 3. Хешируем пароль (Ваш код)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Ошибка при хешировании пароля", http.StatusInternalServerError)
		// В случае ошибки просто прекращаем выполнение, ничего не возвращая
		return
	}

	// 4. Сохраняем в БД (Вам нужно убедиться, что переменная db доступна в этом пакете/файле)
	// Предполагая, что db - это глобальная переменная, инициализированная в main.go
	// Если db находится в другом пакете, его нужно импортировать
	_, err = db.Exec("INSERT INTO users (username, hashed_password) VALUES (?, ?)",
		username, string(hashedPassword))

	if err != nil {
		// Обработка ошибки UNIQUE constraint (уже существующий пользователь)
		// В реальном приложении нужно проверить тип ошибки. Здесь упрощено.
		http.Error(w, "Ошибка: имя пользователя уже занято.", http.StatusConflict)
		return
	}

	// 5. Установка сессии (Как обсуждали в Шаге 3, добавляем сюда код для создания куки/сессии)
	// ... (session.Values[userKey] = username; session.Save(r, w))

	// 6. Перенаправляем на страницу профиля
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
