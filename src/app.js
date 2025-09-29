const appBody = document.getElementById('app-body');
        const loginContainer = document.getElementById('login-container');
        const signupContainer = document.getElementById('signup-container');
        const profileContainer = document.getElementById('profile-container');
        const toggleBlock = document.getElementById('toggle-block'); 
        const toggleToSignup = document.getElementById('toggle-to-signup');
        const toggleToLogin = document.getElementById('toggle-to-login');
        const loginForm = document.getElementById('login-form'); 
        const goToLoginButton = document.getElementById('go-to-login'); 

        /**
         * Переключает отображение между Входом, Регистрацией и Профилем.
         * @param {string} view - 'login', 'signup', или 'profile'
         */
        function showView(view) {
            // Скрываем все контейнеры
            [loginContainer, signupContainer, profileContainer, toggleBlock].forEach(el => el.classList.add('hidden'));
            
            // Устанавливаем правильный вид и настраиваем центрирование
            appBody.classList.remove('profile-view-active');

            if (view === 'login') {
                loginContainer.classList.remove('hidden');
                toggleBlock.classList.remove('hidden');
                toggleToSignup.classList.remove('hidden');
                toggleToLogin.classList.add('hidden');
                document.title = 'Вход Instagram';
            } else if (view === 'signup') {
                signupContainer.classList.remove('hidden');
                toggleBlock.classList.remove('hidden');
                toggleToSignup.classList.add('hidden');
                toggleToLogin.classList.remove('hidden');
                document.title = 'Регистрация Instagram';
            } else if (view === 'profile') {
                profileContainer.classList.remove('hidden');
                appBody.classList.add('profile-view-active');
                document.title = 'Профиль | 404504';
            }
        }

        // 1. Обработчик: Вход -> Профиль (Симуляция)
        loginForm.addEventListener('submit', (e) => {
            e.preventDefault(); 
            showView('profile');
        });

        // 2. Обработчик: Вход <-> Регистрация
        toggleToSignup.addEventListener('click', (e) => {
            e.preventDefault();
            showView('signup');
        });

        // 3. Обработчик: Регистрация <-> Вход
        toggleToLogin.addEventListener('click', (e) => {
            e.preventDefault();
            showView('login');
        });
        
        // 4. Обработчик: Профиль -> Вход (Симуляция выхода)
        goToLoginButton.addEventListener('click', (e) => {
            e.preventDefault();
            showView('login');
        });
        
        // Инициализация
        window.onload = () => showView('login');