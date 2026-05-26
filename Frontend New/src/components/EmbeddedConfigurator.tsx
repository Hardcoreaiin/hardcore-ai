import React, { useState } from "react";
import "./EmbeddedConfigurator.css";
import {
  X,
  Search,
  ChevronRight,
  ChevronDown,
  Maximize2,
  Minimize2,
  RotateCcw,
  ExternalLink,
  Cpu,
  Zap,
  Clock,
  Wifi,
  Lock,
  Monitor,
  Activity,
  Layers,
  FileCode,
  AlertTriangle,
  CheckCircle,
} from "lucide-react";

interface PinConfig {
  id: string;
  name: string;
  mode: string;
  signal: string;
  label: string;
  enabled: boolean;
}

interface PeripheralConfig {
  name: string;
  enabled: boolean;
  mode?: string;
  speed?: string;
  pullResistor?: string;
  params?: Record<string, string>;
}

interface CategoryItem {
  id: string;
  label: string;
  icon: React.ReactNode;
  children: { id: string; label: string; description: string }[];
}

interface Props {
  selectedBoard: string;
  onClose: () => void;
  isDetached: boolean;
  onDetach: () => void;
}

const BOARD_SPECS: Record<string, { flash: string; ram: string; speed: string; core: string; pins: number; package: string }> = {
  "STM32F401": { flash: "512 KB", ram: "96 KB", speed: "84 MHz", core: "Cortex-M4", pins: 64, package: "LQFP64" },
  "ESP32-S3":  { flash: "16 MB", ram: "512 KB", speed: "240 MHz", core: "Xtensa LX7 (Dual)", pins: 45, package: "QFN56" },
  "RP2040":    { flash: "2 MB", ram: "264 KB", speed: "133 MHz", core: "Cortex-M0+ (Dual)", pins: 40, package: "QFN56" },
};

const CATEGORIES: CategoryItem[] = [
  {
    id: "system-core",
    label: "System Core",
    icon: <Cpu size={13} />,
    children: [
      { id: "rcc", label: "RCC", description: "Reset and Clock Control" },
      { id: "gpio", label: "GPIO", description: "General Purpose I/O" },
      { id: "nvic", label: "NVIC", description: "Nested Vectored Interrupt Controller" },
      { id: "sys", label: "SYS", description: "System Configuration" },
      { id: "dma", label: "DMA", description: "Direct Memory Access" },
    ],
  },
  {
    id: "analog",
    label: "Analog",
    icon: <Activity size={13} />,
    children: [
      { id: "adc1", label: "ADC1", description: "Analog-to-Digital Converter 1" },
      { id: "adc2", label: "ADC2", description: "Analog-to-Digital Converter 2" },
      { id: "dac", label: "DAC", description: "Digital-to-Analog Converter" },
      { id: "comp", label: "COMP", description: "Analog Comparator" },
    ],
  },
  {
    id: "timers",
    label: "Timers",
    icon: <Clock size={13} />,
    children: [
      { id: "tim1", label: "TIM1", description: "Advanced Timer 1 (PWM, Encoder)" },
      { id: "tim2", label: "TIM2", description: "General Purpose Timer 2" },
      { id: "tim3", label: "TIM3", description: "General Purpose Timer 3" },
      { id: "tim4", label: "TIM4", description: "General Purpose Timer 4" },
      { id: "tim5", label: "TIM5", description: "General Purpose Timer 5" },
      { id: "tim9", label: "TIM9", description: "General Purpose Timer 9" },
      { id: "systick", label: "SysTick", description: "System Tick Timer (HAL)" },
    ],
  },
  {
    id: "connectivity",
    label: "Connectivity",
    icon: <Wifi size={13} />,
    children: [
      { id: "usart1", label: "USART1", description: "Universal Sync/Async Receiver Transmitter 1" },
      { id: "usart2", label: "USART2", description: "Universal Sync/Async Receiver Transmitter 2" },
      { id: "usart6", label: "USART6", description: "Universal Sync/Async Receiver Transmitter 6" },
      { id: "spi1", label: "SPI1", description: "Serial Peripheral Interface 1" },
      { id: "spi2", label: "SPI2", description: "Serial Peripheral Interface 2" },
      { id: "i2c1", label: "I2C1", description: "Inter-Integrated Circuit 1" },
      { id: "i2c2", label: "I2C2", description: "Inter-Integrated Circuit 2" },
      { id: "usb",  label: "USB OTG FS", description: "USB On-The-Go Full Speed" },
      { id: "sdio", label: "SDIO", description: "Secure Digital I/O Interface" },
    ],
  },
  {
    id: "multimedia",
    label: "Multimedia",
    icon: <Monitor size={13} />,
    children: [
      { id: "i2s",  label: "I2S", description: "Inter-IC Sound Controller" },
      { id: "dcmi", label: "DCMI", description: "Digital Camera Interface" },
    ],
  },
  {
    id: "security",
    label: "Security",
    icon: <Lock size={13} />,
    children: [
      { id: "crc",  label: "CRC", description: "Cyclic Redundancy Check" },
      { id: "rng",  label: "RNG", description: "Random Number Generator" },
    ],
  },
  {
    id: "computing",
    label: "Computing",
    icon: <Layers size={13} />,
    children: [
      { id: "fpu",  label: "FPU", description: "Floating Point Unit (Cortex-M4)" },
      { id: "mpu",  label: "MPU", description: "Memory Protection Unit" },
    ],
  },
  {
    id: "middleware",
    label: "Middleware",
    icon: <FileCode size={13} />,
    children: [
      { id: "freertos",  label: "FreeRTOS", description: "Real-Time Operating System" },
      { id: "fatfs",    label: "FatFS",    description: "FAT File System" },
      { id: "lwip",     label: "LwIP",     description: "Lightweight IP Stack" },
      { id: "usb-dev",  label: "USB Device", description: "USB Device Library" },
    ],
  },
];

