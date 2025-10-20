package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// --- Настройки WebSocket ---
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// В проде ограничь источники!
		return true
	},
}

// Структура сообщения
type Message struct {
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	// Добавлено поле для определения типа сообщения (чат или обновление профиля)
	Type string `json:"type"`
}

// Клиент
type Client struct {
	conn *websocket.Conn
	send chan []byte
	user UserData // Текущие данные пользователя (для отправки)
}

// Менеджер чата
type ChatHub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	// ✅ ДОБАВЛЕНО: Хранение истории сообщений в памяти
	history []Message
	// ✅ ДОБАВЛЕНО: Канал для обновления данных о пользователях
	profileUpdate chan string // Канал для оповещения о смене Email
}

var hub = ChatHub{
	clients:       make(map[*Client]bool),
	broadcast:     make(chan []byte),
	register:      make(chan *Client),
	unregister:    make(chan *Client),
	history:       make([]Message, 0), // Инициализация истории
	profileUpdate: make(chan string),
}

// --- Запуск цикла хаба ---
func (h *ChatHub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("👤 %s подключился к чату (Email: %s)", client.user.Username, client.user.Email)

			// ✅ ОТПРАВКА ИСТОРИИ НОВОМУ КЛИЕНТУ
			if len(h.history) > 0 {
				historyMsg, _ := json.Marshal(map[string]interface{}{
					"type": "history",
					"data": h.history,
				})
				client.send <- historyMsg
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("🚪 %s вышел из чата", client.user.Username)
			}

		case message := <-h.broadcast:
			// ✅ СОХРАНЕНИЕ В ИСТОРИЮ
			var msg Message
			if err := json.Unmarshal(message, &msg); err == nil && msg.Type == "chat" {
				h.history = append(h.history, msg)
				// Ограничиваем историю, чтобы не занимать слишком много памяти
				if len(h.history) > 100 {
					h.history = h.history[1:]
				}
			}

			// Рассылка всем клиентам
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		// ✅ ДОБАВЛЕНА ЛОГИКА ОБНОВЛЕНИЯ ПРОФИЛЯ
		case oldEmail := <-h.profileUpdate:
			mu.Lock()
			newUserData, exists := users[oldEmail] // Пользователь был обновлен в handlers.go и теперь, возможно, имеет новый ключ
			mu.Unlock()

			if !exists {
				// Если пользователь не найден по старому email, возможно, он теперь по новому.
				// Проверяем всех клиентов, чтобы найти клиента с измененным профилем.
				// (Этот блок может быть сложным, т.к. oldEmail может быть уже удален. Лучше искать по всем клиентам.)
				// Оптимизация: Находим клиента по старому Email, обновляем его UserData и отправляем ему оповещение.

				// Ищем клиента, который только что изменил профиль
				for client := range h.clients {
					// Мы ищем клиента, у которого в его структуре *Client остался старый Email
					if client.user.Email == oldEmail {

						// Находим обновленные данные (по новому или старому Email)
						mu.Lock()
						newUserData, exists := users[newUserData.Email] // newUserData.Email содержит актуальный Email
						mu.Unlock()

						if exists {
							// Обновляем данные клиента в хабе
							client.user = newUserData
							log.Printf("🔄 Обновлены данные клиента %s в чате", client.user.Username)

							// Отправляем всем сообщение об обновлении (например, для изменения имени в чате)
							updateMsg, _ := json.Marshal(map[string]string{
								"type":      "user_update",
								"old_email": oldEmail,
								"new_email": newUserData.Email,
								"username":  newUserData.Username,
								"photo_url": newUserData.PhotoPath,
							})
							h.broadcast <- updateMsg
						}
					}
				}
			}
		}
	}
}

// --- WebSocket обработчик ---
func chatHandler(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userContextKey).(string)
	// ... (логика получения пользователя осталась прежней) ...
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mu.Lock()
	user, exists := users[email]
	mu.Unlock()
	if !exists {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка апгрейда:", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
		user: user,
	}
	hub.register <- client

	go client.writePump()
	go client.readPump()
}

// --- Получение сообщений от клиента ---
func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	for {
		// Читаем сообщение как обычный текст
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// ✅ ДОБАВЛЕНО: Декодируем сообщение, чтобы получить только текст (так как chat.html отправляет JSON)
		var incoming map[string]string
		if err := json.Unmarshal(msg, &incoming); err != nil {
			// Если пришло не JSON, просто считаем это текстом
			incoming = map[string]string{"text": string(msg)}
		}

		message := Message{
			Username:  c.user.Username,
			PhotoURL:  c.user.PhotoPath,
			Text:      incoming["text"],
			Timestamp: time.Now().Format("15:04"),
			Type:      "chat", // ✅ ДОБАВЛЕНО: Тип сообщения
		}

		jsonMsg, _ := json.Marshal(message)
		hub.broadcast <- jsonMsg
	}
}

// --- Отправка сообщений клиенту ---
func (c *Client) writePump() {
	// ... (логика writePump осталась прежней) ...
	defer c.conn.Close()
	for message := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
}

// --- Регистрация маршрута ---
func init() {
	go hub.run()
	http.HandleFunc("/ws", authMiddleware(chatHandler))
	fmt.Println("💬 WebSocket чат доступен по адресу: ws://localhost:8080/ws")
}
