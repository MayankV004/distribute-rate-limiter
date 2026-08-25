# Distributed Rate Limiter - Frontend Dashboard & Topology Simulator

An interactive, real-time visualizer and simulation dashboard built with **Next.js 15 (App Router)**, **TypeScript**, and **Tailwind CSS**. It provides live node topology tracking, customizable traffic generation, and real-time metric visualization for the Go Distributed Rate Limiter API Gateway.

---

## 🔗 Quick Links

- 🏠 **[Root Monorepo README](file:///home/streamliner/rate-liimter/README.md)**
- ⚙️ **[Backend API Gateway README](file:///home/streamliner/rate-liimter/backend/README.md)**

---

## 🎨 Overview & Key Features

- **Interactive Topology Canvas**: Real-time canvas visualizing client requests flowing through the Nginx Load Balancer, distributed Go Gateway replicas, Redis rate limiter store, and upstream services.
- **Live Traffic Generator**: Configurable slider controls to simulate variable traffic loads (RPS), trigger request bursts, and select API key tiers (`Free` vs `Pro`).
- **Real-Time System Metrics**: High-frequency metric cards displaying throughput (RPS), total allowed vs rate-limited (429) requests, p95 latencies, and circuit breaker status.
- **Node Detail Inspector**: Clickable system nodes opening detailed modals for node status, throughput history, and simulated logs.
- **API Gateway Proxy Bridge**: Next.js Serverless API route (`/api/proxy-gateway`) bridging live traffic from the frontend to the Nginx gateway endpoint (`http://localhost/api/v1/search`).

---

## 🏗️ Directory Structure

```text
frontend/
├── src/
│   ├── app/
│   │   ├── layout.tsx              # Root layout & theme providers
│   │   ├── page.tsx                # Main Visualizer Dashboard page
│   │   ├── docs/page.tsx           # Architecture documentation view
│   │   └── api/
│   │       └── proxy-gateway/      # Proxy route executing live requests against Go Gateway
│   ├── components/
│   │   ├── TopologyCanvas.tsx      # Canvas renderer for network node graph & animations
│   │   ├── TrafficController.tsx   # Controls for RPS sliders, key tiers, and burst simulation
│   │   ├── StatsGrid.tsx           # Live metric cards (RPS, 200s, 429s, Latency)
│   │   ├── ScalingSection.tsx      # Gateway replica scaling and load balance distribution
│   │   ├── NodeDetailModal.tsx     # Inspection modal for individual system nodes
│   │   ├── Navbar.tsx              # Application header & navigation links
│   │   └── DocsSection.tsx         # Embedded documentation viewer
│   └── context/
│       └── ThemeContext.tsx        # System-wide dark/light theme management
├── public/                         # Static assets & icons
└── package.json                    # Project dependencies & scripts
```

---

## 🚀 Getting Started

### 1. Install Dependencies

Ensure you have Node.js 18+ installed:

```bash
npm install
```

### 2. Run Development Server

Launch the Next.js development server:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

> **Note:** Ensure the backend infrastructure (`docker compose -f ../backend/deployments/docker-compose.yaml up -d`) is running to enable live API proxying.

### 3. Build for Production

```bash
npm run build
npm run start
```

---

## 🛠️ Tech Stack

- **Framework**: Next.js 15 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS & CSS Modules
- **Icons**: Lucide React
- **Graphics**: HTML5 Canvas API

---

## 📜 License

[MIT](LICENSE)
