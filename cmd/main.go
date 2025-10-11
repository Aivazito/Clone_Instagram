package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./instagram_clone.db")
	if err != nil {
		log.Fatal(err)
	}

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE NOT NULL,
        hashed_password TEXT NOT NULL
};`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("Ошибка при создании таблицы: %s", err)
	}
	log.Println("База данных успешно инициализирована.")

}

func main() {
	initDB()    // Инициализация БД
	initStore() // Инициализация сессий (store)

	// Отображение формы регистрации
	http.HandleFunc("/register", registerFormHandler)

	// Обработка отправленной формы регистрации (POST)
	// Обратите внимание: registerHandler находится в пакете cmd, поэтому вызываем cmd.registerHandler
	http.HandleFunc("/register", cmd.registerHandler)

	// ... другие маршруты: /login, /profile

	fmt.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Пример функции для отображения HTML-формы регистрации (GET-запрос)
func registerFormHandler(w http.ResponseWriter, r *http.Request) {
	// Здесь код для рендеринга вашего HTML-шаблона с формой регистрации
	// tpl.ExecuteTemplate(w, "register.html", nil)
}
