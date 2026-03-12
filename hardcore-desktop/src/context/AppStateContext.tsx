import React, { createContext, useContext, useState, ReactNode } from 'react';

interface ExecutionStep {
    step: number;
    action: string;
    description: string;
    pin?: number;
    value?: number;
    ms?: number;
}

interface Component {
    id: string;
    name: string;
    type: string;
    pins: string[];
    voltage: string;
    current?: string;
    purpose: string;
}

interface Connection {
    from: string;
    to: string;
    pin: number;
    color: string;
}

interface LearningData {
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

// Pin analysis result from the deterministic /analyze-pins endpoint
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

interface ArchitectureReport {
    overview: string;
    hardware: string;
    memory_storage: string;
    security: string;
    firmware_pipeline: string;
    wireless: string;
    compliance: string;
    bom: string;
    risks: string;
}

interface AppState {
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

    // UI state
    activeTab: 'architecture' | 'security' | 'compliance' | 'bom' | 'risk' | 'code';
}

interface AppContextType {
    state: AppState;
    setGeneratedCode: (code: string) => void;
    setGeneratedFiles: (files: Record<string, string>) => void;
    setComponents: (components: Component[]) => void;
    setConnections: (connections: Connection[]) => void;
    setPinJson: (pinJson: any) => void;
    setPinAnalysis: (analysis: PinAnalysis | null) => void;
    setArchitectureReport: (report: ArchitectureReport | null) => void;
    setActiveTab: (tab: any) => void;
    clearAll: () => void;
}

const initialState: AppState = {
    generatedCode: '',
    generatedFiles: {},
    components: [],
    connections: [],
    pinJson: null,
    architectureReport: null,
    pinAnalysis: null,
    activeTab: 'architecture',
};

const AppContext = createContext<AppContextType | undefined>(undefined);

export const AppStateProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
    const [state, setState] = useState<AppState>(initialState);

    const setGeneratedCode = (code: string) => setState(prev => ({ ...prev, generatedCode: code }));
    const setGeneratedFiles = (files: Record<string, string>) => setState(prev => ({ ...prev, generatedFiles: files }));
    const setComponents = (components: Component[]) => setState(prev => ({ ...prev, components }));
    const setConnections = (connections: Connection[]) => setState(prev => ({ ...prev, connections }));
    const setPinJson = (pinJson: any) => setState(prev => ({ ...prev, pinJson }));
    const setPinAnalysis = (pinAnalysis: PinAnalysis | null) => setState(prev => ({ ...prev, pinAnalysis }));
    const setArchitectureReport = (architectureReport: ArchitectureReport | null) => setState(prev => ({ ...prev, architectureReport }));
    const setActiveTab = (tab: any) => setState(prev => ({ ...prev, activeTab: tab }));
    const clearAll = () => setState(initialState);

    return (
        <AppContext.Provider
            value={{
                state,
                setGeneratedCode,
                setGeneratedFiles,
                setComponents,
                setConnections,
                setPinJson,
                setPinAnalysis,
                setArchitectureReport,
                setActiveTab,
                clearAll,
            }}
        >
            {children}
        </AppContext.Provider>
    );
};

export const useAppState = () => {
    const context = useContext(AppContext);
    if (!context) {
        throw new Error('useAppState must be used within AppStateProvider');
    }
    return context;
};
