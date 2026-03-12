import React, { useState, useCallback } from 'react';
import { ChevronRight, ChevronDown, Folder, FileText, Cpu, Shield, Activity, ListChecks } from 'lucide-react';

interface FileExplorerProps {
    onFileClick: (path: string, content: string) => void;
    files: { path: string; name: string; content: string }[];
    activePath: string | null;
}

const FileExplorer: React.FC<FileExplorerProps> = ({ onFileClick, files, activePath }) => {
    const [expanded, setExpanded] = useState<Set<string>>(new Set(['/project']));

    const toggle = useCallback((path: string) => {
        setExpanded(prev => {
            const n = new Set(prev);
            if (n.has(path)) n.delete(path);
            else n.add(path);
            return n;
        });
    }, []);

    const getFileIcon = (name: string) => {
        if (name.endsWith('.cpp') || name.endsWith('.c')) return <Cpu className="w-3.5 h-3.5 text-blue-400" />;
        if (name.endsWith('.h')) return <Shield className="w-3.5 h-3.5 text-amber-400" />;
        if (name.endsWith('.md')) return <FileText className="w-3.5 h-3.5 text-emerald-400" />;
        if (name.includes('bom')) return <ListChecks className="w-3.5 h-3.5 text-rose-400" />;
        if (name.includes('risk') || name.includes('circuitry')) return <Activity className="w-3.5 h-3.5 text-orange-400" />;
        return <FileText className="w-3.5 h-3.5 text-neutral-500" />;
    };

    return (
        <div className="flex flex-col h-full bg-neutral-900/50 backdrop-blur-xl border-r border-white/5">
            <div className="p-4 border-b border-white/5">
                <h3 className="text-[10px] font-black text-neutral-500 uppercase tracking-[0.2em]">Project Explorer</h3>
            </div>

            <div className="flex-1 overflow-y-auto py-2">
                <div
                    onClick={() => toggle('/project')}
                    className="flex items-center gap-2 py-2 px-4 cursor-pointer hover:bg-white/5 transition-colors group"
                >
                    {expanded.has('/project') ?
                        <ChevronDown className="w-3.5 h-3.5 text-neutral-600 group-hover:text-neutral-400" /> :
                        <ChevronRight className="w-3.5 h-3.5 text-neutral-600 group-hover:text-neutral-400" />
                    }
                    <Folder className="w-4 h-4 text-blue-500/70" />
                    <span className="text-[11px] font-bold text-neutral-400 uppercase tracking-tight">Industrial Project</span>
                </div>

                {expanded.has('/project') && (
                    <div className="mt-1">
                        {files.map(file => (
                            <div
                                key={file.path}
                                onClick={() => onFileClick(file.path, file.content)}
                                className={`flex items-center gap-3 py-2 px-8 cursor-pointer transition-all ${activePath === file.path
                                        ? 'bg-blue-600/10 text-blue-400 border-r-2 border-blue-500'
                                        : 'hover:bg-white/5 text-neutral-500 hover:text-neutral-300'
                                    }`}
                            >
                                {getFileIcon(file.name)}
                                <span className={`text-xs truncate ${activePath === file.path ? 'font-bold' : 'font-medium'}`}>
                                    {file.name}
                                </span>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            <div className="p-4 border-t border-white/5 bg-black/20">
                <div className="flex items-center gap-2">
                    <div className="w-1.5 h-1.5 rounded-full bg-blue-500 shadow-[0_0_8px_#3b82f6]" />
                    <span className="text-[9px] font-black text-neutral-600 uppercase tracking-widest">Workspace Synced</span>
                </div>
            </div>
        </div>
    );
};

export default FileExplorer;
