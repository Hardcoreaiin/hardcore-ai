import { create } from "zustand";

export interface FileItem {
  name: string;
  path: string;
  isFolder: boolean;
  children?: FileItem[];
}

export interface RegisterItem {
  name: string;
  value: string;
  description: string;
  bits?: { name: string; value: number; range: string; description: string }[];
}

export interface ChatMessage {
  id: string;
  sender: "user" | "ai";
  text: string;
  timestamp: string;
}

export interface PlotDataPoint {
  time: string;
  temp: number;
  voltage: number;
  current: number;
}

interface WorkspaceState {
  // Project & Files
  activeFile: string | null;
  fileContents: { [path: string]: string };
  fileTree: FileItem[];
  
  // Compilation & Flashing
  isCompiling: boolean;
  isFlashing: boolean;
  buildLogs: string[];
  
  // GDB Debugging
  isDebugging: boolean;
  debuggerActive: boolean;
  currentLine: number | null;
  breakpoints: number[];
  callStack: string[];
  registers: RegisterItem[];
  crashed: boolean;
  crashReason: string | null;
  
  // Telemetry & Serial
  serialLogs: string[];
  serialConnected: boolean;
  activePort: string;
  baudRate: number;
  plotData: PlotDataPoint[];
  
  // AI Panel
  aiMessages: ChatMessage[];
  aiWaiting: boolean;
  
  // UI Tabs
  activeBottomTab: "terminal" | "plotter" | "registers" | "memory";
  
  // Welcome & Custom UI State
  showWelcomeScreen: boolean;
  activeSidebarTab: "explorer" | "search" | "git" | "debug" | "extensions" | "boards";
  selectedBoard: "STM32F401" | "ESP32-S3" | "RP2040";
  selectedProbe: "ST-Link V2" | "J-Link" | "CMSIS-DAP";
  toolchainPath: string;
  
  // Setters/Actions
  setActiveFile: (path: string | null) => void;
  updateFileContent: (path: string, content: string) => void;
  setCompiling: (val: boolean) => void;
  setFlashing: (val: boolean) => void;
  addBuildLog: (log: string) => void;
  clearBuildLogs: () => void;
  toggleBreakpoint: (line: number) => void;
  startDebugging: () => void;
  stopDebugging: () => void;
  toggleSerialConnection: () => void;
  addSerialLog: (log: string) => void;
  addPlotPoint: (pt: PlotDataPoint) => void;
  sendAiMessage: (text: string) => void;
  setBottomTab: (tab: "terminal" | "plotter" | "registers" | "memory") => void;
  triggerCrash: () => void;
  resolveCrash: () => void;
  stepOver: () => void;
  continueExecution: () => void;
  
  setShowWelcomeScreen: (val: boolean) => void;
  setActiveSidebarTab: (tab: "explorer" | "search" | "git" | "debug" | "extensions" | "boards") => void;
  setSelectedBoard: (board: "STM32F401" | "ESP32-S3" | "RP2040") => void;
  setSelectedProbe: (probe: "ST-Link V2" | "J-Link" | "CMSIS-DAP") => void;
  setToolchainPath: (path: string) => void;
}

const mockFiles: FileItem[] = [
  {
    name: "src",
    path: "/src",
    isFolder: true,
    children: [
      { name: "main.c", path: "/src/main.c", isFolder: false },
      { name: "stm32f4xx_it.c", path: "/src/stm32f4xx_it.c", isFolder: false },
      { name: "system_stm32f4xx.c", path: "/src/system_stm32f4xx.c", isFolder: false }
    ]
  },
  {
    name: "include",
    path: "/include",
    isFolder: true,
    children: [
      { name: "main.h", path: "/include/main.h", isFolder: false },
      { name: "stm32f4xx_it.h", path: "/include/stm32f4xx_it.h", isFolder: false }
    ]
  },
  {
    name: "CMakeLists.txt",
    path: "/CMakeLists.txt",
    isFolder: false
  },
  {
    name: "stm32f401.ld",
    path: "/stm32f401.ld",
    isFolder: false
  }
];

