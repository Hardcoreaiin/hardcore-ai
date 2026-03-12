import React from 'react';
import { useAppState } from '../../context/AppStateContext';
import { ShoppingCart, FileSpreadsheet, Package, ListChecks } from 'lucide-react';

const BOMGenerator: React.FC = () => {
    const { state } = useAppState();
    const { architectureReport } = state;

    if (!architectureReport) return null;

    return (
        <div className="p-8 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <div className="flex items-center gap-4 border-b border-white/5 pb-6">
                <div className="w-12 h-12 rounded-xl bg-violet-500/10 flex items-center justify-center border border-violet-500/20">
                    <ShoppingCart className="w-6 h-6 text-violet-400" />
                </div>
                <div>
                    <h1 className="text-2xl font-black text-white tracking-tight">Bill of Materials</h1>
                    <p className="text-sm text-neutral-500 font-medium uppercase tracking-wider">High-Level Component Selection</p>
                </div>
            </div>

            <div className="glass-panel rounded-2xl border-white/5 overflow-hidden">
                <div className="grid grid-cols-4 p-4 bg-white/5 text-[10px] font-black text-neutral-500 uppercase tracking-[0.2em] border-b border-white/5">
                    <div className="col-span-1 flex items-center gap-2">Category</div>
                    <div className="col-span-3 flex items-center gap-2">Recommended Specification</div>
                </div>

                <div className="divide-y divide-white/5">
                    <div className="p-6">
                        <pre className="text-sm text-neutral-300 whitespace-pre-wrap font-sans leading-relaxed">
                            {architectureReport.bom}
                        </pre>
                    </div>
                </div>
            </div>

            <div className="flex items-center gap-4">
                <button className="flex-1 h-14 rounded-2xl bg-violet-600 text-white font-black text-xs uppercase tracking-widest flex items-center justify-center gap-3 hover:bg-violet-500 transition-all shadow-xl shadow-violet-600/10 active:scale-[0.98]">
                    <FileSpreadsheet className="w-4 h-4" />
                    Export CSV Report
                </button>
                <button className="flex-1 h-14 rounded-2xl glass-panel border-white/5 text-white font-black text-xs uppercase tracking-widest flex items-center justify-center gap-3 hover:bg-white/5 transition-all active:scale-[0.98]">
                    <Package className="w-4 h-4 text-violet-400" />
                    Request Quote (Prototype)
                </button>
            </div>
        </div>
    );
};

export default BOMGenerator;
