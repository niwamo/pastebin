document.addEventListener("DOMContentLoaded", async function () {
    const newBinForm = document.getElementById("newBin");
    newBinForm.addEventListener("submit", async function (event) {
        event.preventDefault(); // Prevent default form submission
        let formData = new FormData(this);
        try {
            const controller = new AbortController();
            const id = setTimeout(() => controller.abort(), 2000);
            let response = await fetch("/api/v1.0/newBin", {
                method: "POST",
                headers: {
                    "Content-Type": "application/x-www-form-urlencoded",
                },
                body: new URLSearchParams(formData).toString(),
                signal: controller.signal,
            });
            clearTimeout(id);
            if (response.ok) {
                var table = document.getElementById("recent-bins");
                var row = table.insertRow();
                var titleCell = row.insertCell();
                titleCell.textContent = formData.get("title");
                var contentCell = row.insertCell();
                contentCell.textContent = formData.get("content");
                newBinForm.reset();
            } else {
                alert("Submission failed!");
            }
        } catch (error) {
            alert("Submission failed!");
        }
    });
});
