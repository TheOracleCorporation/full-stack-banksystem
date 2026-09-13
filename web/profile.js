fetch("/api/profile")
    .then(response => {
        if (!response.ok) {
            throw new Error("Ошибка получения профиля");
        }

        return response.json();
    })
    .then(data => {

        document.getElementById("name").textContent =
            data.name;

        document.getElementById("email").textContent =
            data.email;

        document.getElementById("balance").textContent =
            data.balance + " ₽";
    })
    .catch(error => {
        document.getElementById("name").textContent =
            error.message;
    });


document.getElementById("backButton").addEventListener("click", function() {
    window.location.href = "/account.html";
});