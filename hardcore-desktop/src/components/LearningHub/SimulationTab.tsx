import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Play, Pause, RotateCcw, Zap, Info } from 'lucide-react';
import SimulationStage from '../Visualization/SimulationStage';
import { useAppStore } from '../../store/useAppStore';
import { useBoard } from '../../context/BoardContext';

const SimulationTab: React.FC = () => {
    const executionSteps = useAppStore(state => state.executionSteps);
    const generatedCode = useAppStore(state => state.generatedCode);
    const { selectedBoard } = useBoard();
    const [isRunning, setIsRunning] = useState(false);
    const [currentStepIndex, setCurrentStepIndex] = useState(0);

    useEffect(() => {
        let interval: NodeJS.Timeout;
        if (isRunning && currentStepIndex < executionSteps.length) {
            const step = executionSteps[currentStepIndex];
            const delay = step.ms || 1000;

            interval = setTimeout(() => {
                const nextStep = executionSteps[currentStepIndex];
                // Dispatch event for digital twin to react
                window.dispatchEvent(new CustomEvent('execution-step', { detail: nextStep }));
                setCurrentStepIndex(prev => prev + 1);
            }, delay);
        } else if (currentStepIndex >= executionSteps.length && executionSteps.length > 0) {
            // Only stop if we are in step-mode. 
            setIsRunning(false);
        }
        return () => clearTimeout(interval);
    }, [isRunning, currentStepIndex, executionSteps]);

    console.log('[SimulationTab] Rendering. Steps:', executionSteps.length, 'Running:', isRunning);

    const reset = () => {
        setIsRunning(false);
        setCurrentStepIndex(0);
    };

    return (
        <div className="h-full flex flex-col p-4 space-y-4">
            {/* Control Bar */}
            <div className="flex items-center justify-between glass-panel p-4 rounded-2xl border border-white/5 shadow-inner">
                <div className="flex items-center gap-4">
                    <button
                        onClick={() => setIsRunning(!isRunning)}
                        className={`p-3 rounded-xl transition-all ${isRunning ? 'bg-amber-500/20 text-amber-500' : 'bg-emerald-500/20 text-emerald-500'
                            }`}
                    >
                        {isRunning ? <Pause className="w-5 h-5" /> : <Play className="w-5 h-5 ml-0.5" />}
                    </button>
                    <button
                        onClick={reset}
                        className="p-3 rounded-xl bg-white/5 text-neutral-400 hover:text-white transition-all"
                    >
                        <RotateCcw className="w-5 h-5" />
                    </button>
                </div>

                <div className="flex flex-col items-end">
                    <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest">Simulation State</span>
                    <div className="flex items-center gap-2">
                        <div className={`w-2 h-2 rounded-full animate-pulse ${isRunning ? 'bg-emerald-500' : 'bg-neutral-600'}`} />
                        <span className={`text-sm font-bold ${isRunning ? 'text-emerald-400' : 'text-neutral-400'}`}>
                            {isRunning ? 'RUNNING' : 'IDLE'}
                        </span>
                    </div>
                </div>
            </div>

            {/* Main Simulation Stage */}
            <div className="flex-1 flex gap-4 min-h-0">
                {/* Visualizer */}
                <div className="flex-1 glass-panel rounded-3xl overflow-hidden relative border border-white/5 bg-black/40">
                    <div className="absolute top-4 left-4 z-10 flex items-center gap-2">
                        <Zap className="w-4 h-4 text-violet-400" />
                        <span className="text-[10px] font-black text-white uppercase tracking-widest">Active Digital Twin</span>
                    </div>

                    <div className="h-full w-full relative">
                        <SimulationStage isRunning={isRunning} onReset={reset} />
                    </div>

                    {/* Simulation Overlays (e.g., active pins) */}
                    <AnimatePresence>
                        {isRunning && currentStepIndex < executionSteps.length && (
                            <motion.div
                                initial={{ opacity: 0, x: -20 }}
                                animate={{ opacity: 1, x: 0 }}
                                exit={{ opacity: 0, x: 20 }}
                                className="absolute bottom-4 left-4 right-4 glass-panel p-3 rounded-xl border border-emerald-500/30 bg-emerald-500/5 backdrop-blur-md flex items-center gap-3"
                            >
                                <div className="p-2 rounded-lg bg-emerald-500/20 text-emerald-400">
                                    <Play className="w-4 h-4" />
                                </div>
                                <div className="flex-1">
                                    <div className="text-[10px] font-black uppercase text-emerald-500">Executing Step {currentStepIndex + 1}</div>
                                    <div className="text-sm text-emerald-100 font-bold tracking-tight">
                                        {executionSteps[currentStepIndex].description}
                                    </div>
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </div>

                {/* Execution Flow Log */}
                <div className="w-80 glass-panel rounded-3xl border border-white/5 bg-black/40 flex flex-col overflow-hidden">
                    <div className="p-4 border-b border-white/5 flex items-center justify-between">
                        <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest">Flow Log</span>
                        <Info className="w-3 h-3 text-neutral-600" />
                    </div>
                    <div className="flex-1 overflow-y-auto p-2 space-y-2">
                        {executionSteps.length === 0 ? (
                            <div className="p-4 text-center text-neutral-500 text-xs italic">
                                No execution steps generated.
                                <br />Try generating firmware first.
                            </div>
                        ) : (
                            executionSteps.map((step, idx) => (
                                <div
                                    key={idx}
                                    className={`p-2 rounded-lg transition-all border ${idx === currentStepIndex
                                        ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
                                        : idx < currentStepIndex
                                            ? 'bg-neutral-800/20 border-white/5 text-neutral-500'
                                            : 'bg-transparent border-transparent text-neutral-600'
                                        }`}
                                >
                                    <div className="text-[9px] font-bold opacity-50">STEP {step.step}</div>
                                    <div className="text-[11px] font-medium leading-tight">{step.description}</div>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
};

export default SimulationTab;
