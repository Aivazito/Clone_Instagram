// ЛОГИН
document.getElementById("loginForm")?.addEventListener("submit", (e) => {
  e.preventDefault();
  const username = document.getElementById("username").value;
  const password = document.getElementById("password").value;

  const storedUser = JSON.parse(localStorage.getItem(username));
  if (storedUser && storedUser.password === password) {
    localStorage.setItem("loggedInUser", username);
    window.location.href = "user_profile.html";
  } else {
    alert("Неверное имя пользователя или пароль");
  }
});

// РЕГИСТРАЦИЯ
document.getElementById("registerForm")?.addEventListener("submit", (e) => {
  e.preventDefault();
  const username = document.getElementById("regUsername").value;
  const password = document.getElementById("regPassword").value;
  const email = document.getElementById("regEmail").value;

  if (localStorage.getItem(username)) {
    alert("Пользователь уже существует!");
  } else {
    localStorage.setItem(username, JSON.stringify({ password, email }));
    alert("Регистрация успешна!");
    window.location.href = "index.html";
  }
});

// ПРОФИЛЬ
if (document.getElementById("displayUsername")) {
  const user = localStorage.getItem("loggedInUser");
  if (!user) {
    window.location.href = "index.html";
  } else {
    document.getElementById("displayUsername").textContent = user;
  }

  document.getElementById("logoutBtn").addEventListener("click", () => {
    localStorage.removeItem("loggedInUser");
    window.location.href = "index.html";
  });
}
