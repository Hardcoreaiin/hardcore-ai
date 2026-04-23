import React, { useState, useEffect } from 'react';
import { Play, Pause, RotateCcw, CheckCircle, Circle } from 'lucide-react';
import { useAppStore } from '../../store/useAppStore';

const ExecutionStepsPanel: React.FC = () => {
    const executionSteps = useAppStore(state => state.executionSteps);
    const [isPlaying, setIsPlaying] = useState(false);
    const [currentStep, setCurrentStep] = useState(0);

    const steps = executionSteps;

    useEffect(() => {
        if (steps && steps.length > 0) {
            setCurrentStep(0);
        }
    }, [steps]);

    useEffect(() => {
        let interval: any;
        if (isPlaying && steps && currentStep < steps.length) {
            const step = steps[currentStep];
            const delay = step.action === 'wait' ? Math.min(step.ms || 1000, 2000) : 800;

            // Dispatch event for visualization
            window.dispatchEvent(new CustomEvent('execution-step', { detail: step }));

            interval = setTimeout(() => {
                if (currentStep + 1 < steps.length) {
                    setCurrentStep(curr => curr + 1);
                } else {
                    setIsPlaying(false);
                }
            }, delay);
        }
        return () => clearTimeout(interval);
    }, [isPlaying, currentStep, steps]);

    const handleRestart = () => {
        setCurrentStep(0);
        setIsPlaying(true);
    };

    if (!steps || steps.length === 0) {
        return (
            <div className="h-full flex flex-col items-center justify-center text-neutral-500 p-6 text-center">
                <Circle className="w-10 h-10 mb-3 opacity-20" />
                <p className="text-sm">No execution steps available</p>
                <p className="text-xs mt-1 text-neutral-600">Generate firmware to see step-by-step execution</p>
            </div>
        );
    }

    return (
        <div className="h-full flex flex-col bg-neutral-950">
            {/* Header with Controls */}
            <div className="flex items-center justify-between px-4 py-2 border-b border-neutral-800 bg-neutral-900">
                <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold text-green-400 uppercase tracking-wider">Execution Steps</span>
                    <span className="text-xs text-neutral-500">({steps.length} steps)</span>
                </div>
                <div className="flex gap-2">
                    <button
                        onClick={handleRestart}
                        className="p-1.5 hover:bg-neutral-800 rounded text-neutral-400 hover:text-white transition-colors"
                        title="Restart"
                    >
                        <RotateCcw className="w-4 h-4" />
                    </button>
                    <button
                        onClick={() => setIsPlaying(!isPlaying)}
                        className={`p-1.5 rounded transition-colors ${isPlaying ? 'bg-red-500/20 text-red-400' : 'bg-green-500/20 text-green-400'}`}
                        title={isPlaying ? "Pause" : "Play"}
                    >
                        {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                    </button>
                </div>
            </div>

            {/* Progress Bar */}
            <div className="h-1 w-full bg-neutral-800">
                <div
                    className="h-full bg-green-500 transition-all duration-300 ease-out"
                    style={{ width: `${((currentStep + 1) / steps.length) * 100}%` }}
                />
            </div>

            {/* Steps List */}
            <div className="flex-1 overflow-y-auto p-3 space-y-2">
                {steps.map((step, index) => {
                    const isActive = index === currentStep;
                    const isCompleted = index < currentStep;

                    return (
                        <div
                            key={index}
                            className={`flex items-start gap-3 p-3 rounded border transition-all ${isActive
                                ? 'bg-green-900/20 border-green-500/50 shadow-[0_0_10px_rgba(34,197,94,0.3)]'
                                : isCompleted
                                    ? 'bg-neutral-800/50 border-neutral-700'
                                    : 'bg-neutral-900 border-neutral-800'
                                }`}
                        >
                            {/* Step Icon */}
                            <div className={`flex-shrink-0 mt-0.5 ${isCompleted ? 'text-green-500' : isActive ? 'text-green-400' : 'text-neutral-600'}`}>
                                {isCompleted ? <CheckCircle className="w-5 h-5" /> : <Circle className="w-5 h-5" />}
                            </div>

                            {/* Step Content */}
                            <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2 mb-1">
                                    <span className={`text-xs font-mono px-1.5 py-0.5 rounded ${isActive ? 'bg-green-500/20 text-green-300' : 'bg-neutral-800 text-neutral-500'
                                        }`}>
                                        Step {step.step}
                                    </span>
                                    <span className={`text-xs font-medium uppercase ${isActive ? 'text-green-300' : 'text-neutral-400'
                                        }`}>
                                        {step.action}
                                    </span>
                                </div>
                                <p className={`text-sm ${isActive ? 'text-neutral-200' : 'text-neutral-400'}`}>
                                    {step.description}
                                </p>
                                {step.pin !== undefined && (
                                    <div className="mt-1 text-xs text-neutral-500">
                                        GPIO {step.pin} → {step.value === 1 ? 'HIGH' : 'LOW'}
                                    </div>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
};

export default ExecutionStepsPanel;
