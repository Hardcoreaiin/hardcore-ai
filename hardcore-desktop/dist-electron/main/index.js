"use strict";
const electron = require("electron");
const path = require("path");
const child_process = require("child_process");
let mainWindow = null;
let backendProcess = null;
const isDev = !electron.app.isPackaged;
function createWindow() {
  mainWindow = new electron.BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1200,
    minHeight: 700,
    frame: true,
    backgroundColor: "#0a0a0f",
    webPreferences: {
      preload: path.join(__dirname, "../preload/index.js"),
      contextIsolation: true,
      nodeIntegration: false
    }
  });
  if (isDev) {
    mainWindow.loadURL("http://localhost:5173");
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, "../../dist-react/index.html"));
  }
  mainWindow.on("closed", () => {
    mainWindow = null;
  });
}
const PORT = 8003;
function clearPort(port) {
  console.log(`[Electron] Checking port ${port}...`);
  try {
    if (process.platform === "win32") {
      const cmd = `for /f "tokens=5" %a in ('netstat -aon ^| findstr :${port} ^| findstr LISTENING') do taskkill /F /PID %a`;
      const { execSync } = require("child_process");
      execSync(cmd);
      console.log(`[Electron] Forcefully cleared port ${port}.`);
    }
  } catch (e) {
    console.log(`[Electron] Port ${port} is free or couldn't be cleared (may not be listening).`);
  }
}
function startBackend() {
  var _a, _b;
  const pythonPath = isDev ? "python" : path.join(process.resourcesPath, "python", "python.exe");
  const backendDir = isDev ? path.join(__dirname, "../../../Hardcore.ai/orchestrator") : path.join(process.resourcesPath, "backend");
  console.log("[Electron] Starting backend...");
  clearPort(PORT);
  try {
    backendProcess = child_process.spawn(pythonPath, [
      "-m",
      "uvicorn",
      "server:app",
      "--host",
      "127.0.0.1",
      "--port",
      PORT.toString(),
      "--log-level",
      "info"
    ], {
      cwd: backendDir,
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env }
    });
    (_a = backendProcess.stdout) == null ? void 0 : _a.on("data", (data) => {
      console.log(`[Backend] ${data}`);
    });
    (_b = backendProcess.stderr) == null ? void 0 : _b.on("data", (data) => {
      console.error(`[Backend Error] ${data}`);
    });
    backendProcess.on("close", (code) => {
      console.log(`[Backend] Exited with code ${code}`);
      backendProcess = null;
    });
    backendProcess.on("error", (error) => {
      console.error("[Electron] Failed to start backend:", error);
    });
  } catch (error) {
    console.error("[Electron] Failed to start backend:", error);
  }
}
function stopBackend() {
  if (backendProcess && backendProcess.pid) {
    console.log("[Electron] Stopping backend tree...");
    try {
      if (process.platform === "win32") {
        const { execSync } = require("child_process");
        execSync(`taskkill /F /T /PID ${backendProcess.pid}`);
      } else {
        backendProcess.kill();
      }
    } catch (e) {
      console.error("[Electron] Error during stopBackend:", e);
    }
    backendProcess = null;
  }
}
electron.app.whenReady().then(() => {
  startBackend();
  createWindow();
  electron.app.on("activate", () => {
    if (electron.BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});
electron.app.on("window-all-closed", () => {
  stopBackend();
  if (process.platform !== "darwin") {
    electron.app.quit();
  }
});
electron.app.on("before-quit", () => {
  stopBackend();
});
electron.ipcMain.handle("shell:openExternal", async (_, url) => {
  await electron.shell.openExternal(url);
});
electron.ipcMain.handle("app:getInfo", () => {
  return {
    version: electron.app.getVersion(),
    platform: process.platform,
    isDev
  };
});
