"use strict";
const electron = require("electron");
electron.contextBridge.exposeInMainWorld("electronAPI", {
  // Shell operations
  shell: {
    openExternal: (url) => electron.ipcRenderer.invoke("shell:openExternal", url)
  },
  // App info
  app: {
    getInfo: () => electron.ipcRenderer.invoke("app:getInfo")
  },
  // AI communication (uses HTTP to backend)
  ai: {
    sendMessage: async (history, board) => {
      var _a;
      const response = await fetch("http://localhost:8003/execute", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          prompt: ((_a = history[history.length - 1]) == null ? void 0 : _a.content) || "",
          board_type: board,
          history
        })
      });
      return response.json();
    }
  },
  // PlatformIO operations
  platformio: {
    build: async (projectPath, board) => {
      const response = await fetch("http://localhost:8003/build", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project_path: projectPath, board })
      });
      return response.json();
    },
    flash: async (projectPath, port, board) => {
      const response = await fetch("http://localhost:8003/flash", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project_path: projectPath, port, board })
      });
      return response.json();
    },
    onBuildOutput: (callback) => {
      return () => {
      };
    }
  },
  // Device detection
  devices: {
    detect: async () => {
      const response = await fetch("http://localhost:8003/detect");
      return response.json();
    }
  },
  // Driver installation
  drivers: {
    install: async () => {
      const response = await fetch("http://localhost:8003/install-drivers", {
        method: "POST"
      });
      return response.json();
    }
  }
});
