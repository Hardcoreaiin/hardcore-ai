import React, { useState, useEffect, useRef } from 'react';
import { Upload, FileText, Trash2, Loader2, CheckCircle2, AlertCircle } from 'lucide-react';

const API_HEADERS = {
    'X-Desktop-Key': 'desktop_local_bypass_hardcore_ai',
};

interface ComponentSummary {
    name: string;
    type: string;
    source_file: string;
    interfaces: string[];
    pin_count: number;
    file: string;
}

interface DatasheetUploadProps {
    projectId?: string;
}

const DatasheetUpload: React.FC<DatasheetUploadProps> = ({ projectId = 'current_project' }) => {
    const [components, setComponents] = useState<ComponentSummary[]>([]);
    const [uploading, setUploading] = useState(false);
    const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
    const fileInputRef = useRef<HTMLInputElement>(null);

    // Load components on mount
    useEffect(() => {
        loadComponents();
    }, [projectId]);

    const loadComponents = async () => {
        try {
            const res = await fetch(`http://localhost:8003/datasheet/profiles/${projectId}`, {
                headers: API_HEADERS,
            });
            const data = await res.json();
            if (data.status === 'success') {
                setComponents(data.profiles || []);
            }
        } catch (e) {
            console.error('Failed to load components:', e);
        }
    };

    const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        if (!file.name.toLowerCase().endsWith('.pdf')) {
            setMessage({ type: 'error', text: 'Only PDF files are supported' });
            return;
        }

        setUploading(true);
        setMessage(null);

        const formData = new FormData();
        formData.append('file', file);
        formData.append('project_id', projectId);

        try {
            const res = await fetch('http://localhost:8003/datasheet/upload', {
                method: 'POST',
                headers: API_HEADERS,
                body: formData,
            });

            const data = await res.json();

            if (res.ok && data.status === 'success') {
                setMessage({ type: 'success', text: `${data.profile.name} extracted successfully!` });
                loadComponents();
            } else {
                setMessage({ type: 'error', text: data.detail || 'Failed to parse datasheet' });
            }
        } catch (e) {
            setMessage({ type: 'error', text: 'Backend not running or network error' });
        }

        setUploading(false);
        if (fileInputRef.current) {
            fileInputRef.current.value = '';
        }
    };

    const handleDelete = async (name: string) => {
        try {
            const res = await fetch(`http://localhost:8003/datasheet/profile/${projectId}/${name}`, {
                method: 'DELETE',
                headers: API_HEADERS,
            });
            if (res.ok) {
                loadComponents();
            }
        } catch (e) {
            console.error('Failed to delete:', e);
        }
    };

    return (
        <div className="p-3 space-y-3">
            {/* Upload Button */}
            <div className="flex items-center gap-2">
                <label className="flex items-center gap-2 px-3 py-2 text-sm bg-neutral-800 hover:bg-neutral-700 text-neutral-200 rounded cursor-pointer transition-colors">
                    {uploading ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                        <Upload className="w-4 h-4" />
                    )}
                    <span>{uploading ? 'Parsing...' : 'Upload Datasheet'}</span>
                    <input
                        ref={fileInputRef}
                        type="file"
                        accept=".pdf"
                        onChange={handleUpload}
                        className="hidden"
                        disabled={uploading}
                    />
                </label>
                <span className="text-xs text-neutral-500">PDF datasheets only</span>
            </div>

            {/* Message */}
            {message && (
                <div className={`flex items-center gap-2 px-3 py-2 rounded text-sm ${message.type === 'success'
                        ? 'bg-green-900/40 text-green-300'
                        : 'bg-red-900/40 text-red-300'
                    }`}>
                    {message.type === 'success' ? (
                        <CheckCircle2 className="w-4 h-4" />
                    ) : (
                        <AlertCircle className="w-4 h-4" />
                    )}
                    {message.text}
                </div>
            )}

            {/* Component List */}
            {components.length > 0 && (
                <div className="space-y-2">
                    <div className="text-xs text-neutral-500 uppercase tracking-wider">
                        Uploaded Components ({components.length})
                    </div>
                    <div className="space-y-1">
                        {components.map((c) => (
                            <div
                                key={c.name}
                                className="flex items-center justify-between px-3 py-2 bg-neutral-900 border border-neutral-800 rounded"
                            >
                                <div className="flex items-center gap-2">
                                    <FileText className="w-4 h-4 text-neutral-500" />
                                    <div>
                                        <div className="text-sm text-neutral-200 flex items-center gap-2">
                                            {c.name}
                                            <span className="px-1.5 py-0.5 text-xs bg-neutral-700 text-neutral-300 rounded">
                                                {c.type}
                                            </span>
                                        </div>
                                        <div className="text-xs text-neutral-500">
                                            {c.pin_count} pins • {c.interfaces.join(', ') || 'No interfaces'}
                                        </div>
                                    </div>
                                </div>
                                <button
                                    onClick={() => handleDelete(c.name)}
                                    className="p-1.5 text-neutral-500 hover:text-red-400 hover:bg-neutral-800 rounded"
                                    title="Remove component"
                                >
                                    <Trash2 className="w-4 h-4" />
                                </button>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Empty State */}
            {components.length === 0 && !uploading && (
                <div className="text-center py-6 text-neutral-500 text-sm">
                    <FileText className="w-8 h-8 mx-auto mb-2 opacity-50" />
                    <div>No component datasheets uploaded</div>
                    <div className="text-xs mt-1">Upload a PDF to enable datasheet-driven generation</div>
                </div>
            )}
        </div>
    );
};

export default DatasheetUpload;
