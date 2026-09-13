document.getElementById("loginForm").addEventListener("submit", function(event) {

    event.preventDefault();

    const email = document.getElementById("email").value;
    const password = document.getElementById("password").value;

    fetch("/api/login", {
        method: "POST",

        headers: {
            "Content-Type": "application/json"
        },

        body: JSON.stringify({
            email: email,
            password: password
        })
    })
    .then(response => {

        if (!response.ok) {
            throw new Error("Неверный email или пароль");
        }

        return response.json();
    })
    .then(data => {

        document.getElementById("message").textContent =
            data.message;

        window.location.href = "/account.html";
    })
    .catch(error => {

        document.getElementById("message").textContent =
            error.message;
    });
});