const PERIPHERAL_DEFAULTS: Record<string, PeripheralConfig> = {
  gpio:   { name: "GPIO",   enabled: true,  mode: "Output Push Pull", speed: "High", pullResistor: "No pull-up/down" },
  usart1: { name: "USART1", enabled: false, mode: "Asynchronous", params: { BaudRate: "115200", WordLength: "8 Bits", StopBits: "1", Parity: "None" } },
  usart2: { name: "USART2", enabled: false, mode: "Asynchronous", params: { BaudRate: "115200", WordLength: "8 Bits", StopBits: "1", Parity: "None" } },
  spi1:   { name: "SPI1",   enabled: false, mode: "Full-Duplex Master", params: { Prescaler: "2", CPOL: "Low", CPHA: "1 Edge", DataSize: "8 Bits" } },
  i2c1:   { name: "I2C1",   enabled: false, mode: "I2C", params: { SpeedMode: "Standard (100 kHz)", OwnAddress: "0x00", DualAddress: "Disabled" } },
  tim1:   { name: "TIM1",   enabled: false, mode: "PWM Generation", params: { Prescaler: "83", CounterPeriod: "999", ClockDivision: "No Division" } },
  tim2:   { name: "TIM2",   enabled: false, mode: "Up Counter", params: { Prescaler: "83", CounterPeriod: "999" } },
  adc1:   { name: "ADC1",   enabled: false, mode: "Independent Mode", params: { Resolution: "12 bits", DataAlignment: "Right", Channels: "IN0" } },
  rcc:    { name: "RCC",    enabled: true,  mode: "Crystal/Ceramic Resonator", params: { HSE: "8 MHz", LSE: "32.768 kHz", SYSCLK: "84 MHz" } },
  dma:    { name: "DMA",    enabled: false, mode: "Normal", params: { Priority: "Low", Direction: "P→M" } },
  freertos: { name: "FreeRTOS", enabled: false, mode: "CMSIS_V1", params: { "Heap Size": "3072 B", "Tick Rate": "1000 Hz" } },
  usb:    { name: "USB OTG FS", enabled: false, mode: "Device_Only", params: { Speed: "Full Speed", VBUS: "Enabled" } },
  fpu:    { name: "FPU", enabled: true, mode: "FP Instructions", params: { "Mode": "Enabled (full)" } },
};

