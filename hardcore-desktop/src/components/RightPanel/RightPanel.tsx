import React, { useState, useEffect } from 'react';
import { Network, Cpu, FileText, GraduationCap, Package, BookOpen } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import PinConnectionsTab from '../BottomPanel/PinConnectionsTab';
import PeripheralsTab from '../BottomPanel/PeripheralsTab';
import DatasheetUpload from '../BottomPanel/DatasheetUpload';
import LearningExplanationPanel from './LearningExplanationPanel';
import ComponentLibrary from './ComponentLibrary';
import LessonMode from '../Lessons/LessonMode';
import { useAppState } from '../../context/AppStateContext';

import { PanelGroup, Panel, PanelResizeHandle } from 'react-resizable-panels';

interface RightPanelProps {
    selectedBoard: string;
    learningMode?: boolean;
}

type Tab = 'visual' | 'peripherals' | 'docs' | 'components' | 'lessons';

const RightPanel: React.FC<RightPanelProps> = ({ selectedBoard, learningMode = true }) => {
    const [tab, setTab] = useState<Tab>('visual');
    const { state } = useAppState();
    const { learningData } = state;

    const tabs: { id: Tab; label: string; icon: React.ReactNode }[] = [
        { id: 'visual', label: 'Visual', icon: <Network className="w-4 h-4" /> },
        { id: 'peripherals', label: 'HW', icon: <Cpu className="w-4 h-4" /> },
        { id: 'components', label: 'BOM', icon: <Package className="w-4 h-4" /> },
        { id: 'lessons', label: 'EDU', icon: <BookOpen className="w-4 h-4" /> },
        { id: 'docs', label: 'PDF', icon: <FileText className="w-4 h-4" /> },
    ];

    return (
        <div className="h-full flex flex-col bg-transparent">
            {/* Nav Bar */}
            <div className="flex bg-white/5 border-b border-white/10 p-1">
                {tabs.map(t => (
                    <button
                        key={t.id}
                        onClick={() => setTab(t.id)}
                        className={`group relative flex flex-col items-center gap-1.5 py-4 transition-all flex-1 ${tab === t.id ? 'text-violet-400' : 'text-neutral-600 hover:text-neutral-400'
                            }`}
                    >
                        <div className={`p-2 rounded-xl transition-all ${tab === t.id ? 'bg-violet-500/10 shadow-inner' : 'bg-transparent'}`}>
                            {t.icon}
                        </div>
                        <span className="text-[9px] font-black uppercase tracking-[0.2em]">{t.label}</span>
                        {tab === t.id && (
                            <motion.div
                                layoutId="rightTab"
                                className="absolute bottom-0 left-2 right-2 h-0.5 bg-violet-500 rounded-full shadow-[0_0_10px_#7c3aed]"
                            />
                        )}
                    </button>
                ))}
            </div>

            <div className="flex-1 overflow-hidden relative">
                {learningMode && tab === 'visual' ? (
                    <PanelGroup direction="vertical">
                        <Panel defaultSize={60} minSize={30}>
                            <div className="h-full overflow-hidden">
                                <PinConnectionsTab selectedBoard={selectedBoard} />
                            </div>
                        </Panel>
                        <PanelResizeHandle className="h-1.5 bg-white/5 hover:bg-violet-500/20 transition-all cursor-row-resize flex items-center justify-center group z-30">
                            <div className="w-12 h-0.5 rounded-full bg-neutral-800 group-hover:bg-violet-400 transition-colors" />
                        </PanelResizeHandle>
                        <Panel defaultSize={40} minSize={20}>
                            <div className="h-full overflow-hidden">
                                <LearningExplanationPanel />
                            </div>
                        </Panel>
                    </PanelGroup>
                ) : (
                    <div className="h-full overflow-hidden">
                        {tab === 'visual' && <PinConnectionsTab selectedBoard={selectedBoard} />}
                        {tab === 'peripherals' && <PeripheralsTab selectedBoard={selectedBoard} />}
                        {tab === 'components' && <ComponentLibrary />}
                        {tab === 'lessons' && <LessonMode />}
                        {tab === 'docs' && <DatasheetUpload />}
                    </div>
                )}
            </div>
        </div>
    );
};

export default RightPanel;
