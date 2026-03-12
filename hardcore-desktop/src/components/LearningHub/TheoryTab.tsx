import React, { useState, useCallback, useRef } from 'react';
import { motion } from 'framer-motion';
import { Lightbulb, Zap, Code, Play, Globe, ZoomIn, ZoomOut, Maximize2, BookOpen } from 'lucide-react';
import { useAppState } from '../../context/AppStateContext';

const TheoryTab: React.FC = () => {
    const { state } = useAppState();
    const { theory } = state;
    const [zoom, setZoom] = useState(1);
    const containerRef = useRef<HTMLDivElement>(null);

    const handleWheel = useCallback((e: React.WheelEvent) => {
        if (e.ctrlKey || e.metaKey) {
            e.preventDefault();
            const delta = e.deltaY > 0 ? -0.08 : 0.08;
            setZoom(prev => Math.min(2, Math.max(0.5, prev + delta)));
        }
    }, []);

    if (!theory) {
        return (
            <div className="h-full flex flex-col items-center justify-center text-center gap-4 p-8">
                <div className="p-5 rounded-3xl bg-violet-500/5 border border-violet-500/10">
                    <BookOpen className="w-12 h-12 text-violet-500/30" />
                </div>
                <div>
                    <p className="text-sm font-semibold text-neutral-400">No theory generated yet</p>
                    <p className="text-xs text-neutral-600 mt-1">Generate firmware from the chat to see project theory here</p>
                </div>
            </div>
        );
    }

    const sections = [
        { id: 'concept', title: 'The Core Concept', icon: <Lightbulb className="w-5 h-5" />, gradient: 'from-amber-500 to-orange-500', bg: 'bg-amber-500/5', border: 'border-amber-500/10', accent: 'text-amber-400' },
        { id: 'electrical', title: 'Electrical Principles', icon: <Zap className="w-5 h-5" />, gradient: 'from-blue-500 to-cyan-500', bg: 'bg-blue-500/5', border: 'border-blue-500/10', accent: 'text-blue-400' },
        { id: 'code', title: 'Code Logic', icon: <Code className="w-5 h-5" />, gradient: 'from-emerald-500 to-green-500', bg: 'bg-emerald-500/5', border: 'border-emerald-500/10', accent: 'text-emerald-400' },
        { id: 'execution', title: 'Firmware Execution', icon: <Play className="w-5 h-5" />, gradient: 'from-violet-500 to-purple-500', bg: 'bg-violet-500/5', border: 'border-violet-500/10', accent: 'text-violet-400' },
        { id: 'applications', title: 'Real-World Applications', icon: <Globe className="w-5 h-5" />, gradient: 'from-sky-500 to-blue-500', bg: 'bg-sky-500/5', border: 'border-sky-500/10', accent: 'text-sky-400' },
    ];

    return (
        <div className="h-full flex flex-col overflow-hidden bg-[#08080c]">
            {/* Header with zoom controls */}
            <div className="flex items-center justify-between px-4 py-2 border-b border-white/5 flex-shrink-0 bg-black/40">
                <div className="flex items-center gap-2">
                    <BookOpen className="w-4 h-4 text-violet-400" />
                    <span className="text-xs font-bold text-neutral-300">Project Knowledge Base</span>
                </div>
                <div className="flex items-center gap-1">
                    <button onClick={() => setZoom(z => Math.max(0.5, z - 0.1))} className="p-1.5 rounded-lg hover:bg-white/5 text-neutral-500 hover:text-neutral-300 transition-colors" title="Zoom Out"><ZoomOut className="w-3.5 h-3.5" /></button>
                    <span className="text-[10px] font-mono text-neutral-600 w-10 text-center">{Math.round(zoom * 100)}%</span>
                    <button onClick={() => setZoom(z => Math.min(2, z + 0.1))} className="p-1.5 rounded-lg hover:bg-white/5 text-neutral-500 hover:text-neutral-300 transition-colors" title="Zoom In"><ZoomIn className="w-3.5 h-3.5" /></button>
                    <button onClick={() => setZoom(1)} className="p-1.5 rounded-lg hover:bg-white/5 text-neutral-500 hover:text-neutral-300 transition-colors ml-1" title="Reset"><Maximize2 className="w-3.5 h-3.5" /></button>
                </div>
            </div>

            {/* Scrollable + Zoomable content */}
            <div ref={containerRef} className="flex-1 overflow-y-auto overflow-x-hidden" onWheel={handleWheel}>
                <div style={{ transform: `scale(${zoom})`, transformOrigin: 'top center', transition: 'transform 0.15s ease-out' }}>
                    <div className="p-6 max-w-5xl mx-auto space-y-8 pb-16">
                        {/* Title Section */}
                        <div className="space-y-2">
                            <h2 className="text-4xl font-black bg-gradient-to-r from-violet-400 via-fuchsia-400 to-pink-400 bg-clip-text text-transparent">
                                Project Intelligence
                            </h2>
                            <p className="text-neutral-500 text-sm font-medium tracking-tight">Comprehensive theoretical breakdown of your current hardware configuration</p>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                            {sections.map((section, index) => {
                                const sectionData = theory[section.id];
                                if (!sectionData && section.id !== 'concept') return null;

                                return (
                                    <motion.div
                                        key={section.id}
                                        initial={{ opacity: 0, y: 15, scale: 0.97 }}
                                        animate={{ opacity: 1, y: 0, scale: 1 }}
                                        transition={{ delay: index * 0.08, duration: 0.3 }}
                                        className={`relative p-6 rounded-3xl border ${section.border} ${section.bg} hover:border-violet-500/20 transition-all group overflow-hidden shadow-2xl`}
                                    >
                                        {/* Subtle gradient accent line */}
                                        <div className={`absolute top-0 left-0 right-0 h-[2px] bg-gradient-to-r ${section.gradient} opacity-40 group-hover:opacity-80 transition-opacity`} />

                                        <div className="flex items-center gap-4 mb-5">
                                            <div className={`p-3 rounded-2xl bg-white/5 ${section.accent} group-hover:bg-white/10 transition-colors shadow-inner`}>
                                                {section.icon}
                                            </div>
                                            <h3 className="font-extrabold text-neutral-100 text-base tracking-tight">{section.title}</h3>
                                        </div>

                                        <div className="text-sm text-neutral-400 leading-relaxed">
                                            {typeof sectionData === 'object' ? (
                                                <div className="space-y-4">
                                                    {Object.entries(sectionData).map(([key, value]) => (
                                                        <div key={key} className="flex flex-col gap-1.5 p-3 rounded-xl bg-white/5 border border-white/5">
                                                            <span className={`text-[10px] font-black uppercase tracking-[0.2em] ${section.accent}`}>{key.replace(/_/g, ' ')}</span>
                                                            <span className="text-neutral-200 text-[13px] font-medium leading-relaxed">{String(value)}</span>
                                                        </div>
                                                    ))}
                                                </div>
                                            ) : (
                                                <div className="p-4 rounded-xl bg-white/5 border border-white/5">
                                                    <p className="text-neutral-200 font-medium leading-relaxed">
                                                        {sectionData || `System analysis in progress for ${section.title.toLowerCase()}...`}
                                                    </p>
                                                </div>
                                            )}
                                        </div>
                                    </motion.div>
                                );
                            })}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default TheoryTab;
