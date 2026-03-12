import React, { useState, useEffect } from 'react';
import { BookOpen, Play, Pause, RotateCcw, Lightbulb } from 'lucide-react';

interface LearningPanelProps {
    explanation?: {
        concept: string;
        gpio_logic: string;
        code_explanation: string;
    };
    simulationSteps?: any[];
    onStep?: (step: any) => void;
}

const LearningPanel: React.FC<LearningPanelProps> = ({ explanation, simulationSteps, onStep }) => {
    const [isPlaying, setIsPlaying] = useState(false);
    const [currentStepIndex, setCurrentStepIndex] = useState(0);

    useEffect(() => {
        let interval: any;
        if (isPlaying && simulationSteps && currentStepIndex < simulationSteps.length) {
            const step = simulationSteps[currentStepIndex];
            onStep?.(step); // Execute step visual

            // Determine delay based on step type
            const delay = step.action === 'wait' ? Math.min(step.ms, 2000) : 800; // Cap wait at 2s for UX

            interval = setTimeout(() => {
                if (currentStepIndex + 1 < simulationSteps.length) {
                    setCurrentStepIndex(curr => curr + 1);
                } else {
                    setIsPlaying(false); // End of simulation
                }
            }, delay);
        }
        return () => clearTimeout(interval);
    }, [isPlaying, currentStepIndex, simulationSteps]);

    if (!explanation) return (
        <div className="h-full flex flex-col items-center justify-center text-neutral-500 p-6 text-center">
            <BookOpen className="w-12 h-12 mb-4 opacity-20" />
            <p className="text-sm">Waiting for lesson content...</p>
            <p className="text-xs">Ask for code to start learning.</p>
        </div>
    );

    return (
        <div className="h-full flex flex-col bg-neutral-900 overflow-y-auto">
            {/* Header */}
            <div className="flex items-center gap-2 px-4 py-3 border-b border-neutral-800 bg-violet-950/20">
                <BookOpen className="w-5 h-5 text-violet-400" />
                <span className="font-semibold text-violet-100">Learning Mode</span>
            </div>

            {/* Content */}
            <div className="p-4 space-y-6">

                {/* Concept Section */}
                <div className="space-y-2">
                    <h3 className="text-xs font-bold uppercase tracking-wider text-violet-400 flex items-center gap-2">
                        <Lightbulb className="w-3 h-3" /> Core Concept
                    </h3>
                    <div className="p-3 bg-neutral-800 rounded border-l-2 border-violet-500 text-sm text-neutral-200 leading-relaxed shadow-sm">
                        {explanation.concept}
                    </div>
                </div>

                {/* GPIO Logic */}
                <div className="space-y-2">
                    <h3 className="text-xs font-bold uppercase tracking-wider text-blue-400">Why these pins?</h3>
                    <p className="text-sm text-neutral-400">{explanation.gpio_logic}</p>
                </div>

                {/* Simulation Control */}
                {simulationSteps && simulationSteps.length > 0 && (
                    <div className="p-4 bg-neutral-950 rounded border border-neutral-800 mt-4">
                        <div className="flex items-center justify-between mb-3">
                            <h3 className="text-xs font-bold uppercase tracking-wider text-green-400">Execution Simulation</h3>
                            <div className="flex gap-2">
                                <button onClick={() => { setCurrentStepIndex(0); setIsPlaying(true); }} className="p-1.5 hover:bg-neutral-800 rounded text-neutral-400 hover:text-white" title="Restart">
                                    <RotateCcw className="w-4 h-4" />
                                </button>
                                <button onClick={() => setIsPlaying(!isPlaying)} className={`p-1.5 rounded transition-colors ${isPlaying ? 'bg-red-500/20 text-red-400' : 'bg-green-500/20 text-green-400'}`} title={isPlaying ? "Pause" : "Play"}>
                                    {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                                </button>
                            </div>
                        </div>

                        {/* Current Step Display */}
                        <div className="h-8 flex items-center text-sm font-mono text-neutral-300 bg-neutral-900 px-3 rounded border border-neutral-800">
                            {isPlaying || currentStepIndex > 0 ? (
                                <span className="flex items-center gap-2">
                                    <span className="text-xs px-1.5 py-0.5 rounded bg-neutral-800 text-neutral-500">Step {currentStepIndex + 1}</span>
                                    {simulationSteps[currentStepIndex]?.description}
                                </span>
                            ) : (
                                <span className="text-neutral-600">Ready to simulate</span>
                            )}
                        </div>

                        {/* Progress Bar */}
                        <div className="h-1 w-full bg-neutral-800 mt-3 rounded-full overflow-hidden">
                            <div
                                className="h-full bg-green-500 transition-all duration-300 ease-out"
                                style={{ width: `${((currentStepIndex + 1) / simulationSteps.length) * 100}%` }}
                            />
                        </div>
                    </div>
                )}

                {/* Code Explanation */}
                <div className="space-y-2">
                    <h3 className="text-xs font-bold uppercase tracking-wider text-orange-400">How the code works</h3>
                    <div className="text-sm text-neutral-400 leading-relaxed whitespace-pre-wrap">
                        {explanation.code_explanation}
                    </div>
                </div>

            </div>
        </div>
    );
};

export default LearningPanel;
