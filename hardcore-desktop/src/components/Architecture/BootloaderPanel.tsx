import React from 'react';
import { useAppStore } from '../../store/useAppStore';
import { Terminal, Shield } from 'lucide-react';

const BootloaderPanel: React.FC = () => {
    const report = useAppStore(state => state.architectureReport);
    if (!report?.bootloader) return null;

    const { bootloader } = report;

    return (
        <div className="space-y-6">
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-md">
                <div className="flex items-center gap-4 mb-6 border-b border-white/5 pb-6">
                    <div className="p-3 rounded-xl bg-amber-600/20 border border-amber-500/30">
                        <Terminal className="w-6 h-6 text-amber-400" />
                    </div>
                    <div>
                        <h3 className="text-xl font-black text-white tracking-tight">Bootloader Configuration</h3>
                        <p className="text-xs text-neutral-500 font-bold uppercase tracking-widest">Secondary Program Loader & Board Config</p>
                    </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="p-4 rounded-xl bg-white/5 border border-white/5">
                        <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest mb-2 block">Bootloader Type</span>
                        <div className="text-sm font-bold text-amber-400">{bootloader.type}</div>
                    </div>
                    <div className="p-4 rounded-xl bg-white/5 border border-white/5">
                        <span className="text-[10px] font-black text-neutral-500 uppercase tracking-widest mb-2 block">Configuration File</span>
                        <div className="text-sm font-bold text-blue-400">{bootloader.config_file}</div>
                    </div>
                </div>

                {bootloader.secure_boot && (
                    <div className="mt-6 p-4 rounded-xl bg-amber-500/5 border border-amber-500/10">
                        <div className="flex items-center gap-2 mb-2 text-amber-400">
                            <Shield className="w-4 h-4" />
                            <span className="text-[10px] font-black uppercase tracking-widest">Secure Boot Strategy</span>
                        </div>
                        <p className="text-sm text-neutral-300 italic">{bootloader.secure_boot}</p>
                    </div>
                )}
            </div>
        </div>
    );
};

export default BootloaderPanel;
