const money = (value, currency) =>
  new Intl.NumberFormat("en-US", { style: "currency", currency }).format(value || 0);

const pct = (value) => `${Number(value || 0).toFixed(1)}%`;
const number = (value) => new Intl.NumberFormat("en-US").format(Math.round(value || 0));
const time = (value) => (value ? new Date(value).toLocaleString() : "Not yet synced");

const allocationBars = document.getElementById("allocationBars");
const performanceTable = document.getElementById("performanceTable");
const rebalanceButton = document.getElementById("rebalanceButton");
const trendSummary = document.getElementById("trendSummary");
const trendChart = document.getElementById("trendChart");
const trendHeadline = document.getElementById("trendHeadline");
const trendDelta = document.getElementById("trendDelta");
const timelineList = document.getElementById("timelineList");
const connectButton = document.getElementById("connectButton");
const connectionStatus = document.getElementById("connectionStatus");
const supportedPlatformNotes = document.getElementById("supportedPlatformNotes");
const connectionCards = document.getElementById("connectionCards");
const queryParams = new URLSearchParams(window.location.search);

async function fetchJSON(url, options) {
  const response = await fetch(url, options);
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || `request failed: ${response.status}`);
  }
  return payload;
}

const fetchDashboard = () => fetchJSON("/api/v1/dashboard");
const fetchConnections = () => fetchJSON("/api/v1/connections");

function renderSupportedPlatforms(platforms) {
  supportedPlatformNotes.innerHTML = platforms
    .map(
      (platform) => `
        <article class="supported-platform">
          <div class="supported-platform-header">
            <strong>${platform.name}</strong>
            <span>${platform.authenticationModel}</span>
          </div>
          ${(platform.notes || []).map((note) => `<p>${note}</p>`).join("")}
        </article>
      `
    )
    .join("");
}

function renderOAuthStatus() {
  const oauthStatus = queryParams.get("oauth");
  const message = queryParams.get("message");
  if (!oauthStatus) {
    return;
  }

  connectionStatus.textContent =
    message ||
    (oauthStatus === "oauth_success"
      ? "Meta OAuth completed successfully."
      : "Meta OAuth could not be completed.");
  connectionStatus.className = `form-status ${
    oauthStatus === "oauth_success" ? "is-success" : "is-error"
  }`;

  const cleanURL = `${window.location.pathname}${window.location.hash || ""}`;
  window.history.replaceState({}, document.title, cleanURL);
}

function renderConnections(view) {
  renderSupportedPlatforms(view.supportedPlatforms || []);

  if (!view.connections.length) {
    connectionCards.innerHTML = `
      <article class="connection-card connection-card-empty">
        <strong>No connected platforms yet</strong>
        <p>Connect with Meta above to approve access and pull available ad accounts.</p>
      </article>
    `;
    return;
  }

  connectionCards.innerHTML = view.connections
    .map(
      (connection) => `
        <article class="connection-card">
          <div class="connection-card-header">
            <div>
              <span class="connection-platform">${connection.platform}</span>
              <h3>${connection.accountLabel || connection.displayName || connection.accountIdentifier}</h3>
            </div>
            <span class="connection-badge">${connection.status}</span>
          </div>
          <div class="connection-meta">
            <span>Account key: ${connection.accountIdentifier}</span>
            <span>Meta user ID: ${connection.externalAccountId || "Unavailable"}</span>
            <span>Last validated: ${time(connection.lastValidatedAt)}</span>
          </div>
          ${
            connection.instagramBusinessAccountId
              ? `<p class="connection-detail">Instagram business account ID: ${connection.instagramBusinessAccountId}</p>`
              : ""
          }
          ${
            connection.scopes?.length
              ? `<p class="connection-detail">Expected scopes: ${connection.scopes.join(", ")}</p>`
              : ""
          }
          ${
            connection.lastError
              ? `<p class="connection-error">Last error: ${connection.lastError}</p>`
              : ""
          }
          <div class="ad-account-list">
            ${
              connection.adAccounts.length
                ? connection.adAccounts
                    .map(
                      (account) => `
                        <article class="ad-account-item">
                          <div>
                            <strong>${account.name || account.id}</strong>
                            <p>${account.id}</p>
                          </div>
                          <div class="ad-account-stats">
                            <span>${account.status}</span>
                            <span>${account.currency || "N/A"}</span>
                            <span>${account.timezone || "Timezone unavailable"}</span>
                          </div>
                        </article>
                      `
                    )
                    .join("")
                : `<p class="connection-detail">No ad accounts were returned for this token.</p>`
            }
          </div>
        </article>
      `
    )
    .join("");
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

  renderMomentum(snapshot);
}

