document.addEventListener('DOMContentLoaded', () => {
    
    // URL-ы для перенаправления. Используем надежные относительные/корневые пути.
    const LOGIN_PAGE = '/'; 
    const PROFILE_PAGE = '/templates/user_profile.html';
    const REDIRECT_DELAY = 1000; // 1 секунда задержки

    // Вспомогательная функция для безопасного получения элемента
    const getElement = (id) => document.getElementById(id);

    // ------------------------------------
    // 1. Обработка регистрации
    // ------------------------------------
    const registerForm = getElement('registerForm');
    const registerMessageElement = getElement('message');

    if (registerForm) {
        console.log('🔗 Регистрационная форма найдена. Подключение...');
        
        registerForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            // ✅ ИСПОЛЬЗУЕМ FormData для отправки текстовых полей и файла
            const formData = new FormData(registerForm);
            
            if (registerMessageElement) registerMessageElement.textContent = '';
            
            try {
                const response = await fetch('/register', {
                    method: 'POST',
                    // ⚠️ НЕ устанавливаем Content-Type, чтобы браузер использовал multipart/form-data
                    body: formData
                });

                const result = await response.json();

                if (response.ok) {
                    registerMessageElement.textContent = result.message;
                    registerMessageElement.style.color = 'green';
                    registerForm.reset();
                    
                    console.log(`✅ Регистрация успешна. Перенаправление на ${LOGIN_PAGE} через ${REDIRECT_DELAY / 1000}с...`);

                    // Успешная регистрация -> Перенаправление на страницу входа
                    setTimeout(() => {
                        window.location.href = LOGIN_PAGE; 
                    }, REDIRECT_DELAY);
                } else {
                    console.error('❌ Ошибка регистрации:', result.message);
                    registerMessageElement.textContent = `Ошибка: ${result.message || 'Неизвестная ошибка'}`;
                    registerMessageElement.style.color = 'red';
                }
            } catch (error) {
                console.error('❌ Ошибка сети при регистрации:', error);
                if (registerMessageElement) {
                    registerMessageElement.textContent = 'Произошла ошибка сети. Сервер недоступен.';
                    registerMessageElement.style.color = 'red';
                }
            }
        });
    }

    
    // ------------------------------------
    // 2. Обработка входа (Логина)
    // ------------------------------------
    const loginForm = getElement('loginForm');
    const loginMessageElement = getElement('loginMessage');

    if (loginForm) {
        console.log('🔗 Форма входа найдена. Подключение...');

        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            // Сбор данных формы
            const data = {
                username: getElement('username')?.value, 
                password: getElement('password')?.value
            };

            if (loginMessageElement) loginMessageElement.textContent = '';
            
            try {
                const response = await fetch('/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (response.ok) {
                    if (loginMessageElement) {
                         loginMessageElement.textContent = result.message;
                         loginMessageElement.style.color = 'green';
                         loginForm.reset();
                    }
                    console.log(`✅ Вход успешен. Перенаправление на ${PROFILE_PAGE} через ${REDIRECT_DELAY / 1000}с...`);

                    // Успешный вход -> Перенаправление на профиль
                    setTimeout(() => {
                        window.location.href = PROFILE_PAGE; 
                    }, REDIRECT_DELAY);
                    
                } else {
                    console.warn('⚠️ Ошибка входа:', result.message);
                    if (loginMessageElement) {
                        loginMessageElement.textContent = `Ошибка: ${result.message || 'Неверные данные'}`;
                        loginMessageElement.style.color = 'red';
                    }
                }
            } catch (error) {
                console.error('❌ Ошибка сети при входе:', error);
                if (loginMessageElement) {
                    loginMessageElement.textContent = 'Произошла ошибка сети. Сервер недоступен.';
                    loginMessageElement.style.color = 'red';
                }
            }
        });
    }

    // ------------------------------------
    // 3. Загрузка данных профиля и выход
    // ------------------------------------
    const displayUsernameElement = getElement('displayUsername');
    const userEmailDisplayElement = getElement('userEmailDisplay'); 
    const logoutBtn = getElement('logoutBtn');
    
    // ✅ НОВЫЙ ЭЛЕМЕНТ: Аватар пользователя, который мы обновили в HTML
    const userAvatarElement = getElement('userAvatar'); 
    
    // Проверяем, что мы на странице профиля
    if (displayUsernameElement) {
        async function loadUserProfile() {
            try {
                const response = await fetch('/user');
                const result = await response.json();

                if (response.ok) {
                    displayUsernameElement.textContent = result.username;
                    
                    // Отображаем Email
                    if (userEmailDisplayElement && result.email) {
                        userEmailDisplayElement.textContent = result.email;
                    } else if (userEmailDisplayElement) {
                        userEmailDisplayElement.textContent = 'Email не указан';
                    }
                    
                    // 🔑 КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Устанавливаем путь к фотографии
                    if (userAvatarElement && result.photo_url) {
                        // Go-сервер возвращает photo_url, например: "/uploads/user_12345.jpg"
                        userAvatarElement.src = result.photo_url;
                        // При ошибке загрузки (например, пользователь не загрузил фото)
                        // сработает onerror, который мы установили в HTML.
                    }

                    console.log(`👋 Пользователь авторизован: ${result.username}, Email: ${result.email}, Фото: ${result.photo_url || 'Нет'}`);
                } else {
                    // 401 Unauthorized от Go-сервера (куки нет или недействителен)
                    console.warn("⚠️ Сессия недействительна. Перенаправление на вход.");
                    window.location.href = LOGIN_PAGE;
                }
            } catch (error) {
                // Ошибка сети (сервер выключен)
                console.error('❌ Ошибка загрузки профиля. Перенаправление.', error);
                window.location.href = LOGIN_PAGE; 
            }
        }
        
        loadUserProfile();

        // Обработка выхода
        if (logoutBtn) {
            logoutBtn.addEventListener('click', async () => {
                try {
                    const response = await fetch('/logout', { method: 'POST' });
                    
                    if (response.ok) {
                        console.log('🚪 Выход успешен. Перенаправление на вход.');
                        window.location.href = LOGIN_PAGE; 
                    }
                } catch (error) {
                    console.error('❌ Ошибка выхода:', error);
                }
            });
        }
    }
});