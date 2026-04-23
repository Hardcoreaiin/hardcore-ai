import React, { useState, useRef, useEffect } from 'react';
import { Send, Loader2, Sparkles, Zap } from 'lucide-react';
import { useAppStore } from '../../store/useAppStore';
import { useBoard } from '../../context/BoardContext';
import { motion, AnimatePresence } from 'framer-motion';

// Desktop bypass header for local authentication
const API_HEADERS = {
    'Content-Type': 'application/json',
    'X-Desktop-Key': 'desktop_local_bypass_hardcore_ai',
};

interface Message { id: string; role: 'user' | 'assistant'; content: string; type?: 'text' | 'teaching' | 'code'; }

const AIAssistant: React.FC = () => {
    const { selectedBoard, detectedBoard } = useBoard();
    const [messages, setMessages] = useState<Message[]>([{
        id: '1',
        role: 'assistant',
        content: 'System ready. Describe an embedded system to generate architecture and firmware (e.g., "Build a secure i.MX93 dual-core system with WiFi 6 and OTA updates").'
    }]);
    const [input, setInput] = useState('');
    const [loading, setLoading] = useState(false);
    const [loadingMessage, setLoadingMessage] = useState('Designing Circuit...');
    const endRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    const statusIntervalRef = useRef<NodeJS.Timeout | null>(null);

    // Use global state
    const setGeneratedCode = useAppStore(state => state.setGeneratedCode);
    const setGeneratedFiles = useAppStore(state => state.setGeneratedFiles);
    const setComponents = useAppStore(state => state.setComponents);
    const setConnections = useAppStore(state => state.setConnections);
    const setPinJson = useAppStore(state => state.setPinJson);
    const setArchitectureReport = useAppStore(state => state.setArchitectureReport);
    const setActiveTab = useAppStore(state => state.setActiveTab);
    const setMode = useAppStore(state => state.setMode);
    
    // Multi-stage pipeline state
    const pipelineStep = useAppStore(state => state.pipelineStep);
    const setPipelineStep = useAppStore(state => state.setPipelineStep);
    const setInternalSpec = useAppStore(state => state.setInternalSpec);

    useEffect(() => { endRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [messages]);

    const send = async (msgOverride?: string) => {
        const userMsg = msgOverride || input.trim();
        if (!userMsg || loading) return;

        setMessages(prev => [...prev, { id: `u${Date.now()}`, role: 'user', content: userMsg }]);
        setInput('');
        setLoading(true);
        setLoadingMessage('Initializing AI Architect...');

        statusIntervalRef.current = setInterval(async () => {
            try {
                const sRes = await fetch('http://localhost:8000/chat/status', { headers: API_HEADERS });
                if (sRes.ok) {
                    const sData = await sRes.json();
                    if (sData.message) setLoadingMessage(sData.message);
                }
            } catch (e) {
                // Ignore polling errors
            }
        }, 1500);

        try {
            const res = await fetch('http://localhost:8000/chat', {
                method: 'POST',
                headers: API_HEADERS,
                body: JSON.stringify({
                    message: userMsg,
                    history: messages.map(m => ({ role: m.role, content: m.content })),
                    board_type: selectedBoard || 'esp32dev',
                    detected_board: detectedBoard || null,
                    pipeline_step: pipelineStep
                }),
            });
            const data = await res.json();

            if (data.response_type === 'architecture' || data.response_type === 'code') {
                const report = data.report;
                if (report) {
                    setArchitectureReport(report);
                    if (report.mode) setMode(report.mode);
                    if (report.internal_spec) setInternalSpec(report.internal_spec);
                    
                    // Logic to increment pipeline step based on AI output
                    if (report.pipeline_step) {
                        if (report.pipeline_step === 'ARCHITECTURE') setPipelineStep('BOM');
                        else if (report.pipeline_step === 'BOM') setPipelineStep('FIRMWARE');
                        else if (report.pipeline_step === 'FIRMWARE') setPipelineStep('TWIN');
                    }
                }

                setMessages(prev => [...prev, {
                    id: `a${Date.now()}`,
                    role: 'assistant',
                    content: data.message || '✅ Engineering stage completed.'
                }]);

                if (data.response_type === 'code') {
                    if (data.firmware) setGeneratedCode(data.firmware);
                    if (data.files) setGeneratedFiles(data.files);
                    setActiveTab('firmware');
                } else {
                    setActiveTab('architecture');
                }
            } else {
                setMessages(prev => [...prev, {
                    id: `a${Date.now()}`,
                    role: 'assistant',
                    content: data.message || (data.questions ? data.questions.join('\n\n') : 'Understood. Proceeding...')
                }]);
                
                if (pipelineStep === 'INTENT') setPipelineStep('CLARIFICATION');
            }
        } catch (e) {
            setMessages(prev => [...prev, {
                id: `a${Date.now()}`,
                role: 'assistant',
                content: 'Failed to connect to backend. Make sure the server is running on port 8000.'
            }]);
        } finally {
            setLoading(false);
            if (statusIntervalRef.current) {
                clearInterval(statusIntervalRef.current);
                statusIntervalRef.current = null;
            }
        }
    };

    return (
        <div className="h-full flex flex-col bg-transparent relative overflow-hidden">
            {/* AI Assistant Header */}
            <div className="p-5 border-b border-white/5 bg-white/5 backdrop-blur-md flex items-center justify-between sticky top-0 z-20">
                <div className="flex items-center gap-4">
                    <div className="w-10 h-10 rounded-xl bg-blue-600/20 flex items-center justify-center border border-blue-500/30 shadow-[0_0_15px_rgba(59,130,246,0.2)]">
                        <Sparkles className="w-5 h-5 text-blue-400" />
                    </div>
                    <div>
                        <h2 className="text-[15px] font-black text-white tracking-tight">Active Architect</h2>
                        <div className="flex items-center gap-1.5">
                            <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse shadow-[0_0_8px_#22c55e]" />
                            <span className="text-[10px] text-neutral-500 uppercase font-black tracking-[0.2em]">Ready to Build</span>
                        </div>
                    </div>
                </div>
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-6 scrollbar-hide">
                <AnimatePresence initial={false}>
                    {messages.map((m) => (
                        <motion.div
                            key={m.id}
                            initial={{ opacity: 0, y: 10, scale: 0.98 }}
                            animate={{ opacity: 1, y: 0, scale: 1 }}
                            className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}
                        >
                            <div className={`max-w-[92%] flex flex-col gap-2 ${m.role === 'user' ? 'items-end' : 'items-start'}`}>
                                <div className={`max-w-[85%] p-4 rounded-2xl text-xs font-medium leading-relaxed ${m.role === 'user'
                                    ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/10'
                                    : 'bg-white/5 border border-white/10 text-neutral-300 backdrop-blur-md'
                                    }`}>
                                    {m.content}

                                    {m.type === 'code' && (
                                        <div className="mt-4 p-3 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center gap-3">
                                            <div className="p-1.5 rounded-lg bg-blue-500/20">
                                                <Zap className="w-3.5 h-3.5 text-blue-400" />
                                            </div>
                                            <div className="flex flex-col">
                                                <span className="text-[10px] font-black text-blue-400 uppercase tracking-widest">Logic Active</span>
                                                <span className="text-[9px] text-blue-500/70 font-bold uppercase tracking-tighter">Code & Architecture Synced</span>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            </div>
                        </motion.div>
                    ))}
                    {loading && (
                        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex justify-start">
                            <div className="bg-white/5 border border-white/5 rounded-2xl px-5 py-3 flex items-center gap-4">
                                <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                                <span className="text-[10px] text-neutral-400 font-bold tracking-widest uppercase">{loadingMessage}</span>
                            </div>
                        </motion.div>
                    )}
                </AnimatePresence>
                <div ref={endRef} />
            </div>

            <div className="p-5 bg-white/5 backdrop-blur-2xl border-t border-white/5 sticky bottom-0">
                <div className="relative group">
                    <input
                        ref={inputRef}
                        type="text"
                        value={input}
                        onChange={e => setInput(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && send()}
                        placeholder="Explain, design or code..."
                        className="w-full bg-white/5 border border-white/10 rounded-2xl p-4 pr-14 text-xs text-white placeholder-neutral-600 focus:outline-none focus:border-blue-500/50 transition-all resize-none h-24 font-sans font-medium"
                        disabled={loading}
                    />
                    <button
                        onClick={() => send()}
                        disabled={loading || !input.trim()}
                        className="absolute bottom-4 right-4 p-2 rounded-xl bg-blue-600 text-white hover:bg-blue-500 transition-all disabled:opacity-50 disabled:hover:bg-blue-600 shadow-lg shadow-blue-600/20 active:scale-[0.95]"
                    >
                        <Send className="w-4.5 h-4.5" />
                    </button>
                </div>
                <div className="mt-4 text-[9px] text-center text-neutral-600 font-black uppercase tracking-[0.3em]">
                    HardcoreAI Industrial Architecture Engine v2.0
                </div>
            </div>
        </div>
    );
};

export default AIAssistant;