const ACTIVE_TABS = ["Pinout", "Clock", "Configuration", "Project"] as const;
type TabType = typeof ACTIVE_TABS[number];

// ── MCU Pin Visual Component ────────────────────────────────────────────────
const MCUPinChip: React.FC<{ board: string }> = ({ board }) => {
  const specs = BOARD_SPECS[board] ?? BOARD_SPECS["STM32F401"];
  const pinCount = Math.min(specs.pins, 16);
  return (
    <div className="ec-chip-wrapper">
      <div className="ec-chip">
        <div className="ec-chip-notch" />
        <div className="ec-chip-pins ec-pins-top">
          {Array.from({ length: pinCount }).map((_, i) => <div key={i} className="ec-pin" />)}
        </div>
        <div className="ec-chip-pins ec-pins-bottom">
          {Array.from({ length: pinCount }).map((_, i) => <div key={i} className="ec-pin" />)}
        </div>
        <div className="ec-chip-pins ec-pins-left">
          {Array.from({ length: pinCount }).map((_, i) => <div key={i} className="ec-pin ec-pin-h" />)}
        </div>
        <div className="ec-chip-pins ec-pins-right">
          {Array.from({ length: pinCount }).map((_, i) => <div key={i} className="ec-pin ec-pin-h" />)}
        </div>
        <div className="ec-chip-body">
          <div className="ec-brand">ST</div>
          <div className="ec-model">{board}RETx</div>
          <div className="ec-package">{specs.package}</div>
        </div>
      </div>
    </div>
  );
};

// ── Configuration Detail Panel ───────────────────────────────────────────────
const PeripheralDetail: React.FC<{
  peripheralId: string;
  config: PeripheralConfig;
  onToggle: () => void;
  onParamChange: (key: string, value: string) => void;
}> = ({ peripheralId, config, onToggle, onParamChange }) => {
  return (
    <div className="ec-detail-panel">
      <div className="ec-detail-header">
        <div className="ec-detail-title">
          {config.enabled
            ? <CheckCircle size={14} style={{ color: "var(--accent-success)" }} />
            : <AlertTriangle size={14} style={{ color: "var(--text-dark)" }} />}
          <span>{config.name}</span>
        </div>
        <label className="ec-toggle-switch" title={config.enabled ? "Click to disable" : "Click to enable"}>
          <input
            type="checkbox"
            checked={config.enabled}
            onChange={onToggle}
            style={{ display: "none" }}
          />
          <div className={`ec-toggle-track ${config.enabled ? "on" : ""}`}>
            <div className="ec-toggle-thumb" />
          </div>
        </label>
      </div>

      {config.enabled && (
        <div className="ec-detail-body">
          <div className="ec-param-row">
            <label className="ec-param-label">Mode</label>
            <select
              className="ec-param-select"
              value={config.mode ?? ""}
              onChange={e => onParamChange("mode", e.target.value)}
            >
              <option value={config.mode ?? ""}>{config.mode ?? "Default"}</option>
            </select>
          </div>

          {config.speed && (
            <div className="ec-param-row">
              <label className="ec-param-label">Speed</label>
              <select
                className="ec-param-select"
                value={config.speed}
                onChange={e => onParamChange("speed", e.target.value)}
              >
                {["Low", "Medium", "High", "Very High"].map(s => (
                  <option key={s} value={s}>{s}</option>
                ))}
              </select>
            </div>
          )}

          {config.pullResistor && (
            <div className="ec-param-row">
              <label className="ec-param-label">Pull</label>
              <select
                className="ec-param-select"
                value={config.pullResistor}
                onChange={e => onParamChange("pullResistor", e.target.value)}
              >
                {["No pull-up/down", "Pull-up", "Pull-down"].map(p => (
                  <option key={p} value={p}>{p}</option>
                ))}
              </select>
            </div>
          )}

          {config.params && Object.entries(config.params).map(([key, val]) => (
            <div className="ec-param-row" key={key}>
              <label className="ec-param-label">{key}</label>
              <input
                className="ec-param-input"
                value={val}
                onChange={e => onParamChange(key, e.target.value)}
              />
            </div>
          ))}

          <div className="ec-detail-badge">
            {config.enabled ? "✓ Peripheral Active" : "○ Peripheral Inactive"}
          </div>
        </div>
      )}

      {!config.enabled && (
        <div className="ec-disabled-msg">
          Click toggle to enable {config.name} on this device.
        </div>
      )}
    </div>
  );
};

