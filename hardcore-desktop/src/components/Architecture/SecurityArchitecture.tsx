import React from 'react';
import { useAppState } from '../../context/AppStateContext';
import { ShieldAlert, Key, FileLock2, RefreshCw } from 'lucide-react';

const SecurityArchitecture: React.FC = () => {
    const { state } = useAppState();
    const { architectureReport } = state;

    if (!architectureReport) return null;

    return (
        <div className="p-8 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <div className="flex items-center gap-4 border-b border-white/5 pb-6">
                <div className="w-12 h-12 rounded-xl bg-orange-500/10 flex items-center justify-center border border-orange-500/20">
                    <ShieldAlert className="w-6 h-6 text-orange-400" />
                </div>
                <div>
                    <h1 className="text-2xl font-black text-white tracking-tight">Security Hardening</h1>
                    <p className="text-sm text-neutral-500 font-medium uppercase tracking-wider">Root of Trust & Secure Boot Chain</p>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                    <div className="flex items-center gap-2 text-orange-400 font-black text-xs uppercase tracking-widest">
                        <Key className="w-4 h-4" />
                        Root of Trust
                    </div>
                    <p className="text-neutral-400 text-sm leading-relaxed">
                        Hardware-based immutable identity and cryptographic foundations for the entire system.
                    </p>
                </div>
                <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                    <div className="flex items-center gap-2 text-orange-400 font-black text-xs uppercase tracking-widest">
                        <FileLock2 className="w-4 h-4" />
                        Boot Chain
                    </div>
                    <p className="text-neutral-400 text-sm leading-relaxed">
                        Multi-stage signature validation from ROM to Kernel/Application.
                    </p>
                </div>
                <div className="glass-panel p-6 rounded-2xl border-white/5 space-y-4">
                    <div className="flex items-center gap-2 text-orange-400 font-black text-xs uppercase tracking-widest">
                        <RefreshCw className="w-4 h-4" />
                        Anti-Rollback
                    </div>
                    <p className="text-neutral-400 text-sm leading-relaxed">
                        Hardware counters to prevent downgrading to vulnerable firmware versions.
                    </p>
                </div>
            </div>

            <div className="glass-panel overflow-hidden rounded-2xl border-white/5">
                <div className="p-4 bg-orange-500/5 border-b border-white/5">
                    <h3 className="text-xs font-black text-orange-400 uppercase tracking-[0.2em]">Security Implementation Plan</h3>
                </div>
                <div className="p-6">
                    <pre className="text-sm text-neutral-300 whitespace-pre-wrap font-sans leading-relaxed">
                        {architectureReport.security}
                    </pre>
                </div>
            </div>

            <div className="glass-panel overflow-hidden rounded-2xl border-white/5">
                <div className="p-4 bg-orange-500/5 border-b border-white/5">
                    <h3 className="text-xs font-black text-orange-400 uppercase tracking-[0.2em]">Firmware Pipeline & OTA</h3>
                </div>
                <div className="p-6">
                    <pre className="text-sm text-neutral-300 whitespace-pre-wrap font-sans leading-relaxed">
                        {architectureReport.firmware_pipeline}
                    </pre>
                </div>
            </div>
        </div>
    );
};

export default SecurityArchitecture;
