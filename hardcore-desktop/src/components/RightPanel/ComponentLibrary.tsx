import React from 'react';
import { useAppState } from '../../context/AppStateContext';
import { Lightbulb, Gauge, Thermometer, Waves, ToggleLeft, Cpu, Zap, Info, Package } from 'lucide-react';
import { motion } from 'framer-motion';

const ComponentLibrary: React.FC = () => {
    const { state } = useAppState();
    const { components } = state;

    // Icon mapping for component types
    const getComponentIcon = (type: string) => {
        const iconClass = "w-5 h-5";
        switch (type.toLowerCase()) {
            case 'led':
            case 'light':
                return <Lightbulb className={iconClass} />;
            case 'button':
            case 'switch':
                return <ToggleLeft className={iconClass} />;
            case 'motor':
                return <Gauge className={iconClass} />;
            case 'sensor':
            case 'temperature':
                return <Thermometer className={iconClass} />;
            case 'ultrasonic':
                return <Waves className={iconClass} />;
            case 'servo':
            case 'controller':
                return <Cpu className={iconClass} />;
            default:
                return <Zap className={iconClass} />;
        }
    };

    // Color coding by type
    const getComponentColor = (type: string) => {
        switch (type.toLowerCase()) {
            case 'led':
                return { bg: 'bg-yellow-500/10', text: 'text-yellow-400', border: 'border-yellow-500/20 shadow-yellow-500/5' };
            case 'button':
                return { bg: 'bg-blue-500/10', text: 'text-blue-400', border: 'border-blue-500/20 shadow-blue-500/5' };
            case 'motor':
            case 'servo':
                return { bg: 'bg-green-500/10', text: 'text-green-400', border: 'border-green-500/20 shadow-green-500/5' };
            case 'sensor':
                return { bg: 'bg-orange-500/10', text: 'text-orange-400', border: 'border-orange-500/20 shadow-orange-500/5' };
            default:
                return { bg: 'bg-violet-500/10', text: 'text-violet-400', border: 'border-violet-500/20 shadow-violet-500/5' };
        }
    };

    return (
        <div className="h-full flex flex-col bg-neutral-950">
            {/* Header */}
            <div className="px-6 py-4 border-b border-neutral-900 bg-neutral-950/50 backdrop-blur-md sticky top-0 z-10">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-violet-600/10 border border-violet-500/20">
                            <Package className="w-4 h-4 text-violet-400" />
                        </div>
                        <div>
                            <h3 className="text-sm font-bold text-neutral-100">Component Specs</h3>
                            <p className="text-[10px] text-neutral-500 uppercase tracking-widest font-semibold mt-0.5">Hardware Intelligence</p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-y-auto p-6 space-y-4 scrollbar-hide">
                {components.length === 0 ? (
                    <div className="h-full flex flex-col items-center justify-center text-neutral-600 text-center py-12">
                        <div className="w-16 h-16 rounded-full bg-neutral-900 flex items-center justify-center mb-4 border border-neutral-800">
                            <Info className="w-8 h-8 opacity-20" />
                        </div>
                        <p className="text-sm font-medium">No components staged</p>
                        <p className="text-xs mt-1 text-neutral-700 max-w-[200px]">
                            Required hardware will appear here after design generation
                        </p>
                    </div>
                ) : (
                    <div className="grid gap-4">
                        {components.map((component, index) => {
                            const colors = getComponentColor(component.type || 'default');

                            return (
                                <motion.div
                                    key={index}
                                    initial={{ opacity: 0, x: 20 }}
                                    animate={{ opacity: 1, x: 0 }}
                                    transition={{ delay: index * 0.1 }}
                                    className={`relative p-5 rounded-2xl border ${colors.border} ${colors.bg} backdrop-blur-sm group overflow-hidden shadow-xl`}
                                >
                                    {/* Decorative background element */}
                                    <div className={`absolute -right-4 -top-4 w-24 h-24 rounded-full ${colors.bg} blur-3xl opacity-30 group-hover:opacity-50 transition-opacity`} />

                                    <div className="flex items-start gap-4 relative z-10">
                                        {/* Icon Container */}
                                        <div className={`p-3 rounded-xl ${colors.bg} ${colors.text} border ${colors.border} shadow-inner`}>
                                            {getComponentIcon(component.type || 'default')}
                                        </div>

                                        {/* Core Info */}
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center justify-between mb-1.5">
                                                <h4 className="text-sm font-black text-white tracking-tight uppercase">
                                                    {component.name}
                                                </h4>
                                                <span className={`text-[9px] font-black px-2 py-0.5 rounded uppercase border ${colors.border} ${colors.text}`}>
                                                    {component.type || 'Module'}
                                                </span>
                                            </div>

                                            {component.purpose && (
                                                <p className="text-[11px] text-neutral-400 font-medium leading-relaxed mb-4 line-clamp-2">
                                                    {component.purpose}
                                                </p>
                                            )}

                                            {/* Tech Specs Grid */}
                                            <div className="grid grid-cols-2 gap-3 pb-2">
                                                <div className="bg-black/20 rounded-lg p-2.5 border border-white/5">
                                                    <div className="text-[9px] text-neutral-600 uppercase font-black tracking-tighter mb-1">Voltage</div>
                                                    <div className="text-xs font-mono font-bold text-neutral-200">
                                                        {component.voltage || '3.3V - 5V'}
                                                    </div>
                                                </div>
                                                <div className="bg-black/20 rounded-lg p-2.5 border border-white/5">
                                                    <div className="text-[9px] text-neutral-600 uppercase font-black tracking-tighter mb-1">Current</div>
                                                    <div className="text-xs font-mono font-bold text-neutral-200">
                                                        {component.current || '~20mA'}
                                                    </div>
                                                </div>
                                            </div>

                                            {/* Pinout Section */}
                                            {component.pins && component.pins.length > 0 && (
                                                <div className="mt-2 pt-3 border-t border-white/5">
                                                    <div className="text-[9px] text-neutral-600 uppercase font-black tracking-tighter mb-2">Pin Assignments</div>
                                                    <div className="flex flex-wrap gap-1.5">
                                                        {component.pins.map((pin, pIdx) => (
                                                            <span key={pIdx} className="px-2 py-0.5 rounded bg-white/5 text-[10px] font-mono text-neutral-300 border border-white/5 uppercase">
                                                                {pin}
                                                            </span>
                                                        ))}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                </motion.div>
                            );
                        })}
                    </div>
                )}
            </div>
        </div>
    );
};

export default ComponentLibrary;
