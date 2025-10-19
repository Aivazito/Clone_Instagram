package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// --- Инициализация Файловой Системы ---

	// Указываем путь к папке загрузок (static/uploads)
	uploadDir := filepath.Join("static", "uploads")

	// Проверяем и создаем директорию для загрузок, если она не существует
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		log.Printf("Создание директории загрузок: %s", uploadDir)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			log.Fatalf("❌ Не удалось создать директорию загрузок: %v", err)
		}
	}

	// --- Обслуживание Статических Файлов ---

	// 1. Главный маршрут (/)
	// Обслуживаем содержимое папки static/templates по корневому пути (/).
	http.Handle("/", http.FileServer(http.Dir("static/templates")))

	// 2. JavaScript (/js/)
	// Обслуживаем /js/ из папки static/js
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("static/js"))))

	// 3. Загруженные файлы (/uploads/)
	// Обслуживаем /uploads/ из папки static/uploads. Здесь хранятся фото профиля.
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	// --- Обработчики API и Маршрутизация ---

	// Маршруты без защиты (открыты для всех)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)

	// Защищенные маршруты (требуют аутентификации через authMiddleware)
	http.HandleFunc("/user", authMiddleware(userHandler))
	// ✅ ДОБАВЛЕН НОВЫЙ МАРШРУТ ДЛЯ ОБНОВЛЕНИЯ ПРОФИЛЯ
	http.HandleFunc("/user/update", authMiddleware(updateProfileHandler))

	// --- Запуск Сервера ---

	fmt.Println("🚀 Сервер запущен на http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("❌ Ошибка запуска сервера: ", err)
	}
}