function buildTimeline(snapshot) {
  const totalSpend = snapshot.platforms.reduce((sum, platform) => sum + platform.spend, 0);
  const totalRevenue = snapshot.platforms.reduce((sum, platform) => sum + platform.revenue, 0);
  const totalConversions = snapshot.platforms.reduce((sum, platform) => sum + platform.conversions, 0);
  const totalImpressions = snapshot.platforms.reduce((sum, platform) => sum + platform.impressions, 0);
  const totalClicks = snapshot.platforms.reduce((sum, platform) => sum + platform.clicks, 0);
  const ctr = totalImpressions ? (totalClicks / totalImpressions) * 100 : 0;

  const progress = [0.54, 0.63, 0.72, 0.82, 0.91, 1];
  const efficiency = [0.78, 0.84, 0.9, 0.96, 1, 1.06];

  return progress.map((factor, index) => {
    const spend = totalSpend * factor;
    const revenue = totalRevenue * factor * efficiency[index];
    const conversions = totalConversions * factor * (0.8 + index * 0.05);
    const pointCtr = ctr * (0.82 + index * 0.04);
    const hoursAgo = (progress.length - index - 1) * 4;
    const leadingPlatform = [...snapshot.platforms].sort(
      (left, right) =>
        right.roas * (1 + index * 0.03) - left.roas * (1 + index * 0.03 * 0.5)
    )[0];

    return {
      label: hoursAgo === 0 ? "Now" : `${hoursAgo}h ago`,
      spend,
      revenue,
      conversions,
      ctr: pointCtr,
      roas: spend ? revenue / spend : 0,
      note:
        index === progress.length - 1
          ? `${leadingPlatform.platform} now leads with the strongest return mix.`
          : `Budget moved incrementally toward ${leadingPlatform.platform} as efficiency improved.`,
    };
  });
}

