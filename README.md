# 🔥 HARDCOREAI - The AI-Powered Hardware Diagnostic IDE

> **"Cursor for Embedded Engineers"** - A production-grade VS Code extension that automatically detects hardware faults, decodes stack traces, and provides AI-powered fixes for STM32 and other Cortex-M systems.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)
![Extension](https://img.shields.io/badge/VS_Code-Extension-brightgreen)

---

## ✨ Key Features

- 🧠 **Zero-Touch Setup** - Self-contained Python backend with automated virtual environment management.
- 📡 **Live Hardware Monitor** - Real-time connection to ST-Link/J-Link via OpenOCD/GDB.
- 💥 **Automated Fault Decoding** - Instant capture and analysis of HardFault, MemManage, and UsageFault exceptions.
- 🤖 **Expert AI Reasoning** - Integrated Gemini AI that understands register states and provides source-level fix suggestions.
- 📊 **Dynamic Dashboard** - Beautiful, integrated VS Code panel for visualizing fault state and memory.
- 🔍 **Tool Discovery** - Smart auto-discovery of OpenOCD and GDB binaries (STM32CubeIDE compatible).

---

## 🚀 Installation

HARDCOREAI is distributed as a single `.vsix` package for ease of use.

1. **Download** the latest `hardcore-ai-0.1.0.vsix` from the [Releases](https://github.com/Hardcoreaiin/hardcore-ai/releases) page.
2. **Open VS Code**.
3. Go to the **Extensions** view (`Ctrl+Shift+X`).
4. Click the **`...`** (top right) → **"Install from VSIX..."**.
5. Select the downloaded file.

---

## ⚡ Quick Start

### 1. Wait for Auto-Setup
Upon first launch, HARDCOREAI will build its internal "Diagnostic Brain." Watch the status bar for:
`"Setting up HARDCOREAI Intelligence Layer..."`

### 2. Configure your API Key
To enable AI reasoning:
1. Go to **Settings** (`Ctrl + ,`).
2. Search for **"Hardcoreai"**.
3. Enter your **Gemini API Key**.

### 3. Start Monitoring
1. Connect your debugger (ST-Link/J-Link) to your target board.
2. Press `Ctrl+Shift+P` → **`HARDCOREAI: Start Live Hardware Monitor`**.
3. Enter your interface (e.g., `interface/stlink.cfg`) and target (e.g., `target/stm32f1x.cfg`).

---

## 🛠️ Requirements

- **Python 3.10+** installed on your system.
- **OpenOCD** (The extension will try to auto-find it in your STM32CubeIDE or Downloads folders).
- **ARM GNU Toolchain** (for GDB diagnostics).

---

## 🤝 Contributing

We welcome contributions to the hardware reasoning engine and UI components. Please submit PRs or open issues on our [GitHub Repo](https://github.com/Hardcoreaiin/hardcore-ai).

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

**Made with ❤️ by the HardcoreAI Team**
