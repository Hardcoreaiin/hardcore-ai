import React from 'react';
import { Layers, Terminal, Cpu, Shield, Zap, Activity } from 'lucide-react';
import { useAppStore } from '../../store/useAppStore';

const LinuxStackPanel: React.FC = () => {
    const report = useAppStore(state => state.architectureReport);
    if (!report?.linux_stack) return null;

    const { linux_stack } = report;

    return (
        <div className="space-y-6">
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-md">
                <div className="flex items-center gap-4 mb-6 border-b border-white/5 pb-6">
                    <div className="p-3 rounded-xl bg-violet-600/20 border border-violet-500/30">
                        <Layers className="w-6 h-6 text-violet-400" />
                    </div>
                    <div>
                        <h3 className="text-xl font-black text-white tracking-tight">Linux Stack</h3>
                        <p className="text-xs text-neutral-500 font-bold uppercase tracking-widest">OS Distribution & Base Configuration</p>
                    </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="p-4 rounded-xl bg-white/5 border border-white/5">
                        <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest mb-2 block">Linux Distribution</span>
                        <div className="text-sm font-bold text-violet-400">{linux_stack.linux_distribution}</div>
                    </div>
                    <div className="p-4 rounded-xl bg-white/5 border border-white/5">
                        <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest mb-2 block">Kernel Version</span>
                        <div className="text-sm font-bold text-blue-400">{linux_stack.kernel_version}</div>
                    </div>
                    <div className="p-4 rounded-xl bg-white/5 border border-white/5">
                        <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest mb-2 block">OTA Framework</span>
                        <div className="text-sm font-bold text-emerald-400">{linux_stack.ota_framework}</div>
                    </div>
                    <div className="p-4 rounded-xl bg-white/5 border border-white/5">
                        <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest mb-2 block">Filesystem</span>
                        <div className="text-sm font-bold text-orange-400">{linux_stack.filesystem || 'ext4 + dm-verity'}</div>
                    </div>
                    <div className="p-4 rounded-xl bg-white/5 border border-white/5">
                        <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest mb-2 block">Security Modules</span>
                        <div className="text-sm font-bold text-red-400">{linux_stack.security_modules || 'EdgeLock + HAB'}</div>
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-md">
                    <div className="flex items-center gap-3 mb-4">
                        <Terminal className="w-4 h-4 text-blue-400" />
                        <h4 className="text-xs font-black text-white uppercase tracking-widest">Kernel Drivers</h4>
                    </div>
                    <div className="flex flex-wrap gap-2">
                        {linux_stack.drivers.map((driver, i) => (
                            <span key={i} className="px-3 py-1 rounded-full bg-blue-500/10 border border-blue-500/20 text-[10px] font-bold text-blue-400 uppercase">
                                {driver}
                            </span>
                        ))}
                    </div>
                </div>

                <div className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-md">
                    <div className="flex items-center gap-3 mb-4">
                        <Shield className="w-4 h-4 text-emerald-400" />
                        <h4 className="text-xs font-black text-white uppercase tracking-widest">OS Security</h4>
                    </div>
                    <div className="flex flex-wrap gap-2">
                        {linux_stack.security_features.map((feature, i) => (
                            <span key={i} className="px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-[10px] font-bold text-emerald-400 uppercase">
                                {feature}
                            </span>
                        ))}
                    </div>
                </div>
            </div>
        </div>
    );
};

export default LinuxStackPanel;
