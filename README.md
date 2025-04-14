# 🔍 GPU + System Monitor Dashboard 🌡️⚡🧠

Welcome to **GPU + System Monitor** — a lightweight, self-hosted dashboard for monitoring your system’s **GPU performance**, **processes**, and **system health** in real-time! 🚀

Whether you're running multiple GPUs in a render farm, deep learning rig, or just want insight into your workstation's status — this tool gives you what you need 📊.

---

## ✨ Features

- 🎮 **GPU Metrics** via `nvidia-smi`
  - Fan speed, temperature, power draw 🔥⚡
  - Memory usage & utilization 💾
  - Active processes 🧠
- 💻 **System Stats**
  - 🧠 Memory usage (used / total / %)  
  - 💽 Disk usage (used / total / %)  
  - 📈 Load average (1, 5, 15 min)
- 🌐 **Web Dashboard**
  - Auto-refreshing every 5s 🔁
  - Color-coded health indicators 🎨
- 📝 JSON API + SQLite backend
- 🐧 Linux-compatible & easy to extend

---

## 🛠️ Requirements

- Linux 🐧
- NVIDIA GPU(s) with drivers installed
- `nvidia-smi` available
- 🐍 Bash (for the client script)
- 🧰 Go (to build the backend server)
- `jq`, `curl`, `awk`, `df`, `free` commands

---

## 🚀 Getting Started

### 1️⃣ Clone the Repo

```bash
git clone 
cd gpu-monitor


