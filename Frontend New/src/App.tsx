import React, { useState, useEffect, useRef } from "react";
import Editor from "@monaco-editor/react";
import {
  Play,
  Zap,
  Bug,
  FolderOpen,
  FileCode,
  File,
  Send,
  AlertTriangle,
  Sparkles,
  ArrowRight,
  Search,
  GitBranch,
  Blocks,
  Folder,
  Settings,
  HelpCircle,
  X,
  ChevronRight,
  ChevronDown,
  RotateCcw,
  Maximize2,
  MoreHorizontal,
  Plus,
  Moon,
  Minus,
  Square
} from "lucide-react";
import { useWorkspaceStore, FileItem } from "./store/workspaceStore";
import "./App.css";

function App() {
  const {
    activeFile,
    fileContents,
    fileTree,
    isCompiling,
    isFlashing,
    isDebugging,
    currentLine,
    registers,
    crashed,
    crashReason,
    serialLogs,
    serialConnected,
    baudRate,
    plotData,
    aiMessages,
    aiWaiting,
    activeBottomTab,
    setActiveFile,
    updateFileContent,
    setCompiling,
    setFlashing,
    addBuildLog,
    clearBuildLogs,
    startDebugging,
    stopDebugging,
    toggleSerialConnection,
    addSerialLog,
    addPlotPoint,
    sendAiMessage,
    setBottomTab,
    triggerCrash,
    resolveCrash,
    stepOver,
    continueExecution,
    
    // Welcome & Custom UI State
    showWelcomeScreen,
    activeSidebarTab,
    selectedBoard,
    selectedProbe,
    toolchainPath,
    setShowWelcomeScreen,
    setActiveSidebarTab,
    setSelectedBoard,
    setSelectedProbe,
    setToolchainPath
  } = useWorkspaceStore();

  const [aiInput, setAiInput] = useState("");
  const [serialInput, setSerialInput] = useState("");
  const [selectedPeripheral, setSelectedPeripheral] = useState("Core Registers");
  const [aiOpen, setAiOpen] = useState(true); // Default to open matching screenshot

  // Resizable Panels States
  const [sidebarWidth, setSidebarWidth] = useState(260);
  const [rightSidebarWidth, setRightSidebarWidth] = useState(380);
  const [bottomDrawerHeight, setBottomDrawerHeight] = useState(220);
  
  const [isDraggingLeft, setIsDraggingLeft] = useState(false);
  const [isDraggingRight, setIsDraggingRight] = useState(false);
  const [isDraggingBottom, setIsDraggingBottom] = useState(false);

  // Dragging Mouse Event Handlers
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (isDraggingLeft) {
        // limit: min 180px, max 450px
        const newWidth = Math.max(180, Math.min(450, e.clientX - 52)); // 52px is custom activity bar width
        setSidebarWidth(newWidth);
      }
      if (isDraggingRight) {
        // limit: min 280px, max 600px
        const newWidth = Math.max(280, Math.min(600, window.innerWidth - e.clientX));
        setRightSidebarWidth(newWidth);
      }
      if (isDraggingBottom) {
        // limit: min 120px, max 500px
        const newHeight = Math.max(120, Math.min(500, window.innerHeight - e.clientY));
        setBottomDrawerHeight(newHeight);
      }
    };

    const handleMouseUp = () => {
      setIsDraggingLeft(false);
      setIsDraggingRight(false);
      setIsDraggingBottom(false);
    };

    if (isDraggingLeft || isDraggingRight || isDraggingBottom) {
      window.addEventListener("mousemove", handleMouseMove);
      window.addEventListener("mouseup", handleMouseUp);
    }

    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isDraggingLeft, isDraggingRight, isDraggingBottom]);
  
  const terminalEndRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  // Auto-scroll terminal output
  useEffect(() => {
    if (terminalEndRef.current) {
      terminalEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [serialLogs]);

  // Periodic Telemetry Simulator
  useEffect(() => {
    let interval: NodeJS.Timeout;
    if (serialConnected && !crashed) {
      interval = setInterval(() => {
        const time = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
        const prevTemp = plotData.length > 0 ? plotData[plotData.length - 1].temp : 24.5;
        const delta = (Math.random() - 0.45) * 1.2;
        const temp = Math.max(10, Math.min(50, prevTemp + delta));
        const voltage = 3.3 + (Math.random() - 0.5) * 0.05;
        const current = 40.0 + (Math.random() - 0.5) * 4.0;
        
        addPlotPoint({ time, temp, voltage, current });
        addSerialLog(`TEMP_C:${temp.toFixed(2)} | VDD:${voltage.toFixed(2)}V | IDD:${current.toFixed(1)}mA`);
        
        if (temp > 38.0 && !crashed) {
          addSerialLog("WARNING: MCU Core Temperature exceeding safety threshold (>38.0 C)!");
        }
      }, 3000);
    }
    return () => clearInterval(interval);
  }, [serialConnected, plotData, crashed]);

  // Drawing the real-time telemetry line plot on Canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const width = canvas.clientWidth;
    const height = canvas.clientHeight;
    canvas.width = width;
    canvas.height = height;

    ctx.clearRect(0, 0, width, height);

    ctx.strokeStyle = "#1C1C24";
    ctx.lineWidth = 1;
    for (let i = 40; i < width; i += 60) {
      ctx.beginPath();
      ctx.moveTo(i, 0);
      ctx.lineTo(i, height - 20);
      ctx.stroke();
    }
    for (let i = 20; i < height - 20; i += 30) {
      ctx.beginPath();
      ctx.moveTo(40, i);
      ctx.lineTo(width, i);
      ctx.stroke();
    }

    if (plotData.length < 2) {
      ctx.fillStyle = "#94A3B8";
      ctx.font = "12px Inter";
      ctx.fillText("Waiting for serial stream data...", width / 2 - 80, height / 2);
      return;
    }

    const paddingLeft = 40;
    const paddingBottom = 20;
    const graphWidth = width - paddingLeft - 20;
    const graphHeight = height - paddingBottom - 10;

    const temps = plotData.map((d) => d.temp);
    const minTemp = Math.min(...temps) - 1;
    const maxTemp = Math.max(...temps) + 1;
    const tempRange = maxTemp - minTemp || 1;

    ctx.strokeStyle = "#8B5CF6";
    ctx.lineWidth = 2;
    ctx.beginPath();

    plotData.forEach((pt, index) => {
      const x = paddingLeft + (index / (plotData.length - 1)) * graphWidth;
      const y = height - paddingBottom - ((pt.temp - minTemp) / tempRange) * graphHeight;
      if (index === 0) {
        ctx.moveTo(x, y);
      } else {
        ctx.lineTo(x, y);
      }
    });
    ctx.stroke();

    ctx.shadowBlur = 4;
    ctx.shadowColor = "#8B5CF6";
    ctx.strokeStyle = "rgba(139, 92, 246, 0.3)";
    ctx.stroke();
    ctx.shadowBlur = 0;

    ctx.strokeStyle = "#475569";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(paddingLeft, 5);
    ctx.lineTo(paddingLeft, height - paddingBottom);
    ctx.lineTo(width - 10, height - paddingBottom);
    ctx.stroke();

    ctx.fillStyle = "#94A3B8";
    ctx.font = "9px JetBrains Mono";
    ctx.fillText(`${maxTemp.toFixed(1)}°C`, 5, 12);
    ctx.fillText(`${minTemp.toFixed(1)}°C`, 5, height - paddingBottom - 4);
    
    plotData.forEach((pt, index) => {
      if (index % Math.ceil(plotData.length / 5) === 0) {
        const x = paddingLeft + (index / (plotData.length - 1)) * graphWidth;
        ctx.fillText(pt.time.split(":")[2] ? pt.time.substring(3) : pt.time, x - 10, height - 6);
      }
    });

  }, [plotData, activeBottomTab]);

  const handleBuild = () => {
    if (isCompiling) return;
    setCompiling(true);
    clearBuildLogs();
    addBuildLog("HARDCOREAI Build Engine v1.0.0");
    addBuildLog("Scanning active target configurations...");
    addBuildLog(`Found toolchain: ${toolchainPath}`);
    addBuildLog(`Target architecture: ${selectedBoard === "STM32F401" ? "Cortex-M4 (STM32F401RET6)" : selectedBoard === "ESP32-S3" ? "Xtensa LX7 (ESP32-S3)" : "Cortex-M0+ (RP2040)"}`);
    
    setTimeout(() => {
      addBuildLog("Compiling src/main.c...");
      addBuildLog("Compiling src/stm32f4xx_it.c...");
    }, 400);

    setTimeout(() => {
      addBuildLog("Linking build/hardcoreai_app.elf...");
      addBuildLog("Generating map file: build/hardcoreai_app.map");
      addBuildLog("──────────────────────────────────────────");
      addBuildLog("Memory utilization statistics (static ELF analysis):");
      addBuildLog("  FLASH:  26.4 KB / 256.0 KB (10.3%)");
      addBuildLog("  SRAM:   12.1 KB /  64.0 KB (18.9%)");
      addBuildLog("──────────────────────────────────────────");
      addBuildLog("Build Successful. Object binary generated: build/hardcoreai_app.bin");
      setCompiling(false);
    }, 1500);
  };

  const handleFlash = () => {
    if (isFlashing) return;
    setFlashing(true);
    addBuildLog("Launching flashing engine...");
    addBuildLog(`Flashing target via probe: ${selectedProbe}`);
    
    setTimeout(() => {
      addBuildLog("Connection verified. Halting target core...");
      addBuildLog("Erasing sector 0 (16KB)... OK");
      addBuildLog("Erasing sector 1 (16KB)... OK");
      addBuildLog("Writing binary image to flash block 0x08000000...");
    }, 400);

    setTimeout(() => {
      addBuildLog("Verifying integrity checksum... OK");
      addBuildLog("Resetting target CPU core. Start execution...");
      addBuildLog("Flash Completed Successfully.");
      setFlashing(false);
      
      addSerialLog("[SYSTEM] Board reset. Flashed firmware execution initialized.");
    }, 1200);
  };

  const handleDebugToggle = () => {
    if (isDebugging) {
      stopDebugging();
      addBuildLog("Debugger disconnected.");
    } else {
      addBuildLog("Launching GDB debug server...");
      addBuildLog(`Probe: ${selectedProbe} connected to target: ${selectedBoard}`);
      setTimeout(() => {
        startDebugging();
        addBuildLog("Debugger successfully attached. Target halted at main() -> main.c:22");
      }, 800);
    }
  };

  const handleAiSend = (e: React.FormEvent) => {
    e.preventDefault();
    if (!aiInput.trim()) return;
    sendAiMessage(aiInput);
    setAiInput("");
  };

  const renderFileNode = (item: FileItem) => {
    const isFolder = item.isFolder;
    const isActive = activeFile === item.path;

    if (isFolder) {
      return (
        <div key={item.path} style={{ marginBottom: "2px" }}>
          <div className="file-item folder">
            <FolderOpen size={14} style={{ color: "var(--accent-violet)" }} />
            <span>{item.name}</span>
          </div>
          <div className="folder-contents">
            {item.children?.map(child => renderFileNode(child))}
          </div>
        </div>
      );
    } else {
      const isC = item.name.endsWith(".c") || item.name.endsWith(".h");
      return (
        <div
          key={item.path}
          className={`file-item ${isActive ? "active" : ""}`}
          onClick={() => setActiveFile(item.path)}
        >
          {isC ? <FileCode size={14} style={{ color: "var(--accent-violet-hover)" }} /> : <File size={14} />}
          <span>{item.name}</span>
        </div>
      );
    }
  };

  const getCurrentContent = () => {
    if (activeFile && fileContents[activeFile]) {
      return fileContents[activeFile];
    }
    return "";
  };

  const handleEditorChange = (value: string | undefined) => {
    if (activeFile && value !== undefined) {
      updateFileContent(activeFile, value);
    }
  };

  const renderSidebarContent = () => {
    switch (activeSidebarTab) {
      case "explorer":
        return (
          <>
            <div className="panel-header">
              <div className="panel-title">
                <span>PROJECT EXPLORER</span>
              </div>
              <div className="pane-header-actions" style={{ display: "flex", gap: "6px" }}>
                <Plus size={13} style={{ cursor: "pointer", color: "var(--text-muted)" }} />
                <FolderOpen size={12} style={{ cursor: "pointer", color: "var(--text-muted)" }} />
                <RotateCcw size={12} style={{ cursor: "pointer", color: "var(--text-muted)" }} />
                <Minus size={12} style={{ cursor: "pointer", color: "var(--text-muted)" }} />
              </div>
            </div>
            
            <div className="panel-body flex-container-explorer" style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              {/* Workspace Folder Tree */}
              <div className="explorer-section">
                <div className="file-list">
                  <div style={{ marginBottom: "2px" }}>
                    <div className="file-item folder">
                      <Folder size={14} style={{ color: "var(--accent-violet)" }} />
                      <span>Workspace</span>
                    </div>
                    <div className="folder-contents">
                      <div style={{ marginBottom: "2px" }}>
                        <div className="file-item folder">
                          <Folder size={14} style={{ color: "var(--accent-violet)" }} />
                          <span>blinky-stm32f4</span>
                        </div>
                        <div className="folder-contents">
                          {fileTree.map(item => renderFileNode(item))}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* QUICK ACCESS Section */}
              <div className="explorer-sub-section">
                <div className="explorer-sub-header">QUICK ACCESS</div>
                <div className="quick-access-item">
                  <span>Open Folder...</span>
                  <span className="shortcut-tag">Ctrl+K Ctrl+O</span>
                </div>
                <div className="quick-access-item">
                  <span>New File</span>
                  <span className="shortcut-tag">Ctrl+N</span>
                </div>
                <div className="quick-access-item">
                  <span>New Embedded Project</span>
                </div>
                <div className="quick-access-item">
                  <span>Recent Projects</span>
                </div>
              </div>

              {/* WORKSPACE Section */}
              <div className="explorer-sub-section">
                <div className="explorer-sub-header">WORKSPACE</div>
                <div className="workspace-item-row">
                  <Settings size={12} style={{ color: "var(--text-muted)" }} />
                  <span>blinky-stm32f4</span>
                </div>
              </div>
            </div>
          </>
        );
      case "search":
        return (
          <>
            <div className="panel-header">
              <div className="panel-title">
                <span>Search Workspace</span>
              </div>
            </div>
            <div className="panel-body">
              <div className="sidebar-search-panel">
                <input type="text" placeholder="Search string..." />
                <input type="text" placeholder="Files to include (e.g. *.c)" />
                <div style={{ fontSize: "0.75rem", color: "var(--text-dark)", marginTop: "10px" }}>
                  No active search results. Press Enter to search.
                </div>
              </div>
            </div>
          </>
        );
      case "git":
        return (
          <>
            <div className="panel-header">
              <div className="panel-title">
                <span>Source Control</span>
              </div>
            </div>
            <div className="panel-body">
              <div className="sidebar-git-panel">
                <input type="text" placeholder="Commit message (Ctrl+Enter)..." />
                <button className="git-commit-btn">Commit Changes</button>
                <div style={{ fontSize: "0.75rem", color: "var(--text-muted)", marginTop: "12px", borderTop: "1px solid var(--border-color)", paddingTop: "8px" }}>
                  <strong style={{ display: "block", marginBottom: "4px" }}>Staged Changes (2)</strong>
                  <div style={{ padding: "2px 0", color: "var(--accent-success)", fontFamily: "var(--font-mono)", fontSize: "0.7rem" }}>M  src/main.c</div>
                  <div style={{ padding: "2px 0", color: "var(--text-muted)", fontFamily: "var(--font-mono)", fontSize: "0.7rem" }}>M  CMakeLists.txt</div>
                </div>
              </div>
            </div>
          </>
        );
      case "debug":
        return (
          <>
            <div className="panel-header">
              <div className="panel-title">
                <span>Run & Debug</span>
              </div>
            </div>
            <div className="panel-body">
              <div className="sidebar-debug-panel">
                <div style={{ fontSize: "0.75rem", color: "var(--text-muted)", marginBottom: "12px" }}>
                  <strong style={{ display: "block", marginBottom: "4px" }}>Call Stack</strong>
                  <div style={{ fontFamily: "var(--font-mono)", fontSize: "0.7rem", color: "var(--text-muted)" }}>
                    main() at main.c:22
                  </div>
                  <div style={{ fontFamily: "var(--font-mono)", fontSize: "0.7rem", color: "var(--text-dark)" }}>
                    Reset_Handler() at startup_stm32.s:55
                  </div>
                </div>
                <div style={{ fontSize: "0.75rem", color: "var(--text-muted)" }}>
                  <strong style={{ display: "block", marginBottom: "4px" }}>Active Breakpoints</strong>
                  <div style={{ padding: "2px 0", display: "flex", alignItems: "center", gap: "6px" }}>
                    <span style={{ width: "6px", height: "6px", borderRadius: "50%", backgroundColor: "var(--accent-error)" }}></span>
                    <span>main.c: Line 24</span>
                  </div>
                  <div style={{ padding: "2px 0", display: "flex", alignItems: "center", gap: "6px" }}>
                    <span style={{ width: "6px", height: "6px", borderRadius: "50%", backgroundColor: "var(--accent-error)" }}></span>
                    <span>stm32f4xx_it.c: Line 183</span>
                  </div>
                </div>
              </div>
            </div>
          </>
        );
      case "extensions":
        return (
          <>
            <div className="panel-header">
              <div className="panel-title">
                <span>Marketplace Extensions</span>
              </div>
            </div>
            <div className="panel-body">
              <div className="sidebar-extensions-panel">
                <div style={{ border: "1px solid var(--border-color)", padding: "8px", borderRadius: "var(--radius-sm)", display: "flex", flexDirection: "column", gap: "2px", backgroundColor: "var(--bg-tertiary)" }}>
                  <div style={{ fontSize: "0.75rem", fontWeight: 600 }}>C/C++ Tools Pack</div>
                  <div style={{ fontSize: "0.65rem", color: "var(--text-muted)" }}>Rich syntax, toolchain utilities</div>
                  <span style={{ fontSize: "0.65rem", color: "var(--accent-violet)", alignSelf: "flex-end" }}>Installed</span>
                </div>
                <div style={{ border: "1px solid var(--border-color)", padding: "8px", borderRadius: "var(--radius-sm)", display: "flex", flexDirection: "column", gap: "2px", backgroundColor: "var(--bg-tertiary)" }}>
                  <div style={{ fontSize: "0.75rem", fontWeight: 600 }}>ARM SVD Visualizer</div>
                  <div style={{ fontSize: "0.65rem", color: "var(--text-muted)" }}>Visual register mappings for Cortex</div>
                  <span style={{ fontSize: "0.65rem", color: "var(--accent-violet)", alignSelf: "flex-end" }}>Installed</span>
                </div>
              </div>
            </div>
          </>
        );
      case "boards":
        return (
          <>
            <div className="panel-header">
              <div className="panel-title">
                <span>Embedded Target Configuration</span>
              </div>
            </div>
            <div className="panel-body">
              <div className="boards-config-panel">
                <div className="config-group">
                  <label>MCU Board Target</label>
                  <select className="config-select" value={selectedBoard} onChange={(e) => setSelectedBoard(e.target.value as any)}>
                    <option value="STM32F401">STM32F401 (Cortex-M4)</option>
                    <option value="ESP32-S3">ESP32-S3 (Xtensa LX7)</option>
                    <option value="RP2040">RP2040 (Cortex-M0+)</option>
                  </select>
                </div>
                <div className="config-group">
                  <label>Debugger Probe</label>
                  <select className="config-select" value={selectedProbe} onChange={(e) => setSelectedProbe(e.target.value as any)}>
                    <option value="ST-Link V2">ST-Link V2 (SWD)</option>
                    <option value="J-Link">J-Link (SWD/JTAG)</option>
                    <option value="CMSIS-DAP">CMSIS-DAP (SWD)</option>
                  </select>
                </div>
                <div className="config-group">
                  <label>Toolchain Path</label>
                  <div className="path-input-wrapper">
                    <input type="text" className="config-input" value={toolchainPath} onChange={(e) => setToolchainPath(e.target.value)} />
                    <button className="browse-btn" onClick={() => setToolchainPath("/usr/bin/arm-none-eabi-gcc")}>Reset</button>
                  </div>
                </div>
              </div>
            </div>
          </>
        );
      default:
        return null;
    }
  };

  if (showWelcomeScreen) {
    return (
      <div className="helix-app">
        <header className="helix-header">
          <div className="logo-section">
            <div className="logo-text">HARDCORE<span>AI</span></div>
          </div>
          <div className="connection-status">
            <button className="status-pill" onClick={() => setShowWelcomeScreen(false)} style={{ cursor: "pointer", border: "1px solid var(--accent-violet)" }}>
              <span style={{ color: "var(--accent-violet-hover)" }}>Skip to Workspace</span>
              <ArrowRight size={12} style={{ color: "var(--accent-violet-hover)" }} />
            </button>
          </div>
        </header>
        
        <div className="welcome-screen">
          <div className="welcome-container">
            <div className="welcome-header">
              <h1 className="welcome-title">HARDCORE<span>AI</span></h1>
              <p className="welcome-subtitle">
                A premium, modern embedded developer workspace. Optimize your compilation, flashing, and debug loops directly on target microcontrollers with zero unnecessary visual noise.
              </p>
            </div>
            
            <div className="welcome-grid">
              <div className="welcome-column">
                <h3 className="welcome-section-title">Start</h3>
                <div className="welcome-action-list">
                  <button className="welcome-action-btn" onClick={() => {
                    setActiveSidebarTab("explorer");
                    setShowWelcomeScreen(false);
                  }}>
                    <FolderOpen size={16} className="welcome-action-icon" />
                    <span>Open Project Folder...</span>
                  </button>
                  <button className="welcome-action-btn" onClick={() => {
                    setActiveSidebarTab("boards");
                    setShowWelcomeScreen(false);
                  }}>
                    <Settings size={16} className="welcome-action-icon" />
                    <span>Configure Target Hardware...</span>
                  </button>
                  <button className="welcome-action-btn" onClick={() => {
                    setActiveSidebarTab("explorer");
                    setShowWelcomeScreen(false);
                    addBuildLog("Created new embedded project template from STM32F4xx HAL repository.");
                  }}>
                    <FileCode size={16} className="welcome-action-icon" />
                    <span>Create Embedded Project Template</span>
                  </button>
                </div>
              </div>
              
              <div className="welcome-column">
                <h3 className="welcome-section-title">Recent Workspaces</h3>
                <div className="recent-list">
                  <div className="recent-item" onClick={() => {
                    setSelectedBoard("STM32F401");
                    setSelectedProbe("ST-Link V2");
                    setShowWelcomeScreen(false);
                  }}>
                    <div className="recent-name">stm32-motor-driver</div>
                    <div className="recent-path">~/firmware/stm32-motor-driver</div>
                  </div>
                  <div className="recent-item" onClick={() => {
                    setSelectedBoard("ESP32-S3");
                    setSelectedProbe("J-Link");
                    setShowWelcomeScreen(false);
                  }}>
                    <div className="recent-name">esp32-wifi-node</div>
                    <div className="recent-path">~/iot/esp32-wifi-node</div>
                  </div>
                </div>
              </div>
            </div>
            
            <div className="welcome-footer">
              <div className="welcome-footer-logo">HARDCOREAI v1.0.0 (Renderer: React)</div>
              <button className="welcome-enter-btn" onClick={() => setShowWelcomeScreen(false)}>
                <span>Open Workspace</span>
                <ArrowRight size={14} />
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="helix-app">
      {/* 1. Header Command Bar */}
      <header className="helix-header">
        <div className="logo-section">
          <div className="logo-text">HARDCOREAI</div>
          <div className="target-dropdown-pill">
            <span>Target: {selectedBoard}RETx</span>
            <ChevronDown size={11} className="target-dropdown-arrow" />
          </div>
        </div>

        {/* Center Actions Capsule */}
        <div className="command-capsule">
          <button
            className="capsule-btn build"
            onClick={handleBuild}
            disabled={isCompiling || isFlashing}
            title="Compile Project"
          >
            <Play size={12} className="play-triangle-fill" />
            <span>{isCompiling ? "Compiling..." : "Build"}</span>
          </button>
          
          <div className="divider-line"></div>

          <button
            className="capsule-btn flash"
            onClick={handleFlash}
            disabled={isCompiling || isFlashing}
            title="Flash to Device"
          >
            <Zap size={12} />
            <span>{isFlashing ? "Flashing..." : "Flash"}</span>
          </button>

          <div className="divider-line"></div>

          <button
            className={`capsule-btn debug ${isDebugging ? (crashed ? "active crashed" : "active debug-running") : ""}`}
            onClick={handleDebugToggle}
            title="Toggle Debugger (OpenOCD + GDB)"
          >
            <Bug size={12} />
            <span>{isDebugging ? (crashed ? "CRASHED" : "Debug") : "Debug"}</span>
          </button>

          <div className="divider-line"></div>
          
          <div className="capsule-more-options">
            <MoreHorizontal size={13} style={{ color: "var(--text-muted)", cursor: "pointer", padding: "0 4px" }} />
          </div>
        </div>

        {/* Connectivity Status & Tauri Windows controls */}
        <div className="connection-status-group">
          <div className="connection-status">
            {isDebugging && !crashed && (
              <div className="command-capsule" style={{ background: "rgba(6, 182, 212, 0.08)", borderColor: "rgba(6, 182, 212, 0.3)" }}>
                <button className="capsule-btn" style={{ color: "var(--accent-cyan)", padding: "4px 8px" }} onClick={stepOver}>
                  <span>Step Over</span>
                </button>
                <div className="divider-line" style={{ backgroundColor: "rgba(6, 182, 212, 0.3)" }}></div>
                <button className="capsule-btn" style={{ color: "var(--accent-cyan)", padding: "4px 8px" }} onClick={continueExecution}>
                  <span>Run</span>
                </button>
              </div>
            )}
            
            {!crashed ? (
              <button className="status-pill" onClick={() => triggerCrash()} style={{ borderColor: "rgba(239, 68, 68, 0.2)", cursor: "pointer" }} title="Trigger Heat Loop Overheat Exception">
                <span className="status-dot" style={{ backgroundColor: "#EF4444" }}></span>
                <span style={{ color: "#EF4444" }}>Simulate Overheat</span>
              </button>
            ) : (
              <button className="status-pill" onClick={resolveCrash} style={{ borderColor: "rgba(16, 185, 129, 0.4)", cursor: "pointer" }} title="Apply Code Patch Fix">
                <span className="status-dot active"></span>
                <span style={{ color: "#10B981" }}>Apply AI Patch</span>
              </button>
            )}

            <div className="status-pill" onClick={toggleSerialConnection} style={{ cursor: "pointer" }} title="Toggle UART Serial Port Connection">
              <span className={`status-dot ${serialConnected ? "active" : ""}`}></span>
              <span>{serialConnected ? `UART COM4: ${baudRate}` : "UART Offline"}</span>
            </div>

            <div className="status-pill">
              <span className="status-dot ai-active"></span>
              <span>AI Ready</span>
            </div>
          </div>

          {/* Quick Access Top Right */}
          <div className="tauri-controls-group">
            <Search size={14} className="control-icon-btn" />
            <Settings size={14} className="control-icon-btn" />
            <Moon size={14} className="control-icon-btn" />
            <div className="divider-line" style={{ margin: "0 4px", height: "14px" }}></div>
            <Minus size={13} className="control-icon-btn" />
            <Square size={10} className="control-icon-btn" />
            <X size={14} className="control-icon-btn close-btn-highlight" />
          </div>
        </div>
      </header>

      {/* 2. Main Workspace Layout */}
      <div className={`helix-main-workspace ${isDebugging ? "debug-active" : ""} ${aiOpen ? "ai-open" : ""}`}>
        
        {/* Leftmost Activity Bar */}
        <div className="activity-bar">
          <div
            className={`activity-item ${activeSidebarTab === "explorer" ? "active" : ""}`}
            onClick={() => setActiveSidebarTab("explorer")}
            title="Explorer"
          >
            <Folder size={18} />
          </div>
          <div
            className={`activity-item ${activeSidebarTab === "search" ? "active" : ""}`}
            onClick={() => setActiveSidebarTab("search")}
            title="Search"
          >
            <Search size={18} />
          </div>
          <div
            className={`activity-item ${activeSidebarTab === "git" ? "active" : ""}`}
            onClick={() => setActiveSidebarTab("git")}
            title="Source Control"
          >
            <GitBranch size={18} />
          </div>
          <div
            className={`activity-item ${activeSidebarTab === "debug" ? "active" : ""}`}
            onClick={() => setActiveSidebarTab("debug")}
            title="Run and Debug"
          >
            <Bug size={18} />
          </div>
          <div
            className={`activity-item ${activeSidebarTab === "extensions" ? "active" : ""}`}
            onClick={() => setActiveSidebarTab("extensions")}
            title="Extensions"
          >
            <Blocks size={18} />
          </div>
          <div
            className={`activity-item ${activeSidebarTab === "boards" ? "active" : ""}`}
            onClick={() => setActiveSidebarTab("boards")}
            title="Target Configurations"
          >
            <Settings size={18} />
          </div>
          
          <div style={{ flex: 1 }}></div>
          
          <div
            className="activity-item"
            onClick={() => setShowWelcomeScreen(true)}
            title="Open Welcome Screen"
            style={{ color: "var(--text-dark)" }}
          >
            <HelpCircle size={18} />
          </div>
        </div>

        {/* Left Sidebar explorer/board tree */}
        <aside className="workspace-panel" style={{ width: `${sidebarWidth}px`, flexShrink: 0, position: "relative" }}>
          {renderSidebarContent()}
          {/* Vertical Resizer Handle (Right Edge of Left Sidebar) */}
          <div
            className="resizer-handle"
            onMouseDown={() => setIsDraggingLeft(true)}
            style={{
              position: "absolute",
              top: 0,
              right: 0,
              bottom: 0,
              width: "4px",
              cursor: "col-resize",
              zIndex: 10
            }}
          />
        </aside>

        {/* Center Panel: Code Editor */}
        <section className="editor-container">
          <div className="editor-tabs">
            {Object.keys(fileContents).map(filePath => {
              const fileName = filePath.split("/").pop() || filePath;
              const isActive = activeFile === filePath;
              return (
                <div
                  key={filePath}
                  className={`editor-tab ${isActive ? "active" : ""}`}
                  onClick={() => setActiveFile(filePath)}
                >
                  <FileCode size={12} style={{ color: isActive ? "var(--accent-violet-hover)" : "var(--text-muted)" }} />
                  <span>{fileName}</span>
                  {isActive && <div className="active-tab-top-bar"></div>}
                </div>
              );
            })}
            <div className="add-tab-btn">
              <Plus size={13} />
            </div>
          </div>
          
          <div className="monaco-editor-wrapper">
            <Editor
              height="100%"
              theme="vs-dark"
              path={activeFile || "main.c"}
              language="cpp"
              value={getCurrentContent()}
              onChange={handleEditorChange}
              options={{
                fontFamily: "var(--font-mono)",
                fontSize: 13,
                minimap: { enabled: false },
                lineNumbers: "on",
                glyphMargin: true,
                cursorBlinking: "smooth",
                scrollbar: {
                  verticalScrollbarSize: 6,
                  horizontalScrollbarSize: 6
                },
                guides: {
                  indentation: true
                },
                renderLineHighlight: "all"
              }}
              onMount={(_, monaco) => {
                monaco.editor.defineTheme("hardcoreaiTheme", {
                  base: "vs-dark",
                  inherit: true,
                  rules: [
                    { token: "comment", foreground: "10B981", fontStyle: "italic" }, // Emerald comments matching screenshot
                    { token: "keyword", foreground: "8B5CF6", fontStyle: "bold" },
                    { token: "string", foreground: "EF4444" },
                    { token: "number", foreground: "06B6D4" },
                    { token: "type", foreground: "F1F5F9" }
                  ],
                  colors: {
                    "editor.background": "#050508",
                    "editor.lineHighlightBackground": "#121217",
                    "editorGutter.background": "#050508",
                    "editorCursor.foreground": "#8B5CF6"
                  }
                });
                monaco.editor.setTheme("hardcoreaiTheme");
              }}
            />

            {/* Visual debugger indicator overlay */}
            {isDebugging && currentLine !== null && (
              <div
                className={`debug-line-overlay ${crashed ? "crashed" : ""}`}
                style={{
                  top: `${(currentLine - 1) * 19 + 5}px`,
                  height: "19px"
                }}
              />
            )}
          </div>
        </section>

        {/* 3-Panel Stacked Right Sidebar */}
        {!aiOpen ? (
          <div className="ai-collapse-strip" onClick={() => setAiOpen(true)}>
            <Sparkles size={16} className="ai-collapse-strip-icon" />
            <div className="ai-collapse-text">AI Assistant</div>
          </div>
        ) : (
          <aside className="workspace-panel right split-sidebar-right" style={{ width: `${rightSidebarWidth}px`, flexShrink: 0, position: "relative" }}>
            {/* Vertical Resizer Handle (Left Edge of Right Sidebar) */}
            <div
              className="resizer-handle"
              onMouseDown={() => setIsDraggingRight(true)}
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                bottom: 0,
                width: "4px",
                cursor: "col-resize",
                zIndex: 10
              }}
            />
            {/* PANE 1: EMBEDDED CONFIGURATOR */}
            <div className="sidebar-right-pane embedded-configurator-pane">
              <div className="panel-header">
                <div className="panel-title">
                  <span>EMBEDDED CONFIGURATOR</span>
                </div>
                <div className="pane-header-actions">
                  <RotateCcw size={11} className="pane-action-icon" />
                  <Maximize2 size={11} className="pane-action-icon" />
                  <X size={12} className="pane-action-iconClose" onClick={() => setAiOpen(false)} />
                </div>
              </div>
              <div className="pane-tabs">
                <div className="pane-tab">Pinout</div>
                <div className="pane-tab">Clock</div>
                <div className="pane-tab active">Configuration</div>
                <div className="pane-tab">Project</div>
              </div>
              <div className="pane-content configurator-split">
                <div className="configurator-left-col">
                  <div className="configurator-search-box">
                    <Search size={11} className="search-icon-inside" />
                    <input type="text" placeholder="Search (Ctrl+F)" readOnly />
                  </div>
                  <div className="configurator-category-list">
                    <div className="category-item-group">System Core <ChevronRight size={9} /></div>
                    <div className="category-item-group">Analog <ChevronRight size={9} /></div>
                    <div className="category-item-group">Timers <ChevronRight size={9} /></div>
                    <div className="category-item-group">Connectivity <ChevronRight size={9} /></div>
                    <div className="category-item-group">Multimedia <ChevronRight size={9} /></div>
                    <div className="category-item-group">Security <ChevronRight size={9} /></div>
                    <div className="category-item-group">Computing <ChevronRight size={9} /></div>
                    <div className="category-item-group text-active">Middleware <ChevronRight size={9} /></div>
                  </div>
                </div>
                <div className="configurator-right-col">
                  <div className="configurator-chip-header">
                    <span className="chip-name-title">{selectedBoard}RETx</span>
                    <button className="datasheet-btn">Datasheet</button>
                  </div>
                  
                  {/* High-fidelity visual representing the ST Microelectronics chip */}
                  <div className="visual-mcu-chip-wrapper">
                    <div className="visual-mcu-chip">
                      <div className="mcu-corner-notch"></div>
                      
                      {/* Interactive Pine rows top, bottom, left, right */}
                      <div className="mcu-pins-row pins-top">
                        {Array.from({ length: 12 }).map((_, i) => <div key={i} className="mcu-pin"></div>)}
                      </div>
                      <div className="mcu-pins-row pins-bottom">
                        {Array.from({ length: 12 }).map((_, i) => <div key={i} className="mcu-pin"></div>)}
                      </div>
                      <div className="mcu-pins-row pins-left">
                        {Array.from({ length: 12 }).map((_, i) => <div key={i} className="mcu-pin"></div>)}
                      </div>
                      <div className="mcu-pins-row pins-right">
                        {Array.from({ length: 12 }).map((_, i) => <div key={i} className="mcu-pin"></div>)}
                      </div>

                      <div className="visual-mcu-chip-body">
                        <div className="mcu-logo-brand">ST</div>
                        <div className="mcu-model-text">{selectedBoard}RETx</div>
                        <div className="mcu-package-text">LQFP64</div>
                      </div>
                    </div>
                  </div>

                  <div className="configurator-specs-list">
                    <div className="spec-row-item">
                      <span className="spec-label">Flash</span>
                      <span className="spec-value">512 KB</span>
                    </div>
                    <div className="spec-row-item">
                      <span className="spec-label">RAM</span>
                      <span className="spec-value">96 KB</span>
                    </div>
                    <div className="spec-row-item">
                      <span className="spec-label">Max Speed</span>
                      <span className="spec-value">84 MHz</span>
                    </div>
                    <div className="spec-row-item">
                      <span className="spec-label">Core</span>
                      <span className="spec-value">Cortex-M4</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* PANE 2: HARDCOREAI COPILOT */}
            <div className="sidebar-right-pane ai-copilot-pane">
              <div className="panel-header">
                <div className="panel-title">
                  <Sparkles size={11} style={{ color: "var(--accent-violet)", marginRight: "4px" }} />
                  <span>HARDCOREAI COPILOT</span>
                </div>
                <div className="pane-header-actions">
                  <RotateCcw size={11} className="pane-action-icon" />
                  <Maximize2 size={11} className="pane-action-icon" />
                  <X size={12} className="pane-action-iconClose" onClick={() => setAiOpen(false)} />
                </div>
              </div>
              <div className="pane-content ai-copilot-chat-content">
                {/* Simulated crash details warning inside Copilot */}
                {crashed && (
                  <div className="crash-diagnostics-panel" style={{ margin: "0 0 10px 0" }}>
                    <div className="crash-header">
                      <AlertTriangle size={14} />
                      <span>CPU Crash Halted</span>
                    </div>
                    <div className="crash-msg">{crashReason}</div>
                    <button className="crash-btn-fix" onClick={resolveCrash}>
                      <span>Apply AI Patch Fix</span>
                      <ArrowRight size={12} />
                    </button>
                  </div>
                )}

                <div className="chat-welcome-bubble">
                  <p className="greeting-text">Hello! I'm HardcoreAI Copilot.</p>
                  <p className="sub-text">I can help you with:</p>
                  <ul className="help-topics-list">
                    <li>✦ Explaining code</li>
                    <li>✦ Fixing errors</li>
                    <li>✦ Embedded debugging</li>
                    <li>✦ Memory & register analysis</li>
                    <li>✦ HAL & peripheral usage</li>
                  </ul>
                </div>
                
                {/* Dynamically renders other messages in stack if sent */}
                {aiMessages.slice(1).map((msg) => (
                  <div key={msg.id} className={`chat-bubble ${msg.sender}`} style={{ marginTop: "8px" }}>
                    <div style={{ whiteSpace: "pre-wrap" }}>{msg.text}</div>
                    <span className="timestamp">{msg.timestamp}</span>
                  </div>
                ))}

                {aiWaiting && (
                  <div className="chat-bubble ai" style={{ opacity: 0.7, marginTop: "8px" }}>
                    <p>Copilot is analyzing registers and stack frame...</p>
                  </div>
                )}

                <form onSubmit={handleAiSend} className="chat-input-row-wrapper">
                  <div className="chat-input-inner">
                    <input
                      type="text"
                      placeholder="Ask anything about your code or project..."
                      value={aiInput}
                      onChange={(e) => setAiInput(e.target.value)}
                    />
                    <button type="submit" className="send-circle-btn">
                      <Send size={11} />
                    </button>
                  </div>
                </form>
              </div>
            </div>

            {/* PANE 3: VARIABLES / WATCH / CALL STACK / BREAKPOINTS */}
            <div className="sidebar-right-pane debug-variables-pane">
              <div className="pane-tabs">
                <div className="pane-tab active">VARIABLES</div>
                <div className="pane-tab">WATCH</div>
                <div className="pane-tab">CALL STACK</div>
                <div className="pane-tab">BREAKPOINTS</div>
                <div style={{ flex: 1 }}></div>
                <MoreHorizontal size={13} className="pane-action-icon" style={{ padding: "0 4px" }} />
              </div>
              <div className="pane-content variables-table-content">
                <table className="variables-data-table">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Value</th>
                      <th>Type</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td className="var-name">counter</td>
                      <td className="var-value-cell"><span className="purple-badge">42</span></td>
                      <td className="var-type">uint32_t</td>
                    </tr>
                    <tr>
                      <td className="var-name">status</td>
                      <td className="var-value-cell">0</td>
                      <td className="var-type">uint8_t</td>
                    </tr>
                    <tr>
                      <td className="var-name">errorCode</td>
                      <td className="var-value-cell">HAL_OK</td>
                      <td className="var-type-muted">HAL_StatusTypeDef</td>
                    </tr>
                    <tr className="expandable-row">
                      <td colSpan={3} className="globals-header-row">
                        <ChevronRight size={10} style={{ transform: "rotate(0deg)", marginRight: "2px" }} /> Globals
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </aside>
        )}
      </div>

      {/* 3. Bottom Instrumentation Drawer */}
      <footer className="helix-bottom-drawer" style={{ height: `${bottomDrawerHeight}px`, position: "relative" }}>
        {/* Horizontal Resizer Handle (Top Edge of Bottom Drawer) */}
        <div
          className="resizer-handle"
          onMouseDown={() => setIsDraggingBottom(true)}
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            right: 0,
            height: "4px",
            cursor: "row-resize",
            zIndex: 10
          }}
        />
        <div className="drawer-tabs">
          <div className="tab-group">
            <div className="drawer-tab">PROBLEMS</div>
            <div className="drawer-tab">OUTPUT</div>
            <div className="drawer-tab">DEBUG CONSOLE</div>
            <div className="drawer-tab">TERMINAL</div>
            <div
              className={`drawer-tab ${activeBottomTab === "terminal" ? "active" : ""}`}
              onClick={() => setBottomTab("terminal")}
            >
              <span>SERIAL MONITOR</span>
            </div>
            <div
              className={`drawer-tab ${activeBottomTab === "plotter" ? "active" : ""}`}
              onClick={() => setBottomTab("plotter")}
            >
              <span>MEMORY</span>
            </div>
            <div
              className={`drawer-tab ${activeBottomTab === "registers" ? "active" : ""}`}
              onClick={() => setBottomTab("registers")}
            >
              <span>REGISTERS</span>
            </div>
          </div>
        </div>

        <div className="drawer-content">
          
          {/* TAB 1: Serial Monitor Console */}
          {activeBottomTab === "terminal" && (
            <div className="serial-panel">
              {/* Custom Toolbar matching Serial Monitor Controls */}
              <div className="serial-monitor-toolbar">
                <div className="toolbar-select-group">
                  <span className="select-label">Port</span>
                  <select className="serial-select-dropdown" value="COM4" disabled>
                    <option value="COM4">COM4</option>
                  </select>
                </div>
                <div className="toolbar-select-group">
                  <span className="select-label">Baud Rate</span>
                  <select className="serial-select-dropdown" value={baudRate.toString()} disabled>
                    <option value="115200">115200</option>
                  </select>
                </div>
                <div className="toolbar-checkbox-group">
                  <input type="checkbox" id="chkAutoScroll" defaultChecked />
                  <label htmlFor="chkAutoScroll" className="checkbox-label-styled">Auto Scroll</label>
                </div>
                <div className="toolbar-checkbox-group">
                  <input type="checkbox" id="chkTimestamp" />
                  <label htmlFor="chkTimestamp" className="checkbox-label-styled">Show Timestamp</label>
                </div>
                
                <div style={{ flex: 1 }}></div>

                {/* Right controls */}
                <div className="toolbar-actions-group">
                  <div className="action-circle-icon" onClick={toggleSerialConnection}>
                    <Play size={10} style={{ fill: serialConnected ? "currentColor" : "none" }} />
                  </div>
                  <div className="action-circle-icon" onClick={() => addSerialLog("--- Logs Cleared ---")}>
                    <RotateCcw size={10} />
                  </div>
                  <div className="action-circle-icon">
                    <Settings size={10} />
                  </div>
                </div>
              </div>

              {/* Log stream box with high-fidelity color formatting */}
              <div className="terminal-scroll">
                {serialLogs.map((line, idx) => (
                  <div key={idx} className="terminal-line" style={{ display: "flex", gap: "6px" }}>
                    {line.startsWith("[") && line.indexOf("]") !== -1 ? (
                      <>
                        <span className="log-timestamp">{line.substring(0, line.indexOf("]") + 1)}</span>
                        <span className="log-content-text">{line.substring(line.indexOf("]") + 2)}</span>
                      </>
                    ) : (
                      <span className="log-content-text">{line}</span>
                    )}
                  </div>
                ))}
                <div ref={terminalEndRef} />
              </div>
              
              <div className="terminal-input-bar">
                <input
                  type="text"
                  className="terminal-input"
                  placeholder="Send data to COM4 UART line (e.g. reboot)..."
                  value={serialInput}
                  onChange={(e) => setSerialInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && serialInput.trim()) {
                      addSerialLog(`> ${serialInput}`);
                      setSerialInput("");
                    }
                  }}
                />
              </div>
            </div>
          )}

          {/* TAB 2: Telemetry Plotter */}
          {activeBottomTab === "plotter" && (
            <div className="plotter-panel">
              <div className="plotter-canvas-container">
                <canvas ref={canvasRef} style={{ width: "100%", height: "100%" }} />
              </div>
              <div className="plotter-controls">
                <div className="plotter-metric-badge">
                  <span style={{ width: "6px", height: "6px", borderRadius: "50%", backgroundColor: "#8B5CF6" }}></span>
                  <span style={{ color: "var(--text-active)" }}>TEMP_C: {plotData[plotData.length - 1]?.temp.toFixed(2) || "25.00"}°C</span>
                </div>
                <div className="plotter-metric-badge">
                  <span style={{ width: "6px", height: "6px", borderRadius: "50%", backgroundColor: "#06B6D4" }}></span>
                  <span style={{ color: "var(--text-active)" }}>VDD: {plotData[plotData.length - 1]?.voltage.toFixed(2) || "3.30"}V</span>
                </div>
                <div className="plotter-metric-badge">
                  <span style={{ width: "6px", height: "6px", borderRadius: "50%", backgroundColor: "#CBD5E1" }}></span>
                  <span style={{ color: "var(--text-active)" }}>IDD: {plotData[plotData.length - 1]?.current.toFixed(1) || "40.0"}mA</span>
                </div>
              </div>
            </div>
          )}

          {/* TAB 3: SVD Registers Viewer */}
          {activeBottomTab === "registers" && (
            <div className="registers-panel">
              <div className="peripheral-list">
                {registers.map((reg) => (
                  <div
                    key={reg.name}
                    className={`peripheral-item ${selectedPeripheral === reg.name ? "active" : ""}`}
                    onClick={() => setSelectedPeripheral(reg.name)}
                  >
                    <span>{reg.name}</span>
                    <span className="peripheral-address">{reg.value}</span>
                  </div>
                ))}
              </div>
              <div className="register-details-grid">
                {registers
                  .find((r) => r.name === selectedPeripheral)
                  ?.bits?.map((bit) => (
                    <div key={bit.name} className="register-row">
                      <div className="register-row-header">
                        <span className="register-name">{bit.name}</span>
                        <span className="register-value">0x{bit.value.toString(16).toUpperCase()}</span>
                      </div>
                      <div className="register-desc">{bit.description} (Bits: {bit.range})</div>
                      <div className="register-bitfield">
                        {Array.from({ length: 32 }, (_, i) => 31 - i).map((bitNum) => {
                          const bitValue = (bit.value >> bitNum) & 1;
                          return (
                            <div
                              key={bitNum}
                              className={`bit-box ${bitValue === 1 ? "active" : ""}`}
                              title={`Bit ${bitNum}: ${bitValue}`}
                            >
                              {bitNum}
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  ))}
              </div>
            </div>
          )}

        </div>
      </footer>

      {/* 4. Bottom Status Bar */}
      <footer className="hardcoreai-status-bar">
        <div className="status-bar-left">
          <div className="status-bar-item branch-name">
            <GitBranch size={11} />
            <span>main</span>
          </div>
          <div className="status-bar-item sync-icon">
            <RotateCcw size={10} />
          </div>
          <div className="status-bar-item diagnostic-error">
            <span className="error-circle-small">x</span>
            <span>0</span>
          </div>
          <div className="status-bar-item diagnostic-warning">
            <AlertTriangle size={11} />
            <span>0</span>
          </div>
          <div className="status-bar-item active-board-name">
            <span>{selectedBoard}RETx</span>
          </div>
        </div>

        <div className="status-bar-right">
          <div className="status-bar-item">Ln 26, Col 1</div>
          <div className="status-bar-item">Spaces: 4</div>
          <div className="status-bar-item">UTF-8</div>
          <div className="status-bar-item">LF</div>
          <div className="status-bar-item">C</div>
          <div className="status-bar-item">{selectedBoard}RETx</div>
          <div className="status-bar-item notification-icon">
            <HelpCircle size={11} />
          </div>
          <div className="status-bar-item system-ready-status">
            <span className="ready-dot-glow"></span>
            <span>Ready</span>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default App;
