import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Code, Network, BookOpen, Package, GraduationCap } from 'lucide-react';
import { useAppStore } from '../../store/useAppStore';
import { useBoard } from '../../context/BoardContext';
import MonacoEditor from '../Editor/MonacoEditor';
import TheoryTab from './TheoryTab';
import PinDiagramTab from './PinDiagramTab';
import ComponentLibrary from '../RightPanel/ComponentLibrary';
import FileExplorer from '../Explorer/FileExplorer';
import { Panel, PanelGroup, PanelResizeHandle } from 'react-resizable-panels';

const LearningHub: React.FC = () => {
    const activeTab = useAppStore(state => state.activeTab);
    const generatedCode = useAppStore(state => state.generatedCode);
    const setActiveTab = useAppStore(state => state.setActiveTab);
    const setGeneratedCode = useAppStore(state => state.setGeneratedCode);
    const { selectedBoard } = useBoard();

    const architectureReport = useAppStore(state => state.architectureReport);
    const repoFiles = useAppStore(state => state.architectureReport?.files);
    const appGeneratedFiles = useAppStore(state => state.generatedFiles);
    const deviceTree = useAppStore(state => state.architectureReport?.device_tree);
    const bootloader = useAppStore(state => state.architectureReport?.bootloader);

    const generatedFiles = React.useMemo(() => {
        const files = { ...(repoFiles || {}), ...(appGeneratedFiles || {}) };
        
        if (deviceTree) {
            const { file, content } = deviceTree;
            if (file && content && !files[file]) {
                files[file] = content;
            }
        }
        
        if (bootloader) {
            const { config_file, config_content } = bootloader;
            if (config_file && config_content && !files[config_file]) {
                files[config_file] = config_content;
            }
        }
        
        return files;
    }, [repoFiles, appGeneratedFiles, deviceTree, bootloader]);

    const [activeFile, setActiveFile] = React.useState<string | null>(null);
    const [localFiles, setLocalFiles] = React.useState<Record<string, string>>(generatedFiles);

    // Sync generated files to local state when they change
    React.useEffect(() => {
        setLocalFiles(generatedFiles);
        // Set first file as active if none selected
        if (!activeFile && Object.keys(generatedFiles).length > 0) {
            setActiveFile(Object.keys(generatedFiles)[0]);
        }
    }, [generatedFiles]);

    const handleFileChange = (content: string | undefined) => {
        if (activeFile && content !== undefined) {
            setLocalFiles(prev => ({ ...prev, [activeFile]: content }));
        }
    };

    const getLanguage = (filename: string) => {
        if (filename.endsWith('.cpp') || filename.endsWith('.c') || filename.endsWith('.h')) return 'cpp';
        if (filename.endsWith('.dts') || filename.endsWith('.dtsi')) return 'cpp'; // Close enough for highlighting
        if (filename.endsWith('.json')) return 'json';
        if (filename.endsWith('.md')) return 'markdown';
        return 'text';
    };

    // Memoize the final structure to avoid flickering and ensure all files are visible
    const patchedStructure = React.useMemo(() => {
        if (!architectureReport?.project_structure) {
            return {
                name: 'project-root',
                type: 'directory' as const,
                children: Object.keys(localFiles).map(name => ({ name, type: 'file' as const }))
            };
        }

        const structure = { ...architectureReport.project_structure };
        const allFileNames = new Set<string>();
        
        // Helper to find all files in the tree
        const findFiles = (node: any) => {
            if (node.type === 'file') allFileNames.add(node.name);
            if (node.children) node.children.forEach(findFiles);
        };
        findFiles(structure);

        // Add missing files to the root children
        const missingFiles = Object.keys(localFiles).filter(name => !allFileNames.has(name));
        if (missingFiles.length > 0) {
            structure.children = [
                ...(structure.children || []),
                ...missingFiles.map(name => ({ name, type: 'file' as const }))
            ];
        }

        return structure;
    }, [architectureReport?.project_structure, localFiles]);

    // No internal tabs anymore, LearningHub is the "Firmware" tab content
    const renderIDE = () => {
        return (
            <div className="h-full w-full">
                <PanelGroup direction="horizontal">
                    <Panel defaultSize={20} minSize={15} maxSize={30} className="bg-neutral-900/50 border-r border-white/5">
                        <FileExplorer
                            structure={patchedStructure}
                            activeFile={activeFile}
                            onFileClick={(name) => setActiveFile(name)}
                        />
                    </Panel>
                    <PanelResizeHandle className="w-1 bg-white/5 hover:bg-blue-500/30 transition-colors" />
                    <Panel>
                        <MonacoEditor
                            value={activeFile ? localFiles[activeFile] : ''}
                            onChange={handleFileChange}
                            language={activeFile ? getLanguage(activeFile) : 'cpp'}
                            openFiles={Object.keys(localFiles).map(name => ({
                                path: name,
                                name: name,
                                content: localFiles[name]
                            }))}
                            activeFilePath={activeFile || undefined}
                            onTabClick={(path) => setActiveFile(path)}
                            onTabClose={(path) => {
                                // Close tab logic if we want to support it, but for now just switching is fine
                            }}
                        />
                    </Panel>
                </PanelGroup>
            </div>
        );
    };

    return (
        <div className="h-full flex flex-col bg-transparent">
            {/* IDE Workspace Area */}
            <div className="flex-1 relative overflow-hidden">
                {renderIDE()}
            </div>
        </div>
    );
};

export default LearningHub;
