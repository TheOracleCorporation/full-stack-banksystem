document.getElementById("registerForm").addEventListener("submit", function(event) {
    event.preventDefault();

    const name = document.getElementById("name").value;
    const email = document.getElementById("email").value;
    const password = document.getElementById("password").value;

    fetch("/api/register", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            name: name,
            email: email,
            password: password
        })
    })
    .then(response => {
        if (!response.ok) {
            throw new Error("Пользователь с таким email уже существует");
        }

        return response.json();
    })
    .then(data => {
        document.getElementById("message").textContent = data.message;

        setTimeout(function() {
            window.location.href = "/";
        }, 1000);
    })
    .catch(error => {
        document.getElementById("message").textContent = error.message;
    });
});