import React from 'react';
import { BookOpen } from 'lucide-react';

const LessonMode: React.FC = () => {
    return (
        <div className="h-full flex flex-col items-center justify-center text-neutral-500 p-8 text-center bg-transparent">
            <div className="w-16 h-16 rounded-3xl bg-white/5 flex items-center justify-center mb-6 border border-white/5 shadow-inner">
                <BookOpen className="w-8 h-8 opacity-20" />
            </div>
            <p className="text-sm font-bold text-white mb-2">Lesson Mode</p>
            <p className="text-xs text-neutral-400 max-w-[200px] leading-relaxed">
                Interactive lessons and tutorials will appear here.
            </p>
        </div>
    );
};

export default LessonMode;
