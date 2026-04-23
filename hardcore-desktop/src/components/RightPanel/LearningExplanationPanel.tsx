import { BookOpen, Lightbulb, Cpu, Code, ShieldAlert, GraduationCap, Microscope, HelpCircle, Zap } from 'lucide-react';
import { useAppStore } from '../../store/useAppStore';

const LearningExplanationPanel: React.FC = () => {
    const learningData = useAppStore(state => state.learningData);

    // Handle both object and string formats
    const content = typeof learningData === 'string'
        ? { code_explanation: learningData }
        : learningData;

    if (!content) {
        return (
            <div className="h-full flex flex-col items-center justify-center text-neutral-500 p-8 text-center bg-transparent">
                <div className="w-16 h-16 rounded-3xl bg-white/5 flex items-center justify-center mb-6 border border-white/5 shadow-inner">
                    <BookOpen className="w-8 h-8 opacity-10" />
                </div>
                <p className="text-xs font-black uppercase tracking-[0.3em] text-neutral-600">Architectural Core Idle</p>
                <p className="text-[10px] mt-2 text-neutral-700 max-w-[180px] leading-relaxed">
                    Awaiting conceptual instruction to initiate pedagogical synchronization.
                </p>
            </div>
        );
    }

    const Section = ({ icon: Icon, title, color, children }: any) => (
        <div className="space-y-3 group">
            <div className="flex items-center gap-3">
                <div className={`p-1.5 rounded-lg bg-${color}-500/10 border border-${color}-500/20 group-hover:shadow-[0_0_10px_rgba(var(--${color}-rgb),0.2)] transition-all`}>
                    <Icon className={`w-3.5 h-3.5 text-${color}-400`} />
                </div>
                <h3 className={`text-[10px] font-black uppercase tracking-[0.2em] text-${color}-400/80`}>{title}</h3>
            </div>
            <div className="glass-panel p-4 rounded-2xl border-white/5 text-[13px] text-neutral-300 leading-relaxed shadow-xl group-hover:border-white/10 transition-all">
                {children}
            </div>
        </div>
    );

    return (
        <div className="h-full flex flex-col bg-transparent overflow-hidden">
            {/* Header */}
            <div className="flex items-center gap-4 px-6 py-5 bg-white/5 backdrop-blur-xl border-b border-white/5 sticky top-0 z-20">
                <div className="p-2 rounded-xl bg-violet-600/20 border border-violet-500/30">
                    <GraduationCap className="w-5 h-5 text-violet-400" />
                </div>
                <div>
                    <span className="text-[15px] font-black text-white tracking-tight">Theory & Pedagogy</span>
                    <p className="text-[10px] text-neutral-500 uppercase tracking-[0.2em] font-black mt-0.5">Academic Core</p>
                </div>
            </div>

            {/* Content Swiper Style */}
            <div className="flex-1 overflow-y-auto p-6 space-y-10 scrollbar-hide pb-20 relative">
                <div className="absolute left-8 top-0 bottom-0 w-px bg-gradient-to-down from-violet-600/20 via-white/5 to-transparent pointer-events-none" />

                {/* Next Step - High Visibility */}
                {content.next_step && (
                    <div className="p-5 bg-violet-600/10 border border-violet-500/30 rounded-2xl shadow-[0_0_30px_rgba(124,58,237,0.1)]">
                        <div className="flex items-center gap-3 mb-2">
                            <Zap className="w-4 h-4 text-violet-400" />
                            <span className="text-[10px] font-black text-violet-400 uppercase tracking-[0.3em]">Objective</span>
                        </div>
                        <p className="text-[14px] font-black text-white leading-tight">
                            {content.next_step}
                        </p>
                    </div>
                )}

                {/* Core Concept */}
                {content.concept && (
                    <Section icon={Lightbulb} title="Conceptual Anchor" color="yellow">
                        <span className="font-bold text-white block mb-2">{content.concept}</span>
                        {content.theory && <p className="text-neutral-400 text-[12px] italic leading-snug">{content.theory}</p>}
                    </Section>
                )}

                {/* Logic Section */}
                {content.code_explanation && (
                    <Section icon={Code} title="Firmware Logic" color="blue">
                        <div className="whitespace-pre-wrap font-medium">
                            {content.code_explanation}
                        </div>
                    </Section>
                )}

                {/* Architecture Section */}
                {(content.hardware_explanation || content.gpio_logic) && (
                    <Section icon={Cpu} title="Silicon layer" color="emerald">
                        <div className="space-y-3">
                            {content.hardware_explanation && <p>{content.hardware_explanation}</p>}
                            {content.gpio_logic && (
                                <div className="pt-3 border-t border-white/5">
                                    <span className="text-[10px] font-black text-emerald-500 uppercase block mb-1">Pin Specification</span>
                                    <p className="text-[12px] text-neutral-400">{content.gpio_logic}</p>
                                </div>
                            )}
                        </div>
                    </Section>
                )}

                {/* Safety & Pitfalls */}
                {(content.safety_notes || content.common_mistakes) && (
                    <div className="space-y-4 pt-4 border-t border-white/5">
                        {content.safety_notes && (
                            <div className="p-4 bg-red-500/5 border border-red-500/20 rounded-2xl">
                                <div className="flex items-center gap-3 mb-2">
                                    <ShieldAlert className="w-4 h-4 text-red-400" />
                                    <span className="text-[10px] font-black text-red-400 uppercase tracking-widest">Protocol Warning</span>
                                </div>
                                <p className="text-[12px] text-red-200/70 leading-relaxed font-medium">{content.safety_notes}</p>
                            </div>
                        )}
                        {content.common_mistakes && (
                            <div className="p-4 bg-amber-500/5 border border-amber-500/20 rounded-2xl">
                                <div className="flex items-center gap-3 mb-2">
                                    <HelpCircle className="w-4 h-4 text-amber-400" />
                                    <span className="text-[10px] font-black text-amber-400 uppercase tracking-widest">Pitfall Alert</span>
                                </div>
                                <p className="text-[12px] text-amber-200/70 italic leading-relaxed font-medium">"{content.common_mistakes}"</p>
                            </div>
                        )}
                    </div>
                )}
            </div>

            {/* Footer Tag */}
            <div className="p-4 flex justify-center sticky bottom-0 bg-gradient-to-t from-black via-black/80 to-transparent">
                <div className="px-4 py-1 rounded-full glass-panel border-white/5 text-[9px] font-black text-neutral-600 uppercase tracking-[0.4em]">
                    Synchronized Pedagogy v2.1
                </div>
            </div>
        </div>
    );
};

export default LearningExplanationPanel;
