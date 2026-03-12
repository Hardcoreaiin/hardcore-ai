import React, { useState, useEffect, useCallback } from 'react';
import { Panel, PanelGroup, PanelResizeHandle } from 'react-resizable-panels';
import { useBoard } from './context/BoardContext';
import { useAuth } from './context/AuthContext';
import { useAppState } from './context/AppStateContext';
import TopBar from './components/TopBar/TopBar';
import AIAssistant from './components/AIAssistant/AIAssistant';
import BottomPanel from './components/BottomPanel/BottomPanel';
import ArchitectureHub from './components/Architecture/ArchitectureHub';
import LoginScreen from './components/Auth/LoginScreen';
import MonacoEditor from './components/Editor/MonacoEditor';
import FileExplorer from './components/Explorer/FileExplorer';
import { Layout, ShieldAlert, Landmark, ShoppingCart, AlertTriangle, Code } from 'lucide-react';
import { motion } from 'framer-motion';

interface OpenFile {
    path: string;
    name: string;
    content: string;
    isDirty: boolean;
}

function App() {
    const { state, setGeneratedCode, setActiveTab } = useAppState();
    const { selectedBoard, setSelectedBoard, detectedBoard, connectedPort, setConnectedPort } = useBoard();
    const { isAuthenticated, isLoading: authLoading } = useAuth();
    const [openFiles, setOpenFiles] = useState<OpenFile[]>([]);
    const [activeFilePath, setActiveFilePath] = useState<string | null>(null);
    const [peripheralCounts, setPeripheralCounts] = useState({ gpio: 0, i2c: 0, spi: 0, uart: 0, timers: 0 });

    const activeFile = openFiles.find(f => f.path === activeFilePath);

    // Sync from Global State
    useEffect(() => {
        if (Object.keys(state.generatedFiles).length > 0) {
            setOpenFiles(prev => {
                let newFiles = [...prev];
                let changed = false;

                Object.entries(state.generatedFiles).forEach(([name, content]) => {
                    const path = name.startsWith('/') ? name : `/src/${name}`;
                    const existing = newFiles.find(f => f.path === path);

                    if (existing) {
                        if (existing.content !== content) {
                            newFiles = newFiles.map(f => f.path === path ? { ...f, content, isDirty: false } : f);
                            changed = true;
                        }
                    } else {
                        newFiles.push({ path, name, content, isDirty: false });
                        changed = true;
                    }
                });

                return changed ? newFiles : prev;
            });

            // Set active file to main.cpp if it was just generated
            if (state.generatedFiles['main.cpp']) {
                setActiveFilePath('/src/main.cpp');
            } else if (Object.keys(state.generatedFiles).length > 0 && !activeFilePath) {
                const firstFile = Object.keys(state.generatedFiles)[0];
                setActiveFilePath(firstFile.startsWith('/') ? firstFile : `/src/${firstFile}`);
            }
        } else if (state.generatedCode && !openFiles.find(f => f.content === state.generatedCode)) {
            const filePath = '/src/main.cpp';
            setOpenFiles(prev => {
                const existing = prev.find(f => f.path === filePath);
                if (existing) return prev.map(f => f.path === filePath ? { ...f, content: state.generatedCode, isDirty: false } : f);
                return [...prev, { path: filePath, name: 'main.cpp', content: state.generatedCode, isDirty: false }];
            });
            setActiveFilePath(filePath);
        }
    }, [state.generatedFiles, state.generatedCode]);

    useEffect(() => {
        const handler = (e: CustomEvent) => setPeripheralCounts(e.detail);
        window.addEventListener('peripheral-update', handler as EventListener);
        return () => window.removeEventListener('peripheral-update', handler as EventListener);
    }, []);

    const handleFileClick = useCallback((path: string, content: string) => {
        const name = path.split('/').pop() || 'unknown';
        setOpenFiles(prev => prev.find(f => f.path === path) ? prev : [...prev, { path, name, content, isDirty: false }]);
        setActiveFilePath(path);
        // If we click a file, we probably want to see the code tab if it wasn't already
        if (state.activeTab !== 'code') {
            setActiveTab('code');
        }
    }, [state.activeTab, setActiveTab]);

    const handleEditorChange = useCallback((value: string | undefined) => {
        if (!value || !activeFilePath) return;
        setOpenFiles(prev => prev.map(f => f.path === activeFilePath ? { ...f, content: value, isDirty: true } : f));
        if (activeFilePath === '/src/main.cpp') {
            setGeneratedCode(value);
        }
    }, [activeFilePath, setGeneratedCode]);

    const handleTabClick = useCallback((path: string) => setActiveFilePath(path), []);
    const handleTabClose = useCallback((path: string) => {
        setOpenFiles(prev => {
            const filtered = prev.filter(f => f.path !== path);
            if (path === activeFilePath) setActiveFilePath(filtered.length ? filtered[filtered.length - 1].path : null);
            return filtered;
        });
    }, [activeFilePath]);

    useEffect(() => {
        if (openFiles.length === 0) {
            const defaultPath = '/src/main.cpp';
            const defaultContent = '// Engineering Project Initialized\n// Target Architecture Design Mode\n\nvoid setup() {\n    // System configuration\n}\n\nvoid loop() {\n    // Main process\n}\n';
            setOpenFiles([{ path: defaultPath, name: 'main.cpp', content: defaultContent, isDirty: false }]);
            setActiveFilePath(defaultPath);
        }
    }, []);

    const tabs = [
        { id: 'architecture', label: 'Overview', icon: <Layout className="w-4 h-4" /> },
        { id: 'security', label: 'Security', icon: <ShieldAlert className="w-4 h-4" /> },
        { id: 'compliance', label: 'Compliance', icon: <Landmark className="w-4 h-4" /> },
        { id: 'bom', label: 'BOM', icon: <ShoppingCart className="w-4 h-4" /> },
        { id: 'risk', label: 'Risk Analysis', icon: <AlertTriangle className="w-4 h-4" /> },
        { id: 'code', label: 'Firmware', icon: <Code className="w-4 h-4" /> },
    ];

    if (authLoading) return <div className="h-screen w-screen flex items-center justify-center bg-black text-neutral-600">Loading...</div>;
    if (!isAuthenticated) return <LoginScreen />;

    return (
        <div className="h-screen w-screen relative flex flex-col overflow-hidden bg-black text-neutral-300 font-sans">
            <div className="relative z-10 flex flex-col h-full w-full">
                <TopBar
                    onPortChange={setConnectedPort}
                    onBoardChange={setSelectedBoard}
                    currentCode={activeFile?.content}
                />

                <PanelGroup direction="horizontal" className="flex-1 p-2 gap-2">
                    {/* Left Panel: AIAssistant - Industrial Sidecar */}
                    <Panel defaultSize={20} minSize={15} maxSize={30} className="glass-panel rounded-2xl overflow-hidden shadow-2xl border border-white/5">
                        <AIAssistant />
                    </Panel>

                    <PanelResizeHandle className="w-1 rounded-full bg-transparent hover:bg-blue-600/20 transition-colors" />

                    {/* Industrial Architecture Area */}
                    <Panel defaultSize={80} minSize={50}>
                        <PanelGroup direction="vertical" className="gap-2">
                            {/* Unified Architecture Hub (Top) */}
                            <Panel defaultSize={70} minSize={30} className="glass-panel rounded-2xl overflow-hidden shadow-xl border border-white/5 flex flex-col">

                                {/* Professional Persistent Tab Bar */}
                                <div className="flex bg-white/5 border-b border-white/10 p-1 px-4 gap-1 overflow-x-auto scrollbar-hide shrink-0">
                                    {tabs.map(t => (
                                        <button
                                            key={t.id}
                                            onClick={() => setActiveTab(t.id)}
                                            className={`group relative flex items-center gap-2 px-5 py-4 transition-all whitespace-nowrap ${state.activeTab === t.id ? 'text-white' : 'text-neutral-500 hover:text-neutral-300'
                                                }`}
                                        >
                                            <div className={`p-1.5 rounded-lg transition-all ${state.activeTab === t.id ? 'bg-white/10 text-blue-400 shadow-[0_0_15px_rgba(96,165,250,0.1)]' : 'bg-transparent'}`}>
                                                {t.icon}
                                            </div>
                                            <span className="text-[11px] font-black uppercase tracking-[0.15em]">{t.label}</span>
                                            {state.activeTab === t.id && (
                                                <motion.div
                                                    layoutId="hubActiveTab"
                                                    className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500 shadow-[0_0_15px_rgba(59,130,246,0.5)]"
                                                />
                                            )}
                                        </button>
                                    ))}
                                </div>

                                <div className="flex-1 min-h-0">
                                    {state.activeTab === 'code' ? (
                                        <PanelGroup direction="horizontal">
                                            {/* File Explorer Sidebar */}
                                            <Panel defaultSize={20} minSize={10} maxSize={30}>
                                                <FileExplorer
                                                    files={openFiles}
                                                    onFileClick={handleFileClick}
                                                    activePath={activeFilePath}
                                                />
                                            </Panel>

                                            <PanelResizeHandle className="w-1 bg-transparent hover:bg-blue-600/20 transition-colors" />

                                            {/* Main Content Area: Editor */}
                                            <Panel defaultSize={80} className="flex flex-col">
                                                <div className="flex bg-neutral-900 border-b border-white/5 overflow-x-auto scrollbar-hide">
                                                    {openFiles.map(f => (
                                                        <div
                                                            key={f.path}
                                                            onClick={() => setActiveFilePath(f.path)}
                                                            className={`px-4 py-2 text-[10px] font-black uppercase tracking-widest cursor-pointer border-r border-white/5 transition-all whitespace-nowrap ${activeFilePath === f.path
                                                                ? 'bg-blue-600/10 text-blue-400 border-b-2 border-blue-500'
                                                                : 'text-neutral-500 hover:bg-white/5'
                                                                }`}
                                                        >
                                                            {f.name}
                                                        </div>
                                                    ))}
                                                </div>
                                                <div className="flex-1">
                                                    <MonacoEditor
                                                        value={activeFile?.content || ''}
                                                        onChange={handleEditorChange}
                                                        language={activeFilePath?.endsWith('.md') ? 'markdown' : (activeFilePath?.endsWith('.h') ? 'cpp' : 'cpp')}
                                                    />
                                                </div>
                                            </Panel>
                                        </PanelGroup>
                                    ) : (
                                        <ArchitectureHub />
                                    )}
                                </div>
                            </Panel>

                            <PanelResizeHandle className="h-1 rounded-full bg-transparent hover:bg-blue-600/20 transition-colors" />

                            {/* Engineering Console (Bottom) */}
                            <Panel defaultSize={30} minSize={10} className="glass-panel rounded-2xl overflow-hidden shadow-lg border border-white/5">
                                <BottomPanel
                                    selectedBoard={selectedBoard || 'esp32'}
                                    detectedBoard={detectedBoard}
                                    connectedPort={connectedPort}
                                />
                            </Panel>
                        </PanelGroup>
                    </Panel>
                </PanelGroup>
            </div>
        </div>
    );
}

export default App;
