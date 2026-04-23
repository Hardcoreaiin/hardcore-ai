import React from 'react';
import { useAppStore } from '../../store/useAppStore';
import { Landmark, CheckCircle2, ShieldCheck, Globe } from 'lucide-react';

const ComplianceReport: React.FC = () => {
    const architectureReport = useAppStore(state => state.architectureReport);

    if (!architectureReport) return null;

    return (
        <div className="p-4 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <div className="flex items-center gap-4 border-b border-white/5 pb-4">
                <div className="w-12 h-12 rounded-xl bg-emerald-500/10 flex items-center justify-center border border-emerald-500/20">
                    <Landmark className="w-6 h-6 text-emerald-400" />
                </div>
                <div>
                    <h1 className="text-2xl font-black text-white tracking-tight">Regulatory Compliance</h1>
                    <p className="text-sm text-neutral-500 font-medium uppercase tracking-wider">Certification Strategy & Standards</p>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                <div className="space-y-6">
                    <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                        <div className="flex items-center gap-3">
                            <Globe className="w-5 h-5 text-emerald-400" />
                            <h3 className="text-sm font-black text-white uppercase tracking-wider">Primary Jurisdictions</h3>
                        </div>
                        <div className="flex flex-wrap gap-2">
                            {['EU RED', 'FCC Part 15C', 'CE Mark', 'RoHS/WEEE', 'UKCA'].map(c => (
                                <span key={c} className="px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-[10px] font-black text-emerald-400 uppercase tracking-widest">
                                    {c}
                                </span>
                            ))}
                        </div>
                    </div>

                    <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                        <div className="flex items-center gap-3">
                            <ShieldCheck className="w-5 h-5 text-emerald-400" />
                            <h3 className="text-sm font-black text-white uppercase tracking-wider">Wireless Modules</h3>
                        </div>
                        <p className="text-sm text-neutral-400 font-sans font-medium italic">
                            Hardware interfaces: {architectureReport.interfaces?.filter((i: any) => i.component.toLowerCase().includes('wifi') || i.component.toLowerCase().includes('bluetooth') || i.component.toLowerCase().includes('radio') || i.component.toLowerCase().includes('wireless')).map((i: any) => `${i.component} (${i.bus})`).join(', ') || 'None specified.'}
                        </p>
                    </div>
                </div>

                <div className="glass-panel p-6 rounded-2xl border-white/5 flex flex-col">
                    <div className="flex items-center gap-3 mb-6">
                        <CheckCircle2 className="w-5 h-5 text-emerald-400" />
                        <h3 className="text-sm font-black text-white uppercase tracking-wider">Compliance Checklist</h3>
                    </div>
                    <div className="flex-1 overflow-y-auto">
                        <ul className="text-sm text-neutral-300 font-sans leading-relaxed space-y-3">
                            {architectureReport.compliance?.map((comp, idx) => (
                                <li key={idx} className="flex gap-3">
                                    <span className="font-bold text-emerald-400 whitespace-nowrap">{comp.standard}:</span>
                                    <span>{comp.description}</span>
                                </li>
                            ))}
                        </ul>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default ComplianceReport;
