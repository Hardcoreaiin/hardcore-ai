import React, { useState, useCallback } from 'react';
import { ChevronRight, ChevronDown, Folder, FileText } from 'lucide-react';

interface ProjectExplorerProps {
    onFileClick: (path: string, content: string) => void;
    currentProject: { id: string; name: string; path: string } | null;
    files: { path: string; name: string; content: string }[];
}

const ProjectExplorer: React.FC<ProjectExplorerProps> = ({ onFileClick, files }) => {
    const [expanded, setExpanded] = useState<Set<string>>(new Set(['/current_project']));
    const [selected, setSelected] = useState<string | null>(null);

    const toggle = useCallback((path: string) => setExpanded(prev => { const n = new Set(prev); n.has(path) ? n.delete(path) : n.add(path); return n; }), []);
    const click = useCallback((path: string, content: string) => { setSelected(path); onFileClick(path, content); }, [onFileClick]);

    return (
        <div className="h-full py-2 overflow-y-auto bg-neutral-950">
            <div onClick={() => toggle('/current_project')} className="flex items-center gap-1 py-1 cursor-pointer hover:bg-neutral-900 px-2">
                {expanded.has('/current_project') ? <ChevronDown className="w-4 h-4 text-neutral-500" /> : <ChevronRight className="w-4 h-4 text-neutral-500" />}
                <Folder className="w-4 h-4 text-violet-500" />
                <span className="text-xs font-bold text-neutral-400 uppercase tracking-tighter ml-1">Current Project</span>
            </div>
            {expanded.has('/current_project') && (
                <div className="mt-1">
                    {files.map(file => (
                        <div
                            key={file.path}
                            onClick={() => click(file.path, file.content)}
                            className={`flex items-center gap-2 py-1 cursor-pointer px-4 ${selected === file.path ? 'bg-violet-900/20 text-violet-400' : 'hover:bg-neutral-900 text-neutral-400'}`}
                        >
                            <FileText className={`w-3.5 h-3.5 ${selected === file.path ? 'text-violet-400' : 'text-neutral-500'}`} />
                            <span className="text-sm truncate">{file.name}</span>
                        </div>
                    ))}
                    {files.length === 0 && (
                        <div className="px-6 py-2 text-xs text-neutral-600 italic">No files yet</div>
                    )}
                </div>
            )}
        </div>
    );
};

export default ProjectExplorer;
