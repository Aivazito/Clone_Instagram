const ws = new WebSocket("ws://localhost:8080/ws");

const chatBox = document.getElementById("chatBox");
const input = document.getElementById("messageInput");
const sendBtn = document.getElementById("sendBtn");

ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);

    const el = document.createElement("div");
    el.classList.add("flex", "items-center", "gap-2", "mb-2");
    el.innerHTML = `
        <img src="${msg.photo_url}" alt="avatar" class="w-8 h-8 rounded-full">
        <div class="bg-gray-100 rounded-xl p-2">
            <strong>${msg.username}</strong><br>
            ${msg.text}<br>
            <small class="text-gray-400">${msg.timestamp}</small>
        </div>
    `;
    chatBox.appendChild(el);
    chatBox.scrollTop = chatBox.scrollHeight;
};

sendBtn.addEventListener("click", () => {
    const text = input.value.trim();
    if (text) {
        ws.send(text);
        input.value = "";
    }
});
