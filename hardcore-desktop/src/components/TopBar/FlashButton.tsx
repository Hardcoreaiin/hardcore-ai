import React, { useState } from 'react';
import { Zap, Loader2 } from 'lucide-react';
import { useBoard } from '../../context/BoardContext';

const FlashButton: React.FC = () => {
    const { selectedBoard, connectedPort } = useBoard();
    const [isFlashing, setIsFlashing] = useState(false);

    const handleFlash = async () => {
        if (!connectedPort) {
            alert('No device connected. Please detect a device first.');
            return;
        }

        setIsFlashing(true);
        window.dispatchEvent(new CustomEvent('flash-start'));

        const processedLogs = new Set<string>();

        try {
            // 1. Start the flash process
            const initiationResponse = await fetch('http://localhost:8003/flash', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Desktop-Key': 'desktop_local_bypass_hardcore_ai'
                },
                body: JSON.stringify({
                    port: connectedPort,
                    board: selectedBoard || 'esp32dev',
                }),
            });

            const initiationData = await initiationResponse.json();
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
                        headers: { 'X-Desktop-Key': 'desktop_local_bypass_hardcore_ai' }
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

        } catch (error: any) {
            console.error('Flash error:', error);
            window.dispatchEvent(new CustomEvent('flash-log', {
                detail: { message: `[ERROR] ${error.message || 'Connection failed'}`, type: 'error' }
            }));
        } finally {
            setIsFlashing(false);
            window.dispatchEvent(new CustomEvent('flash-complete'));
        }
    };

    return (
        <button
            onClick={handleFlash}
            disabled={isFlashing}
            className="flex items-center gap-2 px-4 py-1.5 bg-accent-primary text-white rounded-lg text-sm font-medium hover:bg-accent-primary/80 disabled:opacity-50 transition-colors"
        >
            {isFlashing ? (
                <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
                <Zap className="w-4 h-4" />
            )}
            {isFlashing ? 'Flashing...' : 'Flash'}
        </button>
    );
};

export default FlashButton;
