document.addEventListener('DOMContentLoaded', () => {
    
    // URL-ы для перенаправления. Используем надежные относительные/корневые пути.
    const LOGIN_PAGE = '/'; 
    const PROFILE_PAGE = '/user_profile.html';
    const REDIRECT_DELAY = 1000; // 1 секунда задержки

    // Вспомогательная функция для безопасного получения элемента
    const getElement = (id) => document.getElementById(id);

    // ------------------------------------
    // 1. Обработка регистрации (Без изменений)
    // ------------------------------------
    const registerForm = getElement('registerForm');
    const registerMessageElement = getElement('message');

    if (registerForm) {
        console.log('🔗 Регистрационная форма найдена. Подключение...');
        
        registerForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            // Используем FormData для отправки текстовых полей и файла
            const formData = new FormData(registerForm);
            
            if (registerMessageElement) registerMessageElement.textContent = '';
            
            try {
                const response = await fetch('/register', {
                    method: 'POST',
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

            // Сбор email и password
            const data = {
                email: getElement('emailInput')?.value,
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
    // 3. Загрузка данных профиля и обработка выхода
    // ------------------------------------
    const displayUsernameElement = getElement('displayUsername');
    const userEmailDisplayElement = getElement('userEmailDisplay'); 
    const logoutBtn = getElement('logoutBtn');
    const userAvatarElement = getElement('userAvatar'); 
    
    // ✅ Элементы формы профиля
    const profileSettingsForm = getElement('profileSettingsForm');
    const firstNameInput = getElement('firstName');
    const lastNameInput = getElement('lastName');
    const emailInput = getElement('email');
    const newPasswordInput = getElement('newPassword');
    const userAvatarMain = getElement('userAvatarMain');
    const displayUsernameMain = getElement('displayUsernameMain');
    const userEmailDisplayMain = getElement('userEmailDisplayMain');
    const profileMessageElement = getElement('profileMessage'); // Элемент для вывода сообщений

    // Проверяем, что мы на странице профиля
    if (displayUsernameElement) {
        // ✅ Функция для загрузки и обновления всех элементов
        async function loadUserProfile() {
            try {
                const response = await fetch('/user');
                const result = await response.json();

                if (response.ok) {
                    // Разделяем полное имя на Имя и Фамилию 
                    const currentFullUsername = result.username || '';
                    const parts = currentFullUsername.split(' ');
                    const firstName = parts[0] || '';
                    const lastName = parts.slice(1).join(' ') || '';

                    // Обновляем данные в боковом меню
                    displayUsernameElement.textContent = currentFullUsername;
                    userEmailDisplayElement.textContent = result.email;
                    
                    // Обновляем данные в главной секции (если элементы существуют)
                    if (displayUsernameMain) displayUsernameMain.textContent = currentFullUsername;
                    if (userEmailDisplayMain) userEmailDisplayMain.textContent = result.email;
                    
                    // ✅ Обновляем форму настроек
                    if (firstNameInput) firstNameInput.value = firstName;
                    if (lastNameInput) lastNameInput.value = lastName;
                    if (emailInput) emailInput.value = result.email;
                    
                    // Обновляем аватары
                    if (userAvatarElement && result.photo_url) userAvatarElement.src = result.photo_url;
                    if (userAvatarMain && result.photo_url) userAvatarMain.src = result.photo_url;


                    console.log(`👋 Пользователь авторизован: ${result.username}, Email: ${result.email}, Фото: ${result.photo_url || 'Нет'}`);
                } else {
                    console.warn("⚠️ Сессия недействительна. Перенаправление на вход.");
                    window.location.href = LOGIN_PAGE;
                }
            } catch (error) {
                console.error('❌ Ошибка загрузки профиля. Перенаправление.', error);
                window.location.href = LOGIN_PAGE; 
            }
        }
        
        loadUserProfile();
        
        // =======================================================================
        // ✅ ОБРАБОТЧИК СОХРАНЕНИЯ НАСТРОЕК ПРОФИЛЯ
        // =======================================================================
        if (profileSettingsForm) {
            profileSettingsForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                
                // Собираем данные формы, включая файл
                const formData = new FormData();
                
                // Формируем полное имя: Имя + Фамилия
                const fullName = (firstNameInput.value.trim() + ' ' + lastNameInput.value.trim()).trim();

                formData.append('username', fullName);
                formData.append('email', emailInput.value);
                
                // Добавляем пароль только если он был введен
                if (newPasswordInput.value) {
                    formData.append('new_password', newPasswordInput.value);
                }

                // Добавляем файл фото, если он выбран
                const photoFile = getElement('photoUpload').files[0];
                if (photoFile) {
                    formData.append('profile_photo', photoFile);
                }
                
                if (profileMessageElement) {
                    profileMessageElement.textContent = 'Сохранение...';
                    profileMessageElement.style.color = 'orange';
                }

                try {
                    const response = await fetch('/user/update', {
                        method: 'POST',
                        // НЕ устанавливаем Content-Type, чтобы браузер использовал multipart/form-data
                        body: formData 
                    });

                    const result = await response.json();

                    if (response.ok) {
                        if (profileMessageElement) {
                             profileMessageElement.textContent = result.message;
                             profileMessageElement.style.color = 'green';
                        }
                        newPasswordInput.value = ''; // Очищаем поле пароля

                        // Если email изменился, перезагружаем страницу, чтобы обновить куку и данные
                        if (result.new_email && result.new_email !== emailInput.value) {
                             console.log('✅ Email изменен. Перезагрузка для обновления сессии...');
                             setTimeout(() => window.location.reload(), 500);
                        } else {
                            // Если успешно, но без смены email, просто обновляем отображение
                            loadUserProfile(); 
                        }
                    } else {
                        console.error('❌ Ошибка сохранения:', result.message);
                        if (profileMessageElement) {
                             profileMessageElement.textContent = `Ошибка: ${result.message || 'Не удалось сохранить настройки'}`;
                             profileMessageElement.style.color = 'red';
                        }
                    }
                } catch (error) {
                    console.error('❌ Ошибка сети при сохранении профиля:', error);
                    if (profileMessageElement) {
                         profileMessageElement.textContent = 'Произошла ошибка сети. Сервер недоступен.';
                         profileMessageElement.style.color = 'red';
                    }
                }
            });
        }


        // Обработка выхода (Без изменений)
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