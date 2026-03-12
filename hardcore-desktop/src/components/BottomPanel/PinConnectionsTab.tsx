import React from 'react';
import { useAppState } from '../../context/AppStateContext';
import ESP32Board from '../Visualization/ESP32Board';
import { Zap, Info } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

interface PinConnectionsTabProps {
    selectedBoard: string;
}

const PinConnectionsTab: React.FC<PinConnectionsTabProps> = ({ selectedBoard }) => {
    const { state } = useAppState();
    const { components, connections } = state;
    const [simulatedPinStates, setSimulatedPinStates] = React.useState<Record<number, 'HIGH' | 'LOW'>>({});

    // Parse active pins
    const activePins = connections.map(conn => conn.pin).filter(Boolean) as number[];

    React.useEffect(() => {
        const initialStates: Record<number, 'HIGH' | 'LOW'> = {};
        connections.forEach(conn => {
            if (conn.pin) initialStates[conn.pin] = 'LOW';
        });
        setSimulatedPinStates(initialStates);
    }, [connections]);

    React.useEffect(() => {
        const handler = (e: any) => {
            const step = e.detail;
            if (step && step.pin !== undefined) {
                setSimulatedPinStates(prev => ({
                    ...prev,
                    [step.pin]: step.value === 1 ? 'HIGH' : 'LOW'
                }));
            }
        };
        window.addEventListener('execution-step', handler);
        return () => window.removeEventListener('execution-step', handler);
    }, []);

    return (
        <div className="h-full flex flex-col bg-transparent overflow-hidden">
            {/* Header */}
            <div className="px-6 py-5 border-b border-white/5 bg-white/5 backdrop-blur-xl sticky top-0 z-20">
                <div className="flex items-center gap-4">
                    <div className="p-2.5 rounded-xl bg-violet-600/20 border border-violet-500/30 shadow-[0_0_15px_rgba(124,58,237,0.2)]">
                        <Zap className="w-5 h-5 text-violet-400" />
                    </div>
                    <div>
                        <h3 className="text-[15px] font-black text-white tracking-tight">Digital Twin</h3>
                        <p className="text-[10px] text-neutral-500 uppercase tracking-[0.2em] font-black mt-0.5">Physical Matrix</p>
                    </div>
                </div>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-y-auto p-6 space-y-10 scrollbar-hide relative">
                {connections.length === 0 ? (
                    <div className="h-full flex flex-col items-center justify-center text-neutral-600 text-center py-20">
                        <div className="w-20 h-20 rounded-3xl bg-white/5 flex items-center justify-center mb-6 border border-white/5 shadow-inner">
                            <Info className="w-10 h-10 opacity-10" />
                        </div>
                        <p className="text-sm font-black uppercase tracking-widest text-neutral-500">No Physical Data</p>
                        <p className="text-[11px] mt-2 text-neutral-700 max-w-[200px] leading-relaxed">
                            Awaiting neuronal generation from the architectural core...
                        </p>
                    </div>
                ) : (
                    <>
                        {/* THE BOARD VISUALIZATION */}
                        <section className="relative">
                            <h4 className="text-[10px] font-black text-neutral-500 uppercase tracking-[0.3em] mb-6 flex items-center gap-3">
                                <span className="w-8 h-px bg-neutral-800" />
                                Hardware Topography
                            </h4>
                            <div className="glass-panel rounded-3xl p-8 shadow-[0_30px_60px_rgba(0,0,0,0.6)] border-white/5 relative overflow-hidden group">
                                <div className="absolute inset-0 bg-gradient-to-br from-violet-600/10 via-transparent to-black/40 pointer-events-none" />
                                <div className="flex justify-center relative z-10 scale-110">
                                    <ESP32Board activePins={activePins} pinStates={simulatedPinStates} />
                                </div>
                            </div>
                        </section>

                        {/* PIN MAPPING GRID */}
                        <section>
                            <h4 className="text-[10px] font-black text-neutral-500 uppercase tracking-[0.3em] mb-6 flex items-center gap-3">
                                <span className="w-8 h-px bg-neutral-800" />
                                Signal Routing
                            </h4>
                            <div className="grid gap-4">
                                {connections.map((conn, index) => (
                                    <motion.div
                                        key={index}
                                        initial={{ opacity: 0, x: -10 }}
                                        animate={{ opacity: 1, x: 0 }}
                                        transition={{ delay: index * 0.05 }}
                                        className="group relative flex items-center gap-5 p-5 glass-panel rounded-2xl hover:border-violet-500/40 hover:bg-white/5 transition-all shadow-xl"
                                    >
                                        {/* Wire / Signal Path SVG */}
                                        <div className="absolute left-0 top-0 bottom-0 w-1.5 rounded-l-2xl"
                                            style={{ backgroundColor: conn.color === 'black' ? '#222' : conn.color }} />

                                        {/* Component Side */}
                                        <div className="flex-1 min-w-0">
                                            <div className="flex flex-col">
                                                <span className="text-[10px] text-neutral-500 font-black uppercase tracking-widest mb-1">Target</span>
                                                <span className="text-[13px] font-black text-white truncate group-hover:neon-text transition-all">
                                                    {conn.from}
                                                </span>
                                            </div>
                                        </div>

                                        {/* Animation Arrow */}
                                        <div className="flex items-center">
                                            <motion.div
                                                animate={{ x: [0, 5, 0] }}
                                                transition={{ duration: 1.5, repeat: Infinity }}
                                                className="text-white/10"
                                            >
                                                <Zap className="w-3 h-3 text-violet-500/40" />
                                            </motion.div>
                                        </div>

                                        {/* Board Side */}
                                        <div className="flex-1 min-w-0 text-right">
                                            <div className="flex flex-col items-end">
                                                <span className="text-[10px] text-neutral-500 font-black uppercase tracking-widest mb-1">Source Pin</span>
                                                <div className="flex items-center gap-2">
                                                    <span className="px-2 py-0.5 rounded-lg bg-white/5 border border-white/5 text-[11px] font-black text-violet-400 font-mono tracking-tighter shadow-inner">
                                                        GPIO {conn.pin}
                                                    </span>
                                                </div>
                                            </div>
                                        </div>

                                        {/* Signal Pulse Indicator */}
                                        {conn.pin && simulatedPinStates[conn.pin] && (
                                            <div className="ml-2 w-[45px] flex justify-center">
                                                <motion.div
                                                    animate={simulatedPinStates[conn.pin] === 'HIGH' ? { scale: [1, 1.1, 1] } : {}}
                                                    transition={{ repeat: Infinity, duration: 0.5 }}
                                                    className={`px-2.5 py-1 rounded-lg text-[9px] font-black tracking-tighter shadow-lg ${simulatedPinStates[conn.pin] === 'HIGH'
                                                        ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                                                        : 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                                                        }`}
                                                >
                                                    {simulatedPinStates[conn.pin]}
                                                </motion.div>
                                            </div>
                                        )}
                                    </motion.div>
                                ))}
                            </div>
                        </section>

                        {/* BOM SECTION */}
                        {components.length > 0 && (
                            <section className="pb-10">
                                <h4 className="text-[10px] font-black text-neutral-500 uppercase tracking-[0.3em] mb-6 flex items-center gap-3">
                                    <span className="w-8 h-px bg-neutral-800" />
                                    Bill of Materials
                                </h4>
                                <div className="space-y-3">
                                    {components.map((comp, index) => (
                                        <div
                                            key={index}
                                            className="flex items-center justify-between p-4 glass-panel border-white/5 rounded-xl group hover:border-white/10 transition-all shadow-lg"
                                        >
                                            <div className="flex items-center gap-4">
                                                <div className="w-2 h-2 rounded-full bg-violet-600 animate-pulse shadow-[0_0_8px_#7c3aed]" />
                                                <span className="text-[13px] font-black text-neutral-100 uppercase tracking-tight">{comp.name}</span>
                                            </div>
                                            <div className="px-3 py-1 rounded-lg bg-white/3 border border-white/5 text-[9px] font-black text-neutral-500 uppercase tracking-widest">
                                                {comp.type || 'Standard'}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </section>
                        )}
                    </>
                )}
            </div>
        </div>
    );
};

export default PinConnectionsTab;
