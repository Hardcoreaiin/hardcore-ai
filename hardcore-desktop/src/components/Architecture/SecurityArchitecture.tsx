import React from 'react';
import { useAppStore } from '../../store/useAppStore';
import { ShieldAlert, Key, FileLock2, RefreshCw } from 'lucide-react';

const SecurityArchitecture: React.FC = () => {
    const architectureReport = useAppStore(state => state.architectureReport);

    if (!architectureReport) return null;

    return (
        <div className="p-4 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <div className="flex items-center gap-4 border-b border-white/5 pb-4">
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
                    <div className="text-sm text-neutral-300 font-sans leading-relaxed space-y-4">
                        <div className="flex items-center gap-2">
                            <span className="font-bold text-orange-400">Secure Enclave:</span> {architectureReport.security?.secure_enclave || 'N/A'}
                        </div>
                        <div className="flex items-center gap-2">
                            <span className="font-bold text-orange-400">Cryptography:</span> {architectureReport.security?.cryptography || 'N/A'}
                        </div>
                        {architectureReport.security?.features && (
                            <div>
                                <span className="font-bold text-orange-400 block mb-1">Features:</span>
                                <ul className="list-disc list-inside ml-2">
                                    {architectureReport.security.features.map((f, i) => <li key={i}>{f}</li>)}
                                </ul>
                            </div>
                        )}
                        <p className="border-t border-white/5 pt-4 mt-4 italic">
                            {architectureReport.security?.explanation}
                        </p>
                    </div>
                </div>
            </div>

            <div className="glass-panel overflow-hidden rounded-2xl border-white/5">
                <div className="p-4 bg-orange-500/5 border-b border-white/5">
                    <h3 className="text-xs font-black text-orange-400 uppercase tracking-[0.2em]">Firmware Pipeline & OTA</h3>
                </div>
                <div className="p-6">
                    <div className="text-sm text-neutral-300 font-sans leading-relaxed space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <span className="font-bold text-orange-400 block">OS/RTOS:</span> 
                                {architectureReport.firmware_stack?.os || 'N/A'}
                            </div>
                            <div>
                                <span className="font-bold text-orange-400 block">Bootloader:</span> 
                                {architectureReport.firmware_stack?.bootloader || 'N/A'}
                            </div>
                            <div>
                                <span className="font-bold text-orange-400 block">OTA Mechanism:</span> 
                                {architectureReport.firmware_stack?.ota || 'N/A'}
                            </div>
                        </div>
                        {architectureReport.firmware_stack?.drivers && (
                            <div>
                                <span className="font-bold text-orange-400 block mb-1">Required Drivers:</span>
                                <div className="flex flex-wrap gap-2">
                                    {architectureReport.firmware_stack.drivers.map((d, i) => (
                                        <span key={i} className="px-2 py-1 bg-white/5 rounded text-xs">{d}</span>
                                    ))}
                                </div>
                            </div>
                        )}
                        <p className="border-t border-white/5 pt-4 mt-4 italic">
                            {architectureReport.firmware_stack?.explanation}
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default SecurityArchitecture;