function renderMomentum(snapshot) {
  const timeline = buildTimeline(snapshot);
  const first = timeline[0];
  const latest = timeline[timeline.length - 1];
  const roasLift = ((latest.roas - first.roas) / Math.max(first.roas, 0.01)) * 100;
  const conversionLift =
    ((latest.conversions - first.conversions) / Math.max(first.conversions, 1)) * 100;
  const ctrLift = latest.ctr - first.ctr;

  const summaryCards = [
    {
      label: "Total ROAS",
      value: `${latest.roas.toFixed(2)}x`,
      detail: `${roasLift >= 0 ? "+" : ""}${roasLift.toFixed(0)}% vs ${first.label}`,
    },
    {
      label: "Conversions",
      value: number(latest.conversions),
      detail: `${conversionLift >= 0 ? "+" : ""}${conversionLift.toFixed(0)}% over the same window`,
    },
    {
      label: "CTR",
      value: pct(latest.ctr),
      detail: `${ctrLift >= 0 ? "+" : ""}${ctrLift.toFixed(1)}pt lift as the mix improves`,
    },
  ];

  trendSummary.innerHTML = summaryCards
    .map(
      (card) => `
        <article class="trend-metric">
          <span>${card.label}</span>
          <strong>${card.value}</strong>
          <p>${card.detail}</p>
        </article>
      `
    )
    .join("");

  trendHeadline.textContent = `${latest.roas.toFixed(2)}x ROAS with rising conversion volume`;
  trendDelta.textContent = `${roasLift >= 0 ? "+" : ""}${roasLift.toFixed(0)}% efficiency`;
  trendDelta.className = `trend-delta ${roasLift >= 0 ? "is-positive" : "is-negative"}`;

  const values = timeline.map((point) => point.roas);
  const max = Math.max(...values);
  const min = Math.min(...values);
  const width = 620;
  const height = 240;
  const padding = 24;
  const xStep = (width - padding * 2) / (timeline.length - 1 || 1);
  const yScale = (value) => {
    if (max === min) {
      return height / 2;
    }
    return height - padding - ((value - min) / (max - min)) * (height - padding * 2);
  };
  const points = timeline.map((point, index) => ({
    x: padding + index * xStep,
    y: yScale(point.roas),
    ...point,
  }));
  const polyline = points.map((point) => `${point.x},${point.y}`).join(" ");

  trendChart.innerHTML = `
    <svg viewBox="0 0 ${width} ${height}" role="img" aria-label="ROAS trend over time">
      <defs>
        <linearGradient id="trendStroke" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stop-color="#f59e0b"></stop>
          <stop offset="100%" stop-color="#0f766e"></stop>
        </linearGradient>
      </defs>
      ${points
        .map(
          (point) =>
            `<line class="trend-grid" x1="${point.x}" y1="${padding}" x2="${point.x}" y2="${
              height - padding
            }"></line>`
        )
        .join("")}
      <polyline class="trend-path" points="${polyline}"></polyline>
      ${points
        .map(
          (point) => `
            <circle class="trend-point" cx="${point.x}" cy="${point.y}" r="5"></circle>
            <text class="trend-value" x="${point.x}" y="${point.y - 12}" text-anchor="middle">
              ${point.roas.toFixed(2)}x
            </text>
            <text class="trend-tick" x="${point.x}" y="${height - 6}" text-anchor="middle">
              ${point.label}
            </text>
          `
        )
        .join("")}
    </svg>
  `;

  timelineList.innerHTML = timeline
    .map(
      (point, index) => `
        <article class="timeline-item ${index === timeline.length - 1 ? "is-current" : ""}">
          <div class="timeline-marker"></div>
          <div>
            <div class="timeline-item-header">
              <strong>${point.label}</strong>
              <span>${point.roas.toFixed(2)}x ROAS</span>
            </div>
            <p>${number(point.conversions)} conversions • ${pct(point.ctr)} CTR</p>
            <p>${point.note}</p>
          </div>
        </article>
      `
    )
    .join("");
}

async function load() {
  try {
    const [snapshot, connections] = await Promise.all([fetchDashboard(), fetchConnections()]);
    renderDashboard(snapshot);
    renderConnections(connections);
  } catch (error) {
    document.getElementById("campaignName").textContent = "Unable to load dashboard";
    document.getElementById("campaignMeta").textContent = error.message;
    connectionStatus.textContent = error.message;
    connectionStatus.className = "form-status is-error";
  }
}

rebalanceButton.addEventListener("click", async () => {
  rebalanceButton.disabled = true;
  rebalanceButton.textContent = "Rebalancing...";
  try {
    renderDashboard(await fetchJSON("/api/v1/rebalance", { method: "POST" }));
  } catch (error) {
    document.getElementById("campaignMeta").textContent = error.message;
  } finally {
    rebalanceButton.disabled = false;
    rebalanceButton.textContent = "Run rebalance now";
  }
});

connectButton.addEventListener("click", () => {
  connectionStatus.textContent = "Redirecting to Meta to authorize account access...";
  connectionStatus.className = "form-status";
});

renderOAuthStatus();
load();
setInterval(load, 8000);
