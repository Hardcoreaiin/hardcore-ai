import React, { useState, useCallback } from 'react';
import { ChevronRight, ChevronDown, Folder, FileText, Cpu, Shield, Activity, ListChecks, Box } from 'lucide-react';

interface FileNode {
    name: string;
    type: 'file' | 'directory';
    children?: FileNode[];
}

interface FileExplorerProps {
    onFileClick: (name: string) => void;
    structure: FileNode | null;
    activeFile: string | null;
}

const FileExplorer: React.FC<FileExplorerProps> = ({ onFileClick, structure, activeFile }) => {
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set(['project-root']));

    const toggleFolder = (path: string) => {
        setExpandedFolders(prev => {
            const next = new Set(prev);
            if (next.has(path)) next.delete(path);
            else next.add(path);
            return next;
        });
    };

    const getFileIcon = (name: string) => {
        const lower = name.toLowerCase();
        if (lower.endsWith('.cpp') || lower.endsWith('.c')) return <Cpu className="w-3.5 h-3.5 text-blue-400" />;
        if (lower.endsWith('.h')) return <Shield className="w-3.5 h-3.5 text-amber-400" />;
        if (lower.endsWith('.dts') || lower.endsWith('.dtsi')) return <Activity className="w-3.5 h-3.5 text-emerald-400" />;
        if (lower.endsWith('.config') || lower.includes('defconfig')) return <Shield className="w-3.5 h-3.5 text-purple-400" />;
        if (lower.endsWith('.md')) return <FileText className="w-3.5 h-3.5 text-neutral-400" />;
        if (lower.includes('bom')) return <ListChecks className="w-3.5 h-3.5 text-rose-400" />;
        return <FileText className="w-3.5 h-3.5 text-neutral-500" />;
    };

    const renderTree = (node: FileNode, path: string = '') => {
        const fullPath = path ? `${path}/${node.name}` : node.name;
        const isExpanded = expandedFolders.has(fullPath);

        if (node.type === 'directory') {
            return (
                <div key={fullPath} className="select-none">
                    <div
                        onClick={() => toggleFolder(fullPath)}
                        className="flex items-center gap-2 py-1.5 px-4 cursor-pointer hover:bg-white/5 transition-colors group"
                        style={{ paddingLeft: `${(path.split('/').length) * 12 + 16}px` }}
                    >
                        {isExpanded ?
                            <ChevronDown className="w-3 h-3 text-neutral-600 group-hover:text-neutral-400" /> :
                            <ChevronRight className="w-3 h-3 text-neutral-600 group-hover:text-neutral-400" />
                        }
                        <Folder className={`w-3.5 h-3.5 ${isExpanded ? 'text-blue-500/70' : 'text-neutral-600'}`} />
                        <span className="text-[11px] font-bold text-neutral-400 uppercase tracking-tight">{node.name}</span>
                    </div>
                    {isExpanded && node.children?.map(child => renderTree(child, fullPath))}
                </div>
            );
        }

        return (
            <div
                key={fullPath}
                onClick={() => onFileClick(node.name)}
                className={`flex items-center gap-3 py-1.5 px-4 cursor-pointer transition-all ${activeFile === node.name
                    ? 'bg-blue-600/10 text-blue-400 border-r-2 border-blue-500'
                    : 'hover:bg-white/5 text-neutral-500 hover:text-neutral-300'
                    }`}
                style={{ paddingLeft: `${(path.split('/').length + 1) * 12 + 16}px` }}
            >
                {getFileIcon(node.name)}
                <span className={`text-[11px] truncate ${activeFile === node.name ? 'font-bold' : 'font-medium'}`}>
                    {node.name}
                </span>
            </div>
        );
    };

    return (
        <div className="flex flex-col h-full bg-[#0D0D0D] border-r border-white/5 overflow-hidden">
            <div className="p-4 border-b border-white/5 bg-black/40 flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <Box className="w-3.5 h-3.5 text-blue-500" />
                    <h3 className="text-[10px] font-black text-neutral-400 uppercase tracking-[0.2em]">Workspace</h3>
                </div>
            </div>

            <div className="flex-1 overflow-y-auto py-3 scrollbar-hide">
                {structure ? renderTree(structure) : (
                    <div className="p-8 text-center">
                        <span className="text-[10px] text-neutral-700 font-bold uppercase tracking-widest">No project generated</span>
                    </div>
                )}
            </div>

            <div className="p-4 border-t border-white/5 bg-black/40">
                <div className="flex items-center gap-2">
                    <div className="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_8px_#22c55e]" />
                    <span className="text-[9px] font-black text-neutral-600 uppercase tracking-[0.3em]">Project Synced</span>
                </div>
            </div>
        </div>
    );
};

export default FileExplorer;