const mockMainC = `// HARDCOREAI: Blinky Firmware for STM32F401RET6
#include "main.h"

/* Private variables ---------------------------------------------------------*/
GPIO_InitTypeDef GPIO_InitStruct = {0};

/* Private function prototypes -----------------------------------------------*/
void SystemClock_Config(void);
static void MX_GPIO_Init(void);

/**
  * @brief  The application entry point.
  * @retval int
  */
// CPU configuration registers, SVD mapping initialized
// target: STM32F401RETx
// debugger: ST-LINK/V2 (SWD interface)
// clocks: HSE osc crystal at 8 MHz
// system frequency: 84 MHz
//

int main(void)
{
  HAL_Init();
  SystemClock_Config();
  MX_GPIO_Init();
  while (1)
  {
    HAL_GPIO_TogglePin(GPIOA, GPIO_PIN_5);
    HAL_Delay(500);
  }
}

/**
 * @brief System Clock Configuration
 * @retval None
 */
void SystemClock_Config(void)
{
  RCC_OscInitTypeDef RCC_OscInitStruct = {0};
  RCC_ClkInitTypeDef RCC_ClkInitStruct = {0};
  
  // Configure the main internal regulator output voltage */
  __HAL_RCC_PWR_CLK_ENABLE();
  __HAL_PWR_VOLTAGESCALING_CONFIG(PWR_REGULATOR_VOLTAGE_SCALE1);
  
  // Initializes the CPU, AHB and APB buses clocks */
  RCC_OscInitStruct.OscillatorType = RCC_OscInitStruct.OscillatorType = RCC_OSCILLATORTYPE_HSE;
  RCC_OscInitStruct.HSEState = RCC_HSE_ON;
  RCC_OscInitStruct.PLL.PLLState = RCC_PLL_ON;
  RCC_OscInitStruct.PLL.PLLSource = RCC_PLLSOURCE_HSE;
}`;

const mockItC = `#include "main.h"
#include "stm32f4xx_it.h"

void NMI_Handler(void) {
}

void HardFault_Handler(void) {
  // Capture crash register values
  printf("!!! HARD_FAULT_INTERRUPT !!!\\r\\n");
  while (1) {
  }
}
`;

