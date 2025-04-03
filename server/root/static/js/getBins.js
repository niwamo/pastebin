async function getBins() {
    var table = document.getElementById("recent-bins");
    try {
        const controller = new AbortController();
        const id = setTimeout(() => controller.abort(), 2000);
        let response = await fetch("/api/v1.0/getBins", {
            method: "GET",
            signal: controller.signal,
        });
        clearTimeout(id);
        if (response.ok) {
            const data = await response.json();
            data.map((item) => {
                var row = table.insertRow();
                var titleCell = row.insertCell();
                titleCell.textContent = item['title'];
                var valueCell = row.insertCell();
                valueCell.textContent = item['content'];
            });
        } else {
            console.log("getBins() response not OK!");
            alert("Failed to get bins!");
        }
    } catch (error) {
        console.log("Error in getBins fetch() call!");
        alert("Failed to get bins!");
    }
}

document.addEventListener("DOMContentLoaded", getBins);
