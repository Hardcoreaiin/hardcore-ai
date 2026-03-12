import React from 'react';
import { useAppState } from '../../context/AppStateContext';
import { AlertTriangle, TrendingDown, Target, Zap } from 'lucide-react';

const RiskAnalysis: React.FC = () => {
    const { state } = useAppState();
    const { architectureReport } = state;

    if (!architectureReport) return null;

    return (
        <div className="p-8 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 font-sans">
            <div className="flex items-center gap-4 border-b border-white/5 pb-6">
                <div className="w-12 h-12 rounded-xl bg-red-500/10 flex items-center justify-center border border-red-500/20">
                    <AlertTriangle className="w-6 h-6 text-red-400" />
                </div>
                <div>
                    <h1 className="text-2xl font-black text-white tracking-tight">Risk & Feasibility</h1>
                    <p className="text-sm text-neutral-500 font-medium uppercase tracking-wider">Engineering Validation & Mitigation</p>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                    <div className="flex items-center gap-2 text-red-400 font-black text-[10px] uppercase tracking-[0.2em]">
                        <TrendingDown className="w-4 h-4" />
                        Supply Chain Risk
                    </div>
                    <div className="h-1.5 w-full bg-white/5 rounded-full overflow-hidden">
                        <div className="h-full bg-red-500/50 w-[45%]" />
                    </div>
                    <p className="text-[11px] text-neutral-500 font-bold">MEDIUM EXPOSURE</p>
                </div>
                <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                    <div className="flex items-center gap-2 text-amber-400 font-black text-[10px] uppercase tracking-[0.2em]">
                        <Zap className="w-4 h-4" />
                        Technical Risk
                    </div>
                    <div className="h-1.5 w-full bg-white/5 rounded-full overflow-hidden">
                        <div className="h-full bg-amber-500/50 w-[30%]" />
                    </div>
                    <p className="text-[11px] text-neutral-500 font-bold">LOW EXPOSURE</p>
                </div>
                <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                    <div className="flex items-center gap-2 text-blue-400 font-black text-[10px] uppercase tracking-[0.2em]">
                        <Target className="w-4 h-4" />
                        Regulatory Risk
                    </div>
                    <div className="h-1.5 w-full bg-white/5 rounded-full overflow-hidden">
                        <div className="h-full bg-blue-500/50 w-[60%]" />
                    </div>
                    <p className="text-[11px] text-neutral-500 font-bold">HIGH ATTENTION REQUIRED</p>
                </div>
            </div>

            <div className="glass-panel p-8 rounded-3xl border-white/5 bg-gradient-to-br from-red-500/5 to-transparent">
                <div className="flex items-center gap-3 mb-6">
                    <AlertTriangle className="w-5 h-5 text-red-400" />
                    <h3 className="text-sm font-black text-white uppercase tracking-wider">Risk Mitigation Strategy</h3>
                </div>
                <pre className="text-sm text-neutral-300 whitespace-pre-wrap font-medium leading-relaxed italic">
                    {architectureReport.risks}
                </pre>
            </div>
        </div>
    );
};

export default RiskAnalysis;
