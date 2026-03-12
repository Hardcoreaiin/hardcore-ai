import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Code, Network, BookOpen, Package, GraduationCap } from 'lucide-react';
import { useAppState } from '../../context/AppStateContext';
import { useBoard } from '../../context/BoardContext';
import MonacoEditor from '../Editor/MonacoEditor';
import TheoryTab from './TheoryTab';
import PinDiagramTab from './PinDiagramTab';
import ComponentLibrary from '../RightPanel/ComponentLibrary';

const LearningHub: React.FC = () => {
    const { state, setActiveTab, setGeneratedCode } = useAppState();
    const { activeTab, generatedCode } = state;
    const { selectedBoard } = useBoard();

    // Local buffer for zero-latency editing
    const [localCode, setLocalCode] = React.useState(generatedCode);

    // Sync local to global with debounce
    React.useEffect(() => {
        const timer = setTimeout(() => {
            if (localCode !== generatedCode) {
                setGeneratedCode(localCode);
            }
        }, 500);
        return () => clearTimeout(timer);
    }, [localCode]);

    // Update local when global changes (e.g. new generation)
    React.useEffect(() => {
        setLocalCode(generatedCode);
    }, [generatedCode]);

    const tabs = [
        { id: 'code', label: 'Code', icon: <Code className="w-4 h-4" /> },
        { id: 'diagram', label: 'Pin Diagram', icon: <Network className="w-4 h-4" /> },
        { id: 'theory', label: 'Theory', icon: <BookOpen className="w-4 h-4" /> },
        { id: 'components', label: 'Components', icon: <Package className="w-4 h-4" /> },
    ];

    const renderContent = () => {
        switch (activeTab) {
            case 'code':
                return (
                    <div className="h-full w-full">
                        <MonacoEditor
                            value={localCode}
                            onChange={(val) => setLocalCode(val || '')}
                            language="cpp"
                            openFiles={[{ path: '/src/main.cpp', name: 'main.cpp', content: localCode, isDirty: false }]}
                            activeFilePath="/src/main.cpp"
                            onTabClick={() => { }}
                            onTabClose={() => { }}
                        />
                    </div>
                );
            case 'diagram':
                return <PinDiagramTab />;
            case 'theory':
                return <TheoryTab />;
            case 'components':
                return (
                    <div className="h-full w-full p-4 overflow-y-auto">
                        <ComponentLibrary />
                    </div>
                );
            default:
                return null;
        }
    };

    return (
        <div className="h-full flex flex-col bg-transparent">
            {/* Unified Nav Bar */}
            <div className="flex bg-white/5 border-b border-white/10 p-1 px-4 gap-1">
                {tabs.map(t => (
                    <button
                        key={t.id}
                        onClick={() => setActiveTab(t.id)}
                        className={`group relative flex items-center gap-2 px-4 py-3 transition-all ${activeTab === t.id ? 'text-violet-400' : 'text-neutral-500 hover:text-neutral-300'
                            }`}
                    >
                        <div className={`p-1.5 rounded-lg transition-all ${activeTab === t.id ? 'bg-violet-500/10' : 'bg-transparent'}`}>
                            {t.icon}
                        </div>
                        <span className="text-[10px] font-black uppercase tracking-widest">{t.label}</span>
                        {activeTab === t.id && (
                            <motion.div
                                layoutId="hubActiveTab"
                                className="absolute bottom-0 left-0 right-0 h-0.5 bg-violet-500 shadow-[0_0_10px_#7c3aed]"
                            />
                        )}
                    </button>
                ))}
            </div>

            {/* Hub Content Area */}
            <div className="flex-1 relative overflow-hidden">
                <AnimatePresence mode="wait">
                    <motion.div
                        key={activeTab}
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -10 }}
                        transition={{ duration: 0.2 }}
                        className="h-full w-full"
                    >
                        {renderContent()}
                    </motion.div>
                </AnimatePresence>
            </div>
        </div>
    );
};

export default LearningHub;
