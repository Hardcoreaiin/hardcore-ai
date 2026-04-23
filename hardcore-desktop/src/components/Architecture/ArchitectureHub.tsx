import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Layout, ShieldAlert, Landmark, ShoppingCart, AlertTriangle, Code } from 'lucide-react';
import { LayoutGrid, Cpu, Shield, ListChecks, Activity, Layers, Terminal, Activity as ActIcon } from 'lucide-react';
import { useAppStore } from '../../store/useAppStore';
import ArchitectureOverview from './ArchitectureOverview';
import SecurityArchitecture from './SecurityArchitecture';
import ComplianceReport from './ComplianceReport';
import BOMGenerator from './BOMGenerator';
import RiskAnalysis from './RiskAnalysis';
import LinuxStackPanel from './LinuxStackPanel';
import BootloaderPanel from './BootloaderPanel';
import { Sparkles } from 'lucide-react';

const ArchitectureHub: React.FC = () => {
    const architectureReport = useAppStore(state => state.architectureReport);

    if (!architectureReport) {
        return (
            <div className="h-full flex flex-col items-center justify-center p-8 text-center space-y-6">
                <div className="w-16 h-16 rounded-2xl bg-blue-500/10 flex items-center justify-center border border-blue-500/20">
                    <Sparkles className="w-8 h-8 text-blue-400" />
                </div>
                <div className="space-y-2 max-w-md">
                    <h3 className="text-xl font-bold text-white">Digital Twin Not Initialized</h3>
                    <p className="text-neutral-500 text-sm leading-relaxed">
                        Describe an embedded system to generate architecture and firmware.
                    </p>
                </div>
                <div className="flex flex-col gap-2 w-full max-w-sm">
                    <div className="p-4 rounded-xl bg-white/5 border border-white/5 text-[11px] font-mono text-neutral-400 text-left">
                        <span className="text-blue-400">Example:</span> Build a secure i.MX93 embedded Linux system with WiFi 6 and OTA updates.
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="h-full w-full overflow-y-auto scrollbar-hide">
            <div className="max-w-6xl mx-auto space-y-12 pb-24">
                <div id="overview">
                    <ArchitectureOverview />
                </div>
                
                {/* Manual Rendering of Hardware Blueprint if needed or included in overview */}
                
                <div id="linux_stack">
                    <LinuxStackPanel />
                </div>

                <div id="bootloader">
                    <BootloaderPanel />
                </div>

                <div id="security">
                    <SecurityArchitecture />
                </div>

                <div id="compliance">
                    <ComplianceReport />
                </div>

                <div id="bom">
                    <BOMGenerator />
                </div>

                <div id="risk">
                    <RiskAnalysis />
                </div>
            </div>
        </div>
    );
};

export default ArchitectureHub;
