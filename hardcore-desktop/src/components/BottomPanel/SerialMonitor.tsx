import React, { useState, useEffect, useRef } from 'react';
import { Terminal, Settings, Play, Square, Trash2, RefreshCw, Clock } from 'lucide-react';

interface SerialMessage {
    id: number;
    text: string;
    type: 'rx' | 'tx' | 'info';
    timestamp: string;
}

const BAUD_RATES = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600];

const API_HEADERS = {
    'Content-Type': 'application/json',
    'X-Desktop-Key': 'desktop_local_bypass_hardcore_ai',
};

const SerialMonitor: React.FC = () => {
    const [messages, setMessages] = useState<SerialMessage[]>([]);
    const [input, setInput] = useState('');
    const [isConnected, setIsConnected] = useState(false);
    const [isConnecting, setIsConnecting] = useState(false);
    const [ports, setPorts] = useState<{ port: string; description: string }[]>([]);
    const [selectedPort, setSelectedPort] = useState('');
    const [baudRate, setBaudRate] = useState(115200);
    const [autoScroll, setAutoScroll] = useState(true);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const pollingRef = useRef<NodeJS.Timeout | null>(null);

    useEffect(() => {
        fetchPorts();
        return () => {
            if (pollingRef.current) clearInterval(pollingRef.current);
        };
    }, []);

    useEffect(() => {
        if (autoScroll) {
            messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
        }
    }, [messages, autoScroll]);

    const fetchPorts = async () => {
        try {
            const res = await fetch('http://localhost:8003/serial/ports', { headers: API_HEADERS });
            const data = await res.json();
            if (data.status === 'success') {
                const availablePorts = data.all_ports || [];
                setPorts(availablePorts);
                if (availablePorts.length > 0 && !selectedPort) {
                    setSelectedPort(availablePorts[0].port);
                }
            }
        } catch (err) {
            console.error('Failed to fetch ports:', err);
        }
    };

    const toggleConnection = async () => {
        if (isConnected) {
            try {
                await fetch('http://localhost:8003/serial/stop', {
                    method: 'POST',
                    headers: API_HEADERS,
                    body: JSON.stringify({ project_id: 1 })
                });
                setIsConnected(false);
                if (pollingRef.current) clearInterval(pollingRef.current);
                addMessage('Serial connection closed.', 'info');
            } catch (err) {
                console.error('Failed to stop serial:', err);
            }
        } else {
            if (!selectedPort) {
                addMessage('Please select a COM port first.', 'info');
                return;
            }
            setIsConnecting(true);
            try {
                const res = await fetch('http://localhost:8003/serial/start', {
                    method: 'POST',
                    headers: API_HEADERS,
                    body: JSON.stringify({
                        project_id: 1,
                        port: selectedPort,
                        baud_rate: baudRate
                    })
                });
                const data = await res.json();
                if (data.status === 'success') {
                    setIsConnected(true);
                    addMessage(`Connected to ${selectedPort} at ${baudRate} baud.`, 'info');
                    startPolling();
                } else {
                    addMessage(`Error: ${data.error || 'Failed to connect'}`, 'rx');
                }
            } catch (err) {
                addMessage(`Connection failed: ${err}`, 'rx');
            } finally {
                setIsConnecting(false);
            }
        }
    };

    const startPolling = () => {
        if (pollingRef.current) clearInterval(pollingRef.current);
        pollingRef.current = setInterval(readSerial, 500);
    };

    const readSerial = async () => {
        try {
            const res = await fetch('http://localhost:8003/serial/read/1', { headers: API_HEADERS });
            const data = await res.json();
            if (data.status === 'success' && data.lines && data.lines.length > 0) {
                const newMsgs = data.lines.map((line: string) => ({
                    id: Date.now() + Math.random(),
                    text: line,
                    type: 'rx',
                    timestamp: new Date().toLocaleTimeString()
                }));
                setMessages(prev => [...prev, ...newMsgs].slice(-1000));
            }
        } catch (err) {
            console.error('Polling error:', err);
        }
    };

    const addMessage = (text: string, type: 'rx' | 'tx' | 'info') => {
        setMessages(prev => [...prev, {
            id: Date.now(),
            text,
            type,
            timestamp: new Date().toLocaleTimeString()
        }].slice(-1000));
    };

    const handleSend = () => {
        if (!input.trim() || !isConnected) return;
        // In a real implementation, we would send this to the backend
        // For now, we just echo it locally
        addMessage(input, 'tx');
        setInput('');
    };

    const handleClear = () => setMessages([]);

    return (
        <div className="h-full flex flex-col bg-black text-neutral-300 font-mono text-xs overflow-hidden">
            {/* Professional Toolbar */}
            <div className="flex items-center gap-4 px-4 py-2 bg-neutral-900 border-b border-neutral-800">
                <div className="flex items-center gap-3">
                    <select
                        value={selectedPort}
                        onChange={(e) => setSelectedPort(e.target.value)}
                        className="bg-neutral-800 border border-neutral-700 rounded px-2 py-1 focus:outline-none focus:border-violet-500"
                    >
                        {ports.length === 0 ? <option value="">No Ports Found</option> : null}
                        {ports.map(p => (
                            <option key={p.port} value={p.port}>{p.port} ({p.description})</option>
                        ))}
                    </select>

                    <select
                        value={baudRate}
                        onChange={(e) => setBaudRate(Number(e.target.value))}
                        className="bg-neutral-800 border border-neutral-700 rounded px-2 py-1 focus:outline-none focus:border-violet-500"
                    >
                        {BAUD_RATES.map(b => (
                            <option key={b} value={b}>{b} baud</option>
                        ))}
                    </select>

                    <button
                        onClick={toggleConnection}
                        disabled={isConnecting}
                        className={`flex items-center gap-1.5 px-3 py-1 rounded transition-all font-medium ${isConnected
                            ? 'bg-red-500/20 text-red-500 hover:bg-red-500/30'
                            : 'bg-violet-500 text-white hover:bg-violet-600'
                            }`}
                    >
                        {isConnected ? <Square size={14} fill="currentColor" /> : <Play size={14} fill="currentColor" />}
                        {isConnecting ? 'Connecting...' : (isConnected ? 'Disconnect' : 'Connect')}
                    </button>
                </div>

                <div className="flex-1" />

                <div className="flex items-center gap-4">
                    <label className="flex items-center gap-2 cursor-pointer opacity-70 hover:opacity-100 transition-opacity">
                        <input
                            type="checkbox"
                            checked={autoScroll}
                            onChange={(e) => setAutoScroll(e.target.checked)}
                            className="w-3 h-3 accent-violet-500"
                        />
                        <span>Autoscroll</span>
                    </label>

                    <button
                        onClick={fetchPorts}
                        className="p-1 hover:bg-neutral-800 rounded transition-colors"
                        title="Refresh Ports"
                    >
                        <RefreshCw size={14} />
                    </button>

                    <button
                        onClick={handleClear}
                        className="p-1 hover:bg-neutral-800 rounded transition-colors"
                        title="Clear Console"
                    >
                        <Trash2 size={14} />
                    </button>
                </div>
            </div>

            {/* Monitor Output */}
            <div className="flex-1 overflow-y-auto p-4 space-y-0.5 selection:bg-violet-500/30">
                {messages.length === 0 ? (
                    <div className="h-full flex flex-col items-center justify-center text-neutral-600">
                        <Terminal size={32} className="mb-2 opacity-20" />
                        <p>No serial output. Select a port and connect.</p>
                    </div>
                ) : (
                    messages.map((msg) => (
                        <div key={msg.id} className="flex gap-3 leading-relaxed group">
                            <span className="text-neutral-600 opacity-40 group-hover:opacity-100 transition-opacity flex items-center gap-1 whitespace-nowrap">
                                <Clock size={10} /> {msg.timestamp}
                            </span>
                            <span className={`flex-1 break-all ${msg.type === 'tx' ? 'text-blue-400' :
                                msg.type === 'info' ? 'text-violet-400 italic' :
                                    'text-green-400'
                                }`}>
                                {msg.type === 'tx' && <span className="text-blue-600 mr-2">TX ❯</span>}
                                {msg.text}
                            </span>
                        </div>
                    ))
                )}
                <div ref={messagesEndRef} />
            </div>

            {/* Input Field */}
            <div className="p-3 bg-neutral-950 border-t border-neutral-900 flex gap-2">
                <div className="relative flex-1">
                    <input
                        type="text"
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleSend()}
                        disabled={!isConnected}
                        placeholder={isConnected ? "Send message to device..." : "Connect to send messages"}
                        className="w-full bg-neutral-900 border border-neutral-800 rounded px-4 py-1.5 focus:outline-none focus:border-violet-500 disabled:opacity-50"
                    />
                    <div className="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] text-neutral-600 hidden md:block">
                        Press ENTER to send
                    </div>
                </div>
                <button
                    onClick={handleSend}
                    disabled={!isConnected}
                    className="bg-neutral-800 hover:bg-neutral-700 disabled:opacity-30 disabled:hover:bg-neutral-800 text-neutral-300 px-4 py-1.5 rounded transition-colors font-medium"
                >
                    Send
                </button>
            </div>
        </div>
    );
};

export default SerialMonitor;
