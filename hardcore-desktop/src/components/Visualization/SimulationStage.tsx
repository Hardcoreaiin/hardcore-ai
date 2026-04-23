import React, { useMemo, useState, useEffect, useRef, useCallback } from 'react';
import { useAppStore } from '../../store/useAppStore';

// Import Wokwi Elements (ensure they are registered)
import '@wokwi/elements';

// Wokwi elements typings are provided by @wokwi/elements - no need to re-declare.

interface SimulationStageProps {
    isRunning: boolean;
    onReset?: () => void;
}

const SimulationStage: React.FC<SimulationStageProps> = ({ isRunning, onReset }) => {
    const diagram = useAppStore(state => state.diagram);
    const executionSteps = useAppStore(state => state.executionSteps);
    const simulationLogic = useAppStore(state => state.simulationLogic);

    // Simulation State
    const [simTime, setSimTime] = useState(0);
    const [pinStates, setPinStates] = useState<Record<number, number>>({});

    // Refs for loop
    const stateRef = useRef<any>({});
    const logicFnRef = useRef<Function | null>(null);
    const requestRef = useRef<number>();
    const startTimeRef = useRef<number>(0);
    const inputsRef = useRef<Record<number, number>>({});
    const lastRunRef = useRef<boolean>(false);

    // Reset handler
    useEffect(() => {
        if (onReset) {
            // We listen to the parent's reset action? 
            // Actually, usually parent calls onReset to notify child? No.
            // If parent handles state, parent should just switch isRunning to false and maybe trigger a reset here?
            // To allow parent to trigger a reset, we can use a key or a specific effect.
            // But here, onReset prop is a callback if the CHILD wanted to reset.
            // Since we lifted state, the PARENT controls reset.
            // So we should expose a way to reset internal state when parent says so.
            // The simplest way is to expose `simTime` to parent, but that's too much coupling.
            // Better: use a ref or an imperative handle.
            // OR: Parent changes a "resetToken" prop. 
            // But let's keep it simple: if isRunning goes false, we just pause.
            // If we need to reset to 0, we can watch a prop.
            // For now, let's assume if isRunning flips to true from false, we resume.
            // If the user hits STOP/RESET in parent, parent might want to send a signal.
            // Let's implement an effect that resets if `isRunning` is false AND `simTime` should be 0? 
            // No, that's ambiguous.
            // Let's rely on `key` prop on the component in parent to hard reset it! 
            // That's the React way. Parent: <SimulationStage key={resetCount} ... />
        }
    }, [onReset]);

    // 1. Initialize Logic Function
    useEffect(() => {
        if (simulationLogic?.logic_js) {
            try {
                // Create function: (inputs, state, time) => { outputs, nextState }
                // eslint-disable-next-line
                const fn = new Function('inputs', 'state', 'time', simulationLogic.logic_js);
                logicFnRef.current = fn;
                stateRef.current = {}; // Reset state
                console.log("Simulation Logic Loaded");
            } catch (e) {
                console.error("Failed to compile simulation logic:", e);
            }
        }
    }, [simulationLogic]);

    // Handle isRunning changes (Resume/Pause)
    useEffect(() => {
        if (isRunning && !lastRunRef.current) {
            // Resume: adjust start time to account for paused duration
            startTimeRef.current = performance.now() - simTime;
        }
        lastRunRef.current = isRunning;
    }, [isRunning, simTime]);

    // 2. Simulation Loop
    const animate = useCallback((time: number) => {
        if (!isRunning) {
            // Stop the loop if not running
            return;
        }

        const elapsed = time - startTimeRef.current;
        setSimTime(elapsed);

        // Run Logic
        if (logicFnRef.current) {
            try {
                const result = logicFnRef.current(inputsRef.current, stateRef.current, elapsed);
                if (result) {
                    if (result.outputs) {
                        setPinStates(prev => ({ ...prev, ...result.outputs }));
                    }
                    if (result.nextState) {
                        stateRef.current = result.nextState;
                    }
                }
            } catch (e) {
                console.error("Runtime Simulation Error:", e);
                // We should probably notify parent to stop, but for now just stop locally effectively
                // But we can't change props.
                // We'll just stop the loop request.
                return;
            }
        } else if (executionSteps.length > 0) {
            // Fallback: Use Execution Trace Animation if no logic
            const stepIndex = Math.floor(elapsed / 100) % executionSteps.length;
            const step = executionSteps[stepIndex];
            if (step && step.pin !== undefined) {
                setPinStates(prev => ({ ...prev, [step.pin]: step.value || 0 }));
            }
        }

        requestRef.current = requestAnimationFrame(animate);
    }, [isRunning, executionSteps]);

    useEffect(() => {
        if (isRunning) {
            requestRef.current = requestAnimationFrame(animate);
        } else {
            if (requestRef.current) cancelAnimationFrame(requestRef.current);
        }
        return () => {
            if (requestRef.current) cancelAnimationFrame(requestRef.current);
        };
    }, [isRunning, animate]);

    // 3. Interaction Handlers
    const handleInput = (pin: number, value: number) => {
        inputsRef.current[pin] = value;
    };

    // 4. Map connections for rendering
    const pinMap = useMemo(() => {
        const map: Record<number, string> = {};
        if (!diagram?.connections) return map;
        diagram.connections.forEach((conn: any[]) => {
            const [source, target] = conn;
            let pinStr = null;
            let targetId = null;
            if (source.startsWith('esp32:')) {
                pinStr = source.split(':')[1];
                targetId = target.split(':')[0];
            } else if (target.startsWith('esp32:')) {
                pinStr = target.split(':')[1];
                targetId = source.split(':')[0];
            }
            if (pinStr && !isNaN(parseInt(pinStr))) {
                map[parseInt(pinStr)] = targetId;
            }
        });
        return map;
    }, [diagram]);

    if (!diagram) {
        return (
            <div className="flex items-center justify-center h-full text-gray-500">
                Waiting for circuit generation...
            </div>
        );
    }

    return (
        <div className="w-full h-full bg-[#1e1e1e] relative overflow-hidden flex flex-col">
            {/* Sim Time Overlay (Mini) */}
            <div className="absolute top-2 right-2 z-10 px-2 py-0.5 rounded bg-black/40 text-[10px] text-neutral-500 font-mono">
                T: {(simTime / 1000).toFixed(1)}s
            </div>

            {/* Stage */}
            <div className="flex-1 relative">
                <div className="absolute inset-0 opacity-20"
                    style={{ backgroundImage: 'radial-gradient(#555 1px, transparent 1px)', backgroundSize: '20px 20px' }} />

                <div className="w-full h-full transform scale-100 origin-top-left p-10">
                    {/* Wires */}
                    <svg className="absolute inset-0 w-full h-full pointer-events-none">
                        {diagram.connections?.map((conn: any[], i: number) => {
                            const [src, tgt, color] = conn;
                            const srcId = src.split(':')[0];
                            const tgtId = tgt.split(':')[0];
                            // Naive coordinate lookup - simplified for robustness
                            const srcPart = diagram.parts?.find((p: any) => p.id === srcId);
                            const tgtPart = diagram.parts?.find((p: any) => p.id === tgtId);
                            if (!srcPart || !tgtPart) return null;

                            const x1 = parseInt(srcPart.left) + 20;
                            const y1 = parseInt(srcPart.top) + 20;
                            const x2 = parseInt(tgtPart.left) + 20;
                            const y2 = parseInt(tgtPart.top) + 20;

                            return (
                                <line key={i} x1={x1} y1={y1} x2={x2} y2={y2}
                                    stroke={color === 'default' ? '#666' : color}
                                    strokeWidth="3" opacity="0.6" />
                            );
                        })}
                    </svg>

                    {/* Parts */}
                    {diagram.parts?.map((part: any, i: number) => {
                        const { type, id, top, left, attrs } = part;
                        const style: React.CSSProperties = { position: 'absolute', top: `${top}px`, left: `${left}px` };

                        // Find connected PIN for this part to set attributes
                        const pin = parseInt(Object.keys(pinMap).find(p => pinMap[parseInt(p)] === id) || '-1');
                        const pinVal = pin !== -1 ? (pinStates[pin] || 0) : 0;

                        if (type === 'wokwi-esp32-devkit-v1') {
                            // @ts-ignore
                            return <wokwi-esp32-devkit-v1 key={id} style={style} />;
                        }
                        if (type === 'wokwi-led') {
                            // @ts-ignore
                            return <wokwi-led key={id} style={style} color={attrs.color} value={pinVal === 1} label={id} />;
                        }
                        if (type === 'wokwi-pushbutton') {
                            // Find which pin this button drives (logic is reversed for input: button drives logic)
                            // But here we need to capture events
                            return (
                                <div key={id} style={{ position: 'absolute', top: `${top}px`, left: `${left}px` } as React.CSSProperties}
                                    onMouseDown={() => handleInput(pin, 1)}
                                    // Use window mouse up to catch release outside element
                                    onMouseUp={() => handleInput(pin, 0)}
                                    onMouseLeave={() => handleInput(pin, 0)}>
                                    <wokwi-pushbutton color={attrs.color} />
                                </div>
                            );
                        }
                        if (type === 'wokwi-resistor') {
                            // @ts-ignore
                            return <wokwi-resistor key={id} style={style} value={attrs.value} />;
                        }
                        return null;
                    })}
                </div>
            </div>
        </div>
    );
};

export default SimulationStage;
