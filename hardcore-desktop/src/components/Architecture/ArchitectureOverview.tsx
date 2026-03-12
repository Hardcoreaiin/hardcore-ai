import React from 'react';
import { useAppState } from '../../context/AppStateContext';
import { Layout, Cpu, Boxes } from 'lucide-react';

const ArchitectureOverview: React.FC = () => {
    const { state } = useAppState();
    const { architectureReport } = state;

    if (!architectureReport) {
        return (
            <div className="h-full flex items-center justify-center text-neutral-500 font-medium">
                No architecture documentation generated.
            </div>
        );
    }

    return (
        <div className="p-8 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <div className="flex items-center gap-4 border-b border-white/5 pb-6">
                <div className="w-12 h-12 rounded-xl bg-blue-500/10 flex items-center justify-center border border-blue-500/20">
                    <Layout className="w-6 h-6 text-blue-400" />
                </div>
                <div>
                    <h1 className="text-2xl font-black text-white tracking-tight">System Architecture</h1>
                    <p className="text-sm text-neutral-500 font-medium uppercase tracking-wider">Technical Specification & Overview</p>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                    <div className="flex items-center gap-2 text-blue-400 font-black text-xs uppercase tracking-widest">
                        <Cpu className="w-4 h-4" />
                        Executive Summary
                    </div>
                    <p className="text-neutral-300 leading-relaxed text-sm">
                        {architectureReport.overview}
                    </p>
                </div>

                <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                    <div className="flex items-center gap-2 text-blue-400 font-black text-xs uppercase tracking-widest">
                        <Boxes className="w-4 h-4" />
                        Hardware Blueprint
                    </div>
                    <p className="text-neutral-300 leading-relaxed text-sm">
                        {architectureReport.hardware}
                    </p>
                </div>
            </div>

            <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                <div className="flex items-center gap-2 text-indigo-400 font-black text-xs uppercase tracking-widest">
                    <Layout className="w-4 h-4" />
                    Memory & Storage Topology
                </div>
                <div className="p-4 bg-white/5 rounded-xl border border-white/5">
                    <pre className="text-xs text-neutral-400 whitespace-pre-wrap font-mono leading-relaxed">
                        {architectureReport.memory_storage}
                    </pre>
                </div>
            </div>
        </div>
    );
};

export default ArchitectureOverview;