export const useWorkspaceStore = create<WorkspaceState>((set) => ({
  activeFile: "/src/main.c",
  fileContents: {
    "/src/main.c": mockMainC,
    "/src/stm32f4xx_it.c": mockItC,
    "/CMakeLists.txt": "cmake_minimum_required(VERSION 3.16)\nproject(hardcoreai_app C)\n\nset(CMAKE_C_STANDARD 11)\nadd_executable(hardcoreai_app src/main.c src/stm32f4xx_it.c)",
    "/stm32f401.ld": "/* Linker Script for STM32F401 */\nMEMORY {\n  FLASH (rx) : ORIGIN = 0x08000000, LENGTH = 256K\n  RAM (xrw)  : ORIGIN = 0x20000000, LENGTH = 64K\n}"
  },
  fileTree: mockFiles,
  isCompiling: false,
  isFlashing: false,
  buildLogs: [
    "HARDCOREAI Build Engine v1.0.0",
    "Initializing CMake project configuration...",
    "Found toolchain: arm-none-eabi-gcc 12.3.1",
    "Ready to build project."
  ],
  isDebugging: false,
  debuggerActive: false,
  currentLine: null,
  breakpoints: [24],
  callStack: ["main() at main.c:20", "Reset_Handler() at startup_stm32f401.s:55"],
  
  // Welcome & Custom UI State
  showWelcomeScreen: false,
  activeSidebarTab: "explorer",
  selectedBoard: "STM32F401",
  selectedProbe: "ST-Link V2",
  toolchainPath: "/usr/bin/arm-none-eabi-gcc",
  registers: [
    {
      name: "GPIOA",
      value: "0x40020000",
      description: "General-Purpose I/O Port A",
      bits: [
        { name: "MODER", value: 0x28000280, range: "31:0", description: "GPIO port mode register" },
        { name: "OTYPER", value: 0x00000000, range: "15:0", description: "GPIO port output type register" },
        { name: "ODR", value: 0x00000020, range: "15:0", description: "GPIO port output data register" }
      ]
    },
    {
      name: "ADC1",
      value: "0x40012000",
      description: "Analog-to-Digital Converter 1",
      bits: [
        { name: "SR", value: 0x00000002, range: "5:0", description: "ADC status register" },
        { name: "CR1", value: 0x00000100, range: "25:0", description: "ADC control register 1" },
        { name: "DR", value: 0x00000A23, range: "11:0", description: "ADC regular data register" }
      ]
    },
    {
      name: "Core Registers",
      value: "CPU Core",
      description: "ARM Cortex-M4 Core Registers",
      bits: [
        { name: "R0", value: 0x00000000, range: "32b", description: "Argument / result register" },
        { name: "R1", value: 0x20000400, range: "32b", description: "General purpose register" },
        { name: "PC", value: 0x080010AC, range: "32b", description: "Program Counter" },
        { name: "LR", value: 0x080012A3, range: "32b", description: "Link Register (return address)" },
        { name: "SP", value: 0x2000FFC0, range: "32b", description: "Stack Pointer" }
      ]
    }
  ],
  crashed: false,
  crashReason: null,
  serialLogs: [
    "[12:41:10.123] System Booting...",
    "[12:41:10.456] MCU: STM32F401RETx",
    "[12:41:10.457] Clock: 84 MHz",
    "[12:41:10.458] Flash: 512 KB | RAM: 96 KB",
    "[12:41:10.459] Hello from HardcoreAI IDE! 🚀"
  ],
  serialConnected: true,
  activePort: "COM4 (ST-Link Virtual Port)",
  baudRate: 115200,
  plotData: [
    { time: "00:01", temp: 24.5, voltage: 3.3, current: 42.1 },
    { time: "00:02", temp: 25.1, voltage: 3.3, current: 42.2 },
    { time: "00:03", temp: 26.3, voltage: 3.28, current: 44.5 },
    { time: "00:04", temp: 27.2, voltage: 3.29, current: 43.1 }
  ],
  aiMessages: [
    {
      id: "1",
      sender: "ai",
      text: "Hello! I am your HARDCOREAI Copilot. I have loaded context for the **STM32F401RET6** target, SVD registers, and your current `CMake` configuration. \n\nHow can I help you write or debug firmware today?",
      timestamp: "21:52"
    }
  ],
  aiWaiting: false,
  activeBottomTab: "terminal",

  setActiveFile: (path) => set({ activeFile: path }),
  updateFileContent: (path, content) =>
    set((state) => ({
      fileContents: { ...state.fileContents, [path]: content }
    })),
  setCompiling: (val) => set({ isCompiling: val }),
  setFlashing: (val) => set({ isFlashing: val }),
  addBuildLog: (log) => set((state) => ({ buildLogs: [...state.buildLogs, log] })),
  clearBuildLogs: () => set({ buildLogs: [] }),
  toggleBreakpoint: (line) =>
    set((state) => ({
      breakpoints: state.breakpoints.includes(line)
        ? state.breakpoints.filter((l) => l !== line)
        : [...state.breakpoints, line]
    })),
  startDebugging: () =>
    set({
      isDebugging: true,
      debuggerActive: true,
      currentLine: 20,
      activeBottomTab: "registers"
    }),
  stopDebugging: () =>
    set({
      isDebugging: false,
      debuggerActive: false,
      currentLine: null,
      crashed: false,
      crashReason: null
    }),
  toggleSerialConnection: () =>
    set((state) => ({ serialConnected: !state.serialConnected })),
  addSerialLog: (log) => set((state) => ({ serialLogs: [...state.serialLogs, log] })),
  addPlotPoint: (pt) => set((state) => ({ plotData: [...state.plotData, pt] })),
  sendAiMessage: (text) =>
    set((state) => {
      const newMsgs = [
        ...state.aiMessages,
        { id: Math.random().toString(), sender: "user" as const, text, timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }
      ];
      
      // Setup response simulation
      setTimeout(() => {
        let aiResponse = "I am scanning your workspace...";
        if (text.toLowerCase().includes("crash") || text.toLowerCase().includes("fault")) {
          aiResponse = "Analyzing crash dump. The register status reports `CFSR = 0x00008200` which corresponds to a **Precise Data Bus Error**. You are attempting to write to the memory address `0x00000000` (which is stored in `R0` as R0=0x00000000). To resolve this, ensure you initialize the pointer variables before assigning values, e.g.: \n```c\n// Fix: allocate or map crash_trigger\nstatic uint32_t buffer_data;\ncrash_trigger = &buffer_data;\n*crash_trigger = 0xDEADC0DE;\n```";
        } else if (text.toLowerCase().includes("gpio") || text.toLowerCase().includes("led")) {
          aiResponse = "To configure a GPIO pin as output on Port A Pin 5, you can use the HAL Library:\n```c\nGPIO_InitTypeDef GPIO_InitStruct = {0};\n__HAL_RCC_GPIOA_CLK_ENABLE();\n\nGPIO_InitStruct.Pin = GPIO_PIN_5;\nGPIO_InitStruct.Mode = GPIO_MODE_OUTPUT_PP;\nGPIO_InitStruct.Pull = GPIO_NOPULL;\nGPIO_InitStruct.Speed = GPIO_SPEED_FREQ_LOW;\nHAL_GPIO_Init(GPIOA, &GPIO_InitStruct);\n```";
        } else {
          aiResponse = "I've reviewed your active target files. Your peripheral initialization MX_ADC1_Init() is structured properly. Let me know if you would like me to generate serial plotting formats for it.";
        }
        
        useWorkspaceStore.setState((state) => ({
          aiMessages: [
            ...state.aiMessages,
            { id: Math.random().toString(), sender: "ai", text: aiResponse, timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }
          ],
          aiWaiting: false
        }));
      }, 1000);

      return {
        aiMessages: newMsgs,
        aiWaiting: true
      };
    }),
  setBottomTab: (tab) => set({ activeBottomTab: tab }),
  triggerCrash: () =>
    set((state) => {
      // Create detailed debug registers for crash state
      const crashRegs = state.registers.map((reg) => {
        if (reg.name === "Core Registers") {
          return {
            ...reg,
            bits: reg.bits?.map((bit) => {
              if (bit.name === "PC") return { ...bit, value: 0x08001A4E }; // Point of crash
              if (bit.name === "R0") return { ...bit, value: 0x00000000 }; // NULL pointer
              return bit;
            })
          };
        }
        return reg;
      });

      return {
        crashed: true,
        crashReason: "HardFault: Precise Data Bus Error (Dereferencing NULL pointer)",
        currentLine: 45, // Points to line *crash_trigger = 0xDEADC0DE
        registers: crashRegs,
        activeBottomTab: "registers"
      };
    }),
  resolveCrash: () =>
    set((state) => {
      const fixedMainC = state.fileContents["/src/main.c"].replace(
        "uint32_t *crash_trigger = NULL;",
        "static uint32_t val_holder = 0;\n  uint32_t *crash_trigger = &val_holder;"
      );
      
      return {
        crashed: false,
        crashReason: null,
        currentLine: 20,
        fileContents: {
          ...state.fileContents,
          "/src/main.c": fixedMainC
        }
      };
    }),
  stepOver: () =>
    set((state) => {
      if (state.currentLine === null) return {};
      let nextLine = state.currentLine + 1;
      if (nextLine > 50) nextLine = 20; // Loop simulation
      return { currentLine: nextLine };
    }),
  continueExecution: () =>
    set({
      currentLine: null // Running state
    }),
  setShowWelcomeScreen: (val) => set({ showWelcomeScreen: val }),
  setActiveSidebarTab: (tab) => set({ activeSidebarTab: tab }),
  setSelectedBoard: (board) => set({ selectedBoard: board }),
  setSelectedProbe: (probe) => set({ selectedProbe: probe }),
  setToolchainPath: (path) => set({ toolchainPath: path })
}));
