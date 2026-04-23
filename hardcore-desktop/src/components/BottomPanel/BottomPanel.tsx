import React, { useState, useEffect } from 'react';
import { Terminal, Activity, Layers } from 'lucide-react';
import { motion } from 'framer-motion';
import SerialMonitor from './SerialMonitor';
import BuildOutput from './BuildOutput';

// Desktop bypass header for local authentication
const API_HEADERS = {
    'Content-Type': 'application/json',
    'X-Desktop-Key': 'desktop_local_bypass_hardcore_ai',
};

interface BottomPanelProps {
    selectedBoard: string;
    detectedBoard?: string | null;
    connectedPort?: string | null;
}

type Tab = 'serial' | 'build';

const BottomPanel: React.FC<BottomPanelProps> = ({ selectedBoard, detectedBoard, connectedPort }) => {
    const [tab, setTab] = useState<Tab>('serial');

    const tabs: { id: Tab; label: string; icon: React.ReactNode }[] = [
        { id: 'serial', label: 'Serial Monitor', icon: <Terminal className="w-4 h-4" /> },
        { id: 'build', label: 'Build Output', icon: <Activity className="w-4 h-4" /> },
    ];

    return (
        <div className="h-full flex flex-col bg-transparent">
            {/* Tabs Bar */}
            <div className="flex bg-white/5 border-b border-white/5 px-2">
                {tabs.map(t => (
                    <button
                        key={t.id}
                        onClick={() => setTab(t.id)}
                        className={`flex items-center gap-2 px-6 py-3 text-[11px] font-black uppercase tracking-widest transition-all relative ${tab === t.id
                            ? 'text-blue-400'
                            : 'text-neutral-500 hover:text-neutral-300'
                            }`}
                    >
                        {t.icon}
                        {t.label}
                        {tab === t.id && (
                            <motion.div
                                layoutId="bottomTab"
                                className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500 shadow-[0_0_10px_#3b82f6]"
                            />
                        )}
                    </button>
                ))}
            </div>
            {/* Content Area */}
            <div className="flex-1 overflow-hidden relative">
                {tab === 'serial' && <SerialMonitor />}
                {tab === 'build' && <BuildOutput />}
            </div>
        </div>
    );
};

export default BottomPanel;
