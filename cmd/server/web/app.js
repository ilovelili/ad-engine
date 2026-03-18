const money = (value, currency) =>
  new Intl.NumberFormat("en-US", { style: "currency", currency }).format(value || 0);

const pct = (value) => `${Number(value || 0).toFixed(1)}%`;

const allocationBars = document.getElementById("allocationBars");
const performanceTable = document.getElementById("performanceTable");
const rebalanceButton = document.getElementById("rebalanceButton");

async function fetchDashboard() {
  const response = await fetch("/api/v1/dashboard");
  if (!response.ok) {
    throw new Error(`dashboard request failed: ${response.status}`);
  }
  return response.json();
}

function renderDashboard(snapshot) {
  document.getElementById("campaignName").textContent = snapshot.name;
  document.getElementById("campaignMeta").textContent =
    `${snapshot.status} • ${snapshot.goal} • ${snapshot.currency}`;
  document.getElementById("totalBudget").textContent = money(snapshot.totalBudget, snapshot.currency);
  document.getElementById("remainingBudget").textContent = money(snapshot.remaining, snapshot.currency);
  document.getElementById("lastRebalance").textContent = new Date(snapshot.lastRebalance).toLocaleTimeString();

  allocationBars.innerHTML = snapshot.platforms
    .map(
      (platform) => `
        <article class="allocation-card platform-${platform.platform}">
          <div class="allocation-card-header">
            <h3>${platform.platform}</h3>
            <strong>${pct(platform.allocationPct)}</strong>
          </div>
          <div class="bar-track">
            <div class="bar-fill" style="width: ${platform.allocationPct}%"></div>
          </div>
          <p>Spend ${money(platform.spend, snapshot.currency)} • Revenue ${money(platform.revenue, snapshot.currency)}</p>
        </article>
      `
    )
    .join("");

  performanceTable.innerHTML = snapshot.platforms
    .map(
      (platform) => `
        <tr>
          <td>${platform.platform}</td>
          <td>${pct(platform.allocationPct)}</td>
          <td>${money(platform.spend, snapshot.currency)}</td>
          <td>${pct(platform.ctr)}</td>
          <td>${platform.roas.toFixed(2)}x</td>
          <td>${platform.conversions}</td>
          <td>${platform.publishedAds}</td>
        </tr>
      `
    )
    .join("");
}

async function load() {
  try {
    const snapshot = await fetchDashboard();
    renderDashboard(snapshot);
  } catch (error) {
    document.getElementById("campaignName").textContent = "Unable to load dashboard";
    document.getElementById("campaignMeta").textContent = error.message;
  }
}

rebalanceButton.addEventListener("click", async () => {
  rebalanceButton.disabled = true;
  rebalanceButton.textContent = "Rebalancing...";
  try {
    const response = await fetch("/api/v1/rebalance", { method: "POST" });
    if (!response.ok) {
      throw new Error(`rebalance request failed: ${response.status}`);
    }
    renderDashboard(await response.json());
  } catch (error) {
    document.getElementById("campaignMeta").textContent = error.message;
  } finally {
    rebalanceButton.disabled = false;
    rebalanceButton.textContent = "Run rebalance now";
  }
});

load();
setInterval(load, 8000);
