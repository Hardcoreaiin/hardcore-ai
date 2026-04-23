import { create } from 'zustand';

export interface LearningData {
    concept?: string;
    theory?: string;
    code_explanation?: string;
    hardware_explanation?: string;
    gpio_logic?: string;
    execution_flow?: string;
    safety_notes?: string;
    common_mistakes?: string;
    next_step?: string;
}
export interface Component {
    id: string;
    name: string;
    type: string;
    pins: string[];
    voltage: string;
    current?: string;
    purpose: string;
}

export interface Connection {
    from: string;
    to: string;
    pin: number;
    color: string;
}

export interface PinInfo {
    gpio: number;
    name: string;
    mode: string;
    component: string;
    functions_used: string[];
    type: string;
    all_functions: string[];
    warnings: string[];
    safe_for_output: boolean;
    safe_for_input: boolean;
}

export interface PinAnalysis {
    board: string;
    summary: string;
    pins: PinInfo[];
    connections: Array<{
        from: string;
        to: string;
        gpio: number;
        mode: string;
        color: string;
        label: string;
        functions: string[];
        var_names: string[];
    }>;
    protocols: Record<string, any>;
    warnings: string[];
    constants: Record<string, number>;
    diagram: any;
}

export interface InternalSpec {
    system_type: string;
    power: string;
    budget: string;
    performance: string;
    environment: string;
    connectivity: string[];
    compute: string;
    constraints: Record<string, any>;
}

// Updated ArchitectureReport for Multi-Mode Engineering
export interface ArchitectureReport {
    mode: string;
    overview: string;
    hardware_blocks?: Array<{ name: string; function: string; details: string }>;
    interfaces?: Array<{ bus: string; component: string; speed: string; notes: string }>;
    power_architecture?: { pmic: string; rails: any[]; design_notes: string };
    boot_chain?: Array<{ name: string; description: string; security: string }>;
    memory_architecture?: { type: string; bus_width: string; routing_constraints: string; explanation: string };
    security?: { secure_enclave: string; cryptography: string; features: string[]; explanation: string };
    linux_stack?: { 
        linux_distribution: string; 
        bootloader: string; 
        kernel_version: string; 
        device_tree: string; 
        drivers: string[]; 
        ota_framework: string; 
        security_features: string[];
        filesystem?: string;
        security_modules?: string;
    };
    bootloader?: {
        type: string;
        config_file: string;
        config_content?: string;
        secure_boot?: string;
        verification_commands?: string[];
    };
    device_tree?: { file: string; content: string };
    firmware_stack?: { os: string; bootloader: string; drivers: string[]; ota: string; explanation: string };
    compliance?: Array<{ standard: string; description: string }>;
    risk_analysis?: Array<{ category: string; severity: number; description: string }>;
    bom?: Array<{ part_number: string; manufacturer: string; function: string; price_estimate: string }>;
    project_structure?: { name: string; type: 'file' | 'directory'; children?: any[] };
    files?: Record<string, string>;
    diagrams?: { hardware_block?: string; boot_chain?: string };
}

interface AppState {
    // Mode
    currentMode: 'FIRMWARE_MODE' | 'EMBEDDED_DESIGN_MODE' | 'SYSTEM_ARCHITECTURE_MODE' | null;
    
    // Code and files (MPU/Industrial firmware)
    generatedCode: string;
    generatedFiles: Record<string, string>;

    // Hardware data (Industrial focus)
    components: Component[];
    connections: Connection[];
    pinJson: any;

    // Architecture Report (V2 Core)
    architectureReport: ArchitectureReport | null;

    // Pin analysis (Still useful for professional hardware)
    pinAnalysis: PinAnalysis | null;
    learningData?: LearningData | string | null;

    // Legacy simulation / learning state (required by simulation and learning components)
    executionSteps: any[];
    diagram: any;
    simulationLogic: any;
    theory: any;

    // UI state
    activeTab: 'architecture' | 'firmware' | 'output' | 'debug' | 'logs' | 'hardfault';
    
    // Multi-Stage Pipeline state
    pipelineStep: 'INTENT' | 'CLARIFICATION' | 'ARCHITECTURE' | 'BOM' | 'FIRMWARE' | 'TWIN';
    internalSpec: InternalSpec | null;
    
    // Actions
    setMode: (mode: 'FIRMWARE_MODE' | 'EMBEDDED_DESIGN_MODE' | 'SYSTEM_ARCHITECTURE_MODE' | null) => void;
    setGeneratedCode: (code: string) => void;
    setGeneratedFiles: (files: Record<string, string>) => void;
    setComponents: (components: Component[]) => void;
    setConnections: (connections: Connection[]) => void;
    setPinJson: (pinJson: any) => void;
    setArchitectureReport: (report: ArchitectureReport | null) => void;
    setPinAnalysis: (analysis: PinAnalysis | null) => void;
    setLearningData: (data: LearningData | string | null) => void;
    setExecutionSteps: (steps: any[]) => void;
    setDiagram: (diagram: any) => void;
    setSimulationLogic: (logic: any) => void;
    setTheory: (theory: any) => void;
    setActiveTab: (tab: 'architecture' | 'firmware' | 'output' | 'debug' | 'logs' | 'hardfault') => void;
    setPipelineStep: (step: 'INTENT' | 'CLARIFICATION' | 'ARCHITECTURE' | 'BOM' | 'FIRMWARE' | 'TWIN') => void;
    setInternalSpec: (spec: InternalSpec | null) => void;
    clearAll: () => void;
}

export const useAppStore = create<AppState>((set) => ({
    // Initial State
    currentMode: null,
    generatedCode: '',
    generatedFiles: {},
    components: [],
    connections: [],
    pinJson: null,
    architectureReport: null,
    pinAnalysis: null,
    learningData: null,
    executionSteps: [],
    diagram: null,
    simulationLogic: null,
    theory: null,
    activeTab: 'architecture',
    pipelineStep: 'INTENT',
    internalSpec: null,

    // Actions
    setMode: (mode) => set({ currentMode: mode }),
    setGeneratedCode: (code) => set({ generatedCode: code }),
    setGeneratedFiles: (files) => set({ generatedFiles: files }),
    setComponents: (components) => set({ components }),
    setConnections: (connections) => set({ connections }),
    setPinJson: (pinJson) => set({ pinJson }),
    setArchitectureReport: (report) => set({ architectureReport: report }),
    setPinAnalysis: (analysis) => set({ pinAnalysis: analysis }),
    setLearningData: (data) => set({ learningData: data }),
    setExecutionSteps: (steps) => set({ executionSteps: steps }),
    setDiagram: (diagram) => set({ diagram }),
    setSimulationLogic: (logic) => set({ simulationLogic: logic }),
    setTheory: (theory) => set({ theory }),
    setActiveTab: (tab) => set({ activeTab: tab }),
    setPipelineStep: (step) => set({ pipelineStep: step }),
    setInternalSpec: (spec) => set({ internalSpec: spec }),
    clearAll: () => set({
        currentMode: null,
        generatedCode: '',
        generatedFiles: {},
        components: [],
        connections: [],
        pinJson: null,
        architectureReport: null,
        pinAnalysis: null,
        learningData: null,
        executionSteps: [],
        diagram: null,
        simulationLogic: null,
        theory: null,
        activeTab: 'architecture',
        pipelineStep: 'INTENT',
        internalSpec: null
    }),
}));
