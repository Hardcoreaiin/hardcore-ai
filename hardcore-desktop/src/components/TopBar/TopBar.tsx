import React, { useState, useRef } from 'react';
import { createPortal } from 'react-dom';
import { useBoard } from '../../context/BoardContext';
import { useAuth } from '../../context/AuthContext';
import { useAppState } from '../../context/AppStateContext';
import { Cpu, Wifi, Zap, ChevronDown, LogOut, Loader2, Settings, GraduationCap, Trophy } from 'lucide-react';
import SettingsModal from '../Settings/SettingsModal';

// Desktop bypass header for local authentication
const API_HEADERS = {
    'Content-Type': 'application/json',
    'X-Desktop-Key': 'desktop_local_bypass_hardcore_ai',
};

interface TopBarProps {
    onPortChange?: (port: string | null) => void;
    onBoardChange?: (board: string) => void;
    currentCode?: string;
}

const boards = [
    { id: 'esp32', name: 'ESP32 DevKit' },
    { id: 'esp32-s3', name: 'ESP32-S3' },
    { id: 'uno', name: 'Arduino Uno' },
    { id: 'nano', name: 'Arduino Nano' },
    { id: 'mega', name: 'Arduino Mega' },
];

const TopBar: React.FC<TopBarProps> = ({ onPortChange, onBoardChange, currentCode }) => {
    const { selectedBoard, setSelectedBoard, connectedPort, setConnectedPort } = useBoard();
    const { setActiveTab } = useAppState();
    const { user, logout } = useAuth();
    const [showProfile, setShowProfile] = useState(false);
    const [showBoards, setShowBoards] = useState(false);
    const [showWireless, setShowWireless] = useState(false);
    const [detecting, setDetecting] = useState(false);
    const [flashing, setFlashing] = useState(false);
    const [showSettings, setShowSettings] = useState(false);
    const profileRef = useRef<HTMLButtonElement>(null);
    const boardRef = useRef<HTMLButtonElement>(null);
    const wirelessRef = useRef<HTMLButtonElement>(null);

    const detect = async () => {
        setDetecting(true);
        setShowBoards(false);
        try {
            const res = await fetch('http://localhost:8003/hardware/ports', { headers: API_HEADERS });
            const data = await res.json();
            if (data.ports?.length) {
                const device = data.ports[0];
                onPortChange?.(device.port);
                setConnectedPort(device.port);
                await fetch('http://localhost:8003/hardware/connect', {
                    method: 'POST',
                    headers: API_HEADERS,
                    body: JSON.stringify({ port: device.port })
                });
                alert(`Connected to ${device.port} (${device.description})`);
            } else {
                alert('No devices detected. Connect a board and try again.');
            }
        } catch (e) {
            alert('Backend not running. Start the backend server first.');
        }
        setDetecting(false);
    };

    const flash = async () => {
        if (!connectedPort) {
            alert('No device connected. Please detect a device first.');
            return;
        }

        setFlashing(true);
        setActiveTab('output'); // Switch to build output automatically
        window.dispatchEvent(new CustomEvent('flash-start'));

        const processedLogs = new Set<string>();

        try {
            // 1. Start the flash process
            const response = await fetch('http://localhost:8003/hardware/flash', {
                method: 'POST',
                headers: API_HEADERS,
                body: JSON.stringify({
                    port: connectedPort,
                    project_id: 1,
                    code: currentCode
                }),
            });
            const initiationData = await response.json();

            if (!initiationData.success) {
                throw new Error(initiationData.error || 'Failed to start build');
            }

            // 2. Poll for logs until build is inactive
            let isBuildActive = true;
            let retryCount = 0;

            while (isBuildActive && retryCount < 180) { // 3 minute timeout
                await new Promise(r => setTimeout(r, 1000));

                try {
                    const logResponse = await fetch('http://localhost:8003/flash/logs', {
                        headers: API_HEADERS
                    });
                    const logData = await logResponse.json();

                    if (logData.logs) {
                        logData.logs.forEach((log: any) => {
                            const logKey = `v1-${log.id}`;
                            if (!processedLogs.has(logKey)) {
                                window.dispatchEvent(new CustomEvent('flash-log', {
                                    detail: {
                                        message: log.message,
                                        timestamp: new Date(log.timestamp * 1000)
                                    }
                                }));
                                processedLogs.add(logKey);
                            }
                        });
                    }

                    isBuildActive = logData.active;
                } catch (e) {
                    console.error('Polling error:', e);
                }
                retryCount++;
            }

        } catch (e) {
            console.error('Flash error:', e);
            window.dispatchEvent(new CustomEvent('flash-log', {
                detail: { message: `[CRITICAL ERROR] ${String(e)}`, type: 'error' }
            }));
            alert(`Flash Failed: ${String(e)}`);
        } finally {
            setFlashing(false);
            window.dispatchEvent(new CustomEvent('flash-complete'));
        }
    };

    const selectBoard = (id: string) => {
        setSelectedBoard(id);
        onBoardChange?.(id);
        setShowBoards(false);
    };

    const btn = "flex items-center gap-2 px-3 py-1.5 text-xs border border-neutral-800 bg-black text-neutral-400 hover:bg-neutral-900 hover:text-neutral-300 hover:border-blue-900 transition-colors";
    const btnActive = "flex items-center gap-2 px-3 py-1.5 text-xs border border-blue-700 bg-blue-950 text-blue-300 hover:bg-blue-900 transition-colors";

    const boardName = boards.find(b => b.id === selectedBoard)?.name || selectedBoard || 'Detect Board';

    return (
        <div className="h-10 flex flex-col border-b border-neutral-900 bg-black relative">
            {flashing && (
                <div className="absolute top-0 left-0 h-[2px] bg-blue-500 z-50 animate-pulse" style={{ width: '100%', transition: 'width 2s ease-in-out' }} />
            )}

            <div className="flex-1 flex items-center justify-between px-4">
                <div className="text-[14px] font-black text-white tracking-widest uppercase">Hardcore<span className="text-blue-500">AI</span> <span className="text-neutral-600 font-medium">Architect</span></div>

                <div className="flex items-center gap-2">
                    <button ref={boardRef} onClick={() => setShowBoards(!showBoards)} className={connectedPort ? btnActive : btn}>
                        <Cpu className={`w-4 h-4 ${connectedPort ? 'text-blue-400' : ''}`} />
                        <span>{detecting ? 'Detecting...' : connectedPort ? `${boardName} (${connectedPort})` : 'Detect Hardware'}</span>
                        <ChevronDown className="w-3 h-3 opacity-50" />
                    </button>
                    {showBoards && createPortal(
                        <>
                            <div className="fixed inset-0 z-50" onClick={() => setShowBoards(false)} />
                            <div className="fixed z-50 w-56 border border-neutral-800 bg-black shadow-xl py-1" style={{ top: (boardRef.current?.getBoundingClientRect().bottom || 0) + 4, left: boardRef.current?.getBoundingClientRect().left || 0 }}>
                                <button onClick={detect} disabled={detecting} className="w-full text-left px-3 py-2 text-xs text-blue-400 hover:bg-blue-950 border-b border-neutral-800 flex items-center gap-2">
                                    {detecting ? <Loader2 className="w-3 h-3 animate-spin" /> : <Zap className="w-3 h-3" />}
                                    {detecting ? 'Detecting...' : 'Scan Ports'}
                                </button>
                                {connectedPort && (
                                    <div className="px-3 py-2 text-xs text-blue-400 bg-blue-950/50 border-b border-neutral-800">
                                        ✓ Linked: {boardName} on {connectedPort}
                                    </div>
                                )}
                                <div className="py-1 text-xs text-neutral-600 px-3">Or select manually:</div>
                                {boards.map(b => (
                                    <button key={b.id} onClick={() => selectBoard(b.id)} className={`w-full text-left px-3 py-2 text-xs hover:bg-neutral-900 ${selectedBoard === b.id ? 'text-blue-400' : 'text-neutral-500'}`}>{b.name}</button>
                                ))}
                            </div>
                        </>,
                        document.body
                    )}

                    <button ref={wirelessRef} onClick={() => setShowWireless(!showWireless)} className={btn}>
                        <Wifi className="w-4 h-4" /> Wireless <ChevronDown className="w-3 h-3 opacity-50" />
                    </button>
                    {showWireless && createPortal(
                        <>
                            <div className="fixed inset-0 z-50" onClick={() => setShowWireless(false)} />
                            <div className="fixed z-50 w-48 border border-neutral-700 bg-neutral-900 rounded shadow-xl py-1" style={{ top: (wirelessRef.current?.getBoundingClientRect().bottom || 0) + 4, left: wirelessRef.current?.getBoundingClientRect().left || 0 }}>
                                <button className="w-full text-left px-3 py-2 text-sm text-neutral-300 hover:bg-neutral-800">OTA Update</button>
                                <button className="w-full text-left px-3 py-2 text-sm text-neutral-300 hover:bg-neutral-800">WiFi Config</button>
                                <button className="w-full text-left px-3 py-2 text-sm text-neutral-300 hover:bg-neutral-800">Bluetooth</button>
                            </div>
                        </>,
                        document.body
                    )}
                </div>

                <div className="flex items-center gap-2">

                    <button onClick={flash} disabled={flashing} className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium bg-neutral-100 text-neutral-900 rounded hover:bg-neutral-200 transition-colors disabled:opacity-50">
                        {flashing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />} Flash
                    </button>
                    <button ref={profileRef} onClick={() => setShowProfile(!showProfile)} className="flex items-center gap-2 px-3 py-1.5 text-sm bg-neutral-800 text-neutral-200 rounded-full hover:bg-neutral-700 transition-colors">
                        <div className="w-5 h-5 rounded-full bg-neutral-600 flex items-center justify-center text-xs font-medium">{user?.name?.charAt(0) || 'D'}</div>
                        {user?.name || 'Demo User'} <ChevronDown className="w-3 h-3 opacity-50" />
                    </button>
                    {showProfile && createPortal(
                        <>
                            <div className="fixed inset-0 z-50" onClick={() => setShowProfile(false)} />
                            <div className="fixed z-50 w-44 border border-neutral-700 bg-neutral-900 rounded shadow-xl py-1" style={{ top: (profileRef.current?.getBoundingClientRect().bottom || 0) + 4, right: window.innerWidth - (profileRef.current?.getBoundingClientRect().right || 0) }}>
                                <div className="px-3 py-2 border-b border-neutral-800">
                                    <div className="text-sm text-neutral-100">{user?.name}</div>
                                    <div className="text-xs text-neutral-500">{user?.email}</div>
                                </div>
                                <button onClick={() => { setShowSettings(true); setShowProfile(false); }} className="w-full flex items-center gap-2 px-3 py-2 text-sm text-neutral-400 hover:bg-neutral-800"><Settings className="w-4 h-4" /> Settings</button>
                                <button onClick={() => { logout(); setShowProfile(false); }} className="w-full flex items-center gap-2 px-3 py-2 text-sm text-neutral-400 hover:bg-neutral-800"><LogOut className="w-4 h-4" /> Sign Out</button>
                            </div>
                        </>,
                        document.body
                    )}
                </div>
            </div>
            <SettingsModal isOpen={showSettings} onClose={() => setShowSettings(false)} />
        </div>
    );
};

export default TopBar;