// ── Main Embedded Configurator ────────────────────────────────────────────────
const EmbeddedConfigurator: React.FC<Props> = ({ selectedBoard, onClose, isDetached, onDetach }) => {
  const [activeTab, setActiveTab] = useState<TabType>("Configuration");
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set(["system-core"]));
  const [selectedPeripheral, setSelectedPeripheral] = useState<string>("gpio");
  const [searchQuery, setSearchQuery] = useState("");
  const [peripheralConfigs, setPeripheralConfigs] = useState<Record<string, PeripheralConfig>>({
    ...PERIPHERAL_DEFAULTS,
  });

  const specs = BOARD_SPECS[selectedBoard] ?? BOARD_SPECS["STM32F401"];

  const toggleCategory = (id: string) => {
    setExpandedCategories(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const togglePeripheral = (id: string) => {
    setPeripheralConfigs(prev => ({
      ...prev,
      [id]: { ...prev[id], enabled: !prev[id]?.enabled },
    }));
  };

  const updateParam = (id: string, key: string, value: string) => {
    setPeripheralConfigs(prev => {
      const curr = prev[id] ?? { name: id, enabled: true };
      if (key === "mode")          return { ...prev, [id]: { ...curr, mode: value } };
      if (key === "speed")         return { ...prev, [id]: { ...curr, speed: value } };
      if (key === "pullResistor")  return { ...prev, [id]: { ...curr, pullResistor: value } };
      return {
        ...prev,
        [id]: { ...curr, params: { ...curr.params, [key]: value } },
      };
    });
  };

  const filteredCategories = CATEGORIES.map(cat => ({
    ...cat,
    children: searchQuery
      ? cat.children.filter(c =>
          c.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.description.toLowerCase().includes(searchQuery.toLowerCase())
        )
      : cat.children,
  })).filter(cat => !searchQuery || cat.children.length > 0);

  const selectedConfig = peripheralConfigs[selectedPeripheral] ?? {
    name: selectedPeripheral.toUpperCase(),
    enabled: false,
  };

  return (
    <div className="ec-root">
      {/* Header */}
      <div className="ec-header">
        <div className="ec-header-left">
          <Cpu size={13} style={{ color: "var(--accent-violet)" }} />
          <span className="ec-header-title">Embedded Configurator</span>
          <span className="ec-board-chip">{selectedBoard}RETx</span>
        </div>
        <div className="ec-header-actions">
          <button className="ec-icon-btn" onClick={onDetach} title={isDetached ? "Dock panel" : "Detach panel"}>
            {isDetached ? <Minimize2 size={12} /> : <ExternalLink size={12} />}
          </button>
          <button className="ec-icon-btn" title="Reset configuration">
            <RotateCcw size={12} />
          </button>
          <button className="ec-icon-btn ec-close-btn" onClick={onClose} title="Close">
            <X size={13} />
          </button>
        </div>
      </div>

      {/* Tab Bar */}
      <div className="ec-tab-bar">
        {ACTIVE_TABS.map(tab => (
          <button
            key={tab}
            className={`ec-tab ${activeTab === tab ? "ec-tab-active" : ""}`}
            onClick={() => setActiveTab(tab)}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* ── PINOUT TAB ── */}
      {activeTab === "Pinout" && (
        <div className="ec-content ec-pinout-tab">
          <div className="ec-pinout-visual">
            <MCUPinChip board={selectedBoard} />
            <div className="ec-pinout-legend">
              <div className="ec-legend-item"><span className="ec-legend-dot" style={{ background: "var(--accent-violet)" }} />GPIO</div>
              <div className="ec-legend-item"><span className="ec-legend-dot" style={{ background: "var(--accent-cyan)" }} />Peripheral</div>
              <div className="ec-legend-item"><span className="ec-legend-dot" style={{ background: "var(--accent-success)" }} />Power</div>
              <div className="ec-legend-item"><span className="ec-legend-dot" style={{ background: "var(--text-dark)" }} />Unassigned</div>
            </div>
          </div>
          <div className="ec-pinout-list">
            <div className="ec-section-title">Pin Assignments</div>
            {[
              { pin: "PA5",  signal: "SPI1_SCK",   mode: "Alternate Function", af: "AF5" },
              { pin: "PA6",  signal: "SPI1_MISO",  mode: "Alternate Function", af: "AF5" },
              { pin: "PA7",  signal: "SPI1_MOSI",  mode: "Alternate Function", af: "AF5" },
              { pin: "PA2",  signal: "USART2_TX",  mode: "Alternate Function", af: "AF7" },
              { pin: "PA3",  signal: "USART2_RX",  mode: "Alternate Function", af: "AF7" },
              { pin: "PB8",  signal: "I2C1_SCL",   mode: "Alternate Function", af: "AF4" },
              { pin: "PB9",  signal: "I2C1_SDA",   mode: "Alternate Function", af: "AF4" },
              { pin: "PC13", signal: "GPIO_Output", mode: "Output Push Pull",  af: "-" },
            ].map(row => (
              <div key={row.pin} className="ec-pin-row">
                <span className="ec-pin-id">{row.pin}</span>
                <span className="ec-pin-signal">{row.signal}</span>
                <span className="ec-pin-mode">{row.mode}</span>
                <span className="ec-pin-af">{row.af}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ── CLOCK TAB ── */}
      {activeTab === "Clock" && (
        <div className="ec-content ec-clock-tab">
          <div className="ec-section-title">Clock Configuration Tree</div>
          <div className="ec-clock-tree">
            {[
              { src: "HSE (8 MHz)", arrow: "→", node: "PLL M=8, N=336, P=2", result: "168 MHz" },
              { src: "PLL Output", arrow: "→", node: "AHB Prescaler /2", result: "SYSCLK 84 MHz" },
              { src: "SYSCLK",    arrow: "→", node: "APB1 Prescaler /2", result: "APB1 42 MHz" },
              { src: "SYSCLK",    arrow: "→", node: "APB2 Prescaler /1", result: "APB2 84 MHz" },
              { src: "APB1",      arrow: "→", node: "Timer × 2",         result: "TIM CLK 84 MHz" },
              { src: "LSI",       arrow: "→", node: "IWDG Clock",         result: "32 kHz" },
            ].map((row, i) => (
              <div key={i} className="ec-clock-row">
                <span className="ec-clock-src">{row.src}</span>
                <span className="ec-clock-arrow">{row.arrow}</span>
                <span className="ec-clock-node">{row.node}</span>
                <span className="ec-clock-result">{row.result}</span>
              </div>
            ))}
          </div>
          <div className="ec-section-title" style={{ marginTop: 16 }}>Clock Source Configuration</div>
          {[
            { label: "HSE Frequency",  value: "8 MHz" },
            { label: "System Clock",   value: specs.speed },
            { label: "USB Clock",      value: "48 MHz (PLL Q=7)" },
            { label: "Core Voltage",   value: "1.2 V (Scale 1)" },
          ].map(row => (
            <div key={row.label} className="ec-param-row">
              <label className="ec-param-label">{row.label}</label>
              <input className="ec-param-input" defaultValue={row.value} />
            </div>
          ))}
        </div>
      )}

      {/* ── CONFIGURATION TAB ── */}
      {activeTab === "Configuration" && (
        <div className="ec-content ec-config-tab">
          {/* Left: Category tree */}
          <div className="ec-cat-col">
            <div className="ec-search-box">
              <Search size={11} className="ec-search-icon" />
              <input
                type="text"
                placeholder="Search peripherals..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                className="ec-search-input"
              />
              {searchQuery && (
                <button className="ec-search-clear" onClick={() => setSearchQuery("")}>
                  <X size={10} />
                </button>
              )}
            </div>

            <div className="ec-cat-list">
              {filteredCategories.map(cat => (
                <div key={cat.id} className="ec-cat-group">
                  <button
                    className="ec-cat-header"
                    onClick={() => toggleCategory(cat.id)}
                  >
                    <span className="ec-cat-icon">{cat.icon}</span>
                    <span className="ec-cat-label">{cat.label}</span>
                    {expandedCategories.has(cat.id)
                      ? <ChevronDown size={11} className="ec-cat-arrow" />
                      : <ChevronRight size={11} className="ec-cat-arrow" />}
                  </button>

                  {expandedCategories.has(cat.id) && (
                    <div className="ec-cat-children">
                      {cat.children.map(child => {
                        const cfg = peripheralConfigs[child.id];
                        const isEnabled = cfg?.enabled ?? false;
                        return (
                          <button
                            key={child.id}
                            className={`ec-child-item ${selectedPeripheral === child.id ? "ec-child-active" : ""}`}
                            onClick={() => setSelectedPeripheral(child.id)}
                          >
                            <span
                              className="ec-child-dot"
                              style={{ background: isEnabled ? "var(--accent-success)" : "var(--border-color)" }}
                            />
                            <span className="ec-child-label">{child.label}</span>
                            {isEnabled && <span className="ec-enabled-badge">ON</span>}
                          </button>
                        );
                      })}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Right: Detail panel + chip */}
          <div className="ec-detail-col">
            <div className="ec-chip-and-specs">
              <MCUPinChip board={selectedBoard} />
              <div className="ec-spec-grid">
                <div className="ec-spec-item"><span className="ec-spec-lbl">Flash</span><span className="ec-spec-val">{specs.flash}</span></div>
                <div className="ec-spec-item"><span className="ec-spec-lbl">RAM</span><span className="ec-spec-val">{specs.ram}</span></div>
                <div className="ec-spec-item"><span className="ec-spec-lbl">Speed</span><span className="ec-spec-val">{specs.speed}</span></div>
                <div className="ec-spec-item"><span className="ec-spec-lbl">Core</span><span className="ec-spec-val">{specs.core}</span></div>
              </div>
            </div>

            <PeripheralDetail
              peripheralId={selectedPeripheral}
              config={selectedConfig}
              onToggle={() => togglePeripheral(selectedPeripheral)}
              onParamChange={(key, val) => updateParam(selectedPeripheral, key, val)}
            />
          </div>
        </div>
      )}

      {/* ── PROJECT TAB ── */}
      {activeTab === "Project" && (
        <div className="ec-content ec-project-tab">
          <div className="ec-section-title">Project Settings</div>
          {[
            { label: "Project Name", value: "blinky-stm32f4" },
            { label: "Toolchain / IDE", value: "Makefile + arm-none-eabi-gcc" },
            { label: "Linker Script", value: "STM32F401RETX_FLASH.ld" },
            { label: "Heap Size", value: "0x200" },
            { label: "Stack Size", value: "0x400" },
            { label: "HAL Version", value: "STM32CubeF4 v1.27.1" },
          ].map(row => (
            <div key={row.label} className="ec-param-row">
              <label className="ec-param-label">{row.label}</label>
              <input className="ec-param-input" defaultValue={row.value} />
            </div>
          ))}

          <div className="ec-section-title" style={{ marginTop: 16 }}>Generated Files</div>
          {["Core/Src/main.c", "Core/Inc/main.h", "Core/Src/stm32f4xx_it.c", "Makefile"].map(f => (
            <div key={f} className="ec-file-row">
              <FileCode size={12} style={{ color: "var(--accent-violet-hover)" }} />
              <span>{f}</span>
            </div>
          ))}

          <button className="ec-generate-btn">
            <Zap size={13} />
            Generate Code
          </button>
        </div>
      )}
    </div>
  );
};

export default EmbeddedConfigurator;
