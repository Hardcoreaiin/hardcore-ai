import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Loader2, AlertTriangle, Cpu, ZoomIn, ZoomOut, Maximize2 } from 'lucide-react';
import { useAppStore, PinInfo } from '../../store/useAppStore';

// ─── ESP32 DevKit Pin Layout (physical board) ──────────────────────────
const LEFT_PINS = [
    { gpio: 'EN', label: 'EN' },
    { gpio: 36, label: 'VP / 36' },
    { gpio: 39, label: 'VN / 39' },
    { gpio: 34, label: '34' },
    { gpio: 35, label: '35' },
    { gpio: 32, label: '32' },
    { gpio: 33, label: '33' },
    { gpio: 25, label: '25' },
    { gpio: 26, label: '26' },
    { gpio: 27, label: '27' },
    { gpio: 14, label: '14' },
    { gpio: 12, label: '12' },
    { gpio: 13, label: '13' },
    { gpio: 'GND1', label: 'GND' },
    { gpio: 'VIN', label: 'VIN' },
];

const RIGHT_PINS = [
    { gpio: '3V3', label: '3V3' },
    { gpio: 23, label: '23' },
    { gpio: 22, label: '22' },
    { gpio: 1, label: 'TX0' },
    { gpio: 3, label: 'RX0' },
    { gpio: 21, label: '21' },
    { gpio: 19, label: '19' },
    { gpio: 18, label: '18' },
    { gpio: 5, label: '5' },
    { gpio: 17, label: '17' },
    { gpio: 16, label: '16' },
    { gpio: 4, label: '4' },
    { gpio: 2, label: '2' },
    { gpio: 15, label: '15' },
    { gpio: 'GND2', label: 'GND' },
];

const MODE_COLORS: Record<string, string> = {
    OUTPUT: '#22c55e', PWM_OUTPUT: '#f59e0b', INPUT: '#3b82f6',
    INPUT_PULLUP: '#60a5fa', INPUT_PULLDOWN: '#93c5fd', ANALOG_INPUT: '#a78bfa',
    DAC_OUTPUT: '#f97316', I2C_SDA: '#ec4899', I2C_SCL: '#ec4899',
    SPI_CLK: '#14b8a6', SPI_MOSI: '#14b8a6', SPI_MISO: '#14b8a6', SPI_SS: '#14b8a6',
    UART_TX: '#f43f5e', UART_RX: '#fb923c', TOUCH_INPUT: '#84cc16',
};
const MODE_LABELS: Record<string, string> = {
    OUTPUT: 'OUTPUT', PWM_OUTPUT: 'PWM', INPUT: 'INPUT', INPUT_PULLUP: 'INPUT↑',
    INPUT_PULLDOWN: 'INPUT↓', ANALOG_INPUT: 'ANALOG', DAC_OUTPUT: 'DAC',
    I2C_SDA: 'I2C SDA', I2C_SCL: 'I2C SCL', SPI_CLK: 'SPI CLK', SPI_MOSI: 'SPI MOSI',
    SPI_MISO: 'SPI MISO', SPI_SS: 'SPI SS', UART_TX: 'UART TX', UART_RX: 'UART RX',
    TOUCH_INPUT: 'TOUCH',
};
const COMPONENT_ICONS: Record<string, string> = {
    'LED': '💡', 'LED / Output Device': '💡', 'RGB LED': '🌈', 'Buzzer': '🔊',
    'Button': '🔘', 'Button / Sensor': '🔘', 'Servo Motor': '⚙️', 'Servo': '⚙️',
    'Potentiometer': '🎛️', 'Analog Sensor': '📡', 'PIR Motion Sensor': '👁️',
    'Ultrasonic Sensor': '📏', 'DHT Sensor': '🌡️', 'Temperature Sensor': '🌡️',
    'Light Sensor (LDR)': '☀️', 'Touch Sensor': '👆', 'Relay': '⚡',
    'DC Motor': '🔄', 'LCD Display': '🖥️', 'OLED Display': '🖥️', 'Display': '🖥️',
    'USB Serial (TX)': '🖥️', 'USB Serial (RX)': '🖥️',
    'I2C Bus (SDA)': '🔗', 'I2C Bus (SCL)': '🔗',
    'SPI Bus (CLK)': '🔗', 'SPI Bus (MOSI)': '🔗', 'SPI Bus (MISO)': '🔗', 'SPI Bus (SS)': '🔗',
};

const BOARD_W = 160, BOARD_H = 480, BOARD_X = 320, BOARD_Y = 30;
const PIN_GAP = 30, PIN_LEN = 20, SVG_W = 900, SVG_H = BOARD_Y * 2 + BOARD_H + 20;

const PinDiagramTab: React.FC = () => {
    const pinAnalysis = useAppStore(state => state.pinAnalysis);
    const setPinAnalysis = useAppStore(state => state.setPinAnalysis);
    const generatedCode = useAppStore(state => state.generatedCode);
    const [analyzing, setAnalyzing] = useState(false);
    const [hoveredGpio, setHoveredGpio] = useState<number | string | null>(null);
    const [zoom, setZoom] = useState(1);
    const [pan, setPan] = useState({ x: 0, y: 0 });
    const [isPanning, setIsPanning] = useState(false);
    const panStart = useRef({ x: 0, y: 0 });
    const panOffset = useRef({ x: 0, y: 0 });
    const containerRef = useRef<HTMLDivElement>(null);
    const lastCodeRef = useRef('');

    // Auto-analyze with debounce
    useEffect(() => {
        const code = generatedCode || '';
        if (!code.trim() || code === lastCodeRef.current) return;
        const timer = setTimeout(async () => {
            lastCodeRef.current = code;
            setAnalyzing(true);
            try {
                const res = await fetch('http://localhost:8003/analyze-pins', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json', 'X-Desktop-Key': 'desktop_local_bypass_hardcore_ai' },
                    body: JSON.stringify({ code }),
                });
                const data = await res.json();
                if (data.status === 'success') setPinAnalysis(data);
            } catch (e) { /* silent */ }
            setAnalyzing(false);
        }, 600);
        return () => clearTimeout(timer);
    }, [generatedCode]);

    // Zoom with scroll wheel
    const handleWheel = useCallback((e: React.WheelEvent) => {
        e.preventDefault();
        const delta = e.deltaY > 0 ? -0.1 : 0.1;
        setZoom(prev => Math.min(3, Math.max(0.3, prev + delta)));
    }, []);

    // Pan with mouse drag
    const handleMouseDown = useCallback((e: React.MouseEvent) => {
        if (e.button !== 0) return;
        setIsPanning(true);
        panStart.current = { x: e.clientX, y: e.clientY };
        panOffset.current = { ...pan };
    }, [pan]);
    const handleMouseMove = useCallback((e: React.MouseEvent) => {
        if (!isPanning) return;
        setPan({
            x: panOffset.current.x + (e.clientX - panStart.current.x),
            y: panOffset.current.y + (e.clientY - panStart.current.y),
        });
    }, [isPanning]);
    const handleMouseUp = useCallback(() => setIsPanning(false), []);

    const resetView = () => { setZoom(1); setPan({ x: 0, y: 0 }); };

    // Pin lookup
    const activeMap = new Map<number, PinInfo>();
    if (pinAnalysis?.pins) { for (const p of pinAnalysis.pins) activeMap.set(p.gpio, p); }
    const getActive = (gpio: number | string): PinInfo | undefined =>
        typeof gpio === 'number' ? activeMap.get(gpio) : undefined;

    // ─── Render a single pin ─────────────────────────────────────────
    const renderPin = (pin: { gpio: number | string; label: string }, index: number, side: 'left' | 'right') => {
        const active = getActive(pin.gpio);
        const isHover = hoveredGpio === pin.gpio;
        const color = active ? MODE_COLORS[active.mode] || '#9ca3af' : '#404040';
        const isSpecial = typeof pin.gpio === 'string';
        const specialColor = pin.label === 'GND' ? '#ef4444' : pin.label === '3V3' ? '#f59e0b' : pin.label === 'VIN' ? '#ef4444' : '#6b7280';
        const y = BOARD_Y + 25 + index * PIN_GAP;
        const pinColor = active ? color : isSpecial ? specialColor : '#555';

        if (side === 'left') {
            const pinTipX = BOARD_X - PIN_LEN;
            return (
                <g key={`l-${index}`} onMouseEnter={() => setHoveredGpio(pin.gpio)} onMouseLeave={() => setHoveredGpio(null)} style={{ cursor: active ? 'pointer' : 'default' }}>
                    <line x1={BOARD_X} y1={y} x2={pinTipX} y2={y} stroke={pinColor} strokeWidth={active ? 3 : 2} strokeLinecap="round" />
                    <circle cx={pinTipX} cy={y} r={active ? 5 : 3.5} fill={pinColor} stroke={isHover ? '#fff' : 'none'} strokeWidth={1.5} />
                    <text x={pinTipX - 8} y={y + 1} textAnchor="end" dominantBaseline="middle" fill={active ? '#e5e5e5' : '#777'} fontSize="10" fontFamily="monospace" fontWeight={active ? 700 : 400}>{pin.label}</text>
                    {active && (
                        <>
                            <line x1={pinTipX} y1={y} x2={60} y2={y} stroke={color} strokeWidth={2} strokeDasharray={isHover ? 'none' : '4 3'} opacity={isHover ? 1 : 0.7} />
                            <rect x={0} y={y - 14} width={60} height={28} rx={6} fill={color + '15'} stroke={color} strokeWidth={1} />
                            <text x={30} y={y - 3} textAnchor="middle" dominantBaseline="middle" fill={color} fontSize="11">{COMPONENT_ICONS[active.component] || '🔌'}</text>
                            <text x={30} y={y + 9} textAnchor="middle" dominantBaseline="middle" fill={color} fontSize="6" fontWeight={700} fontFamily="monospace">{(MODE_LABELS[active.mode] || active.mode).substring(0, 8)}</text>
                        </>
                    )}
                    {isHover && active && (
                        <g>
                            <rect x={pinTipX - 190} y={y - 35} width={175} height={56} rx={8} fill="#0f0f17f0" stroke={color} strokeWidth={1.5} />
                            <text x={pinTipX - 185} y={y - 18} fill="#fff" fontSize="11" fontWeight={700}>GPIO{active.gpio} → {active.component}</text>
                            <text x={pinTipX - 185} y={y - 2} fill={color} fontSize="9" fontWeight={600}>{MODE_LABELS[active.mode] || active.mode}</text>
                            <text x={pinTipX - 185} y={y + 13} fill="#888" fontSize="8">{active.functions_used.map(f => f + '()').join(', ')}</text>
                        </g>
                    )}
                </g>
            );
        } else {
            const pinTipX = BOARD_X + BOARD_W + PIN_LEN;
            return (
                <g key={`r-${index}`} onMouseEnter={() => setHoveredGpio(pin.gpio)} onMouseLeave={() => setHoveredGpio(null)} style={{ cursor: active ? 'pointer' : 'default' }}>
                    <line x1={BOARD_X + BOARD_W} y1={y} x2={pinTipX} y2={y} stroke={pinColor} strokeWidth={active ? 3 : 2} strokeLinecap="round" />
                    <circle cx={pinTipX} cy={y} r={active ? 5 : 3.5} fill={pinColor} stroke={isHover ? '#fff' : 'none'} strokeWidth={1.5} />
                    <text x={pinTipX + 8} y={y + 1} textAnchor="start" dominantBaseline="middle" fill={active ? '#e5e5e5' : '#777'} fontSize="10" fontFamily="monospace" fontWeight={active ? 700 : 400}>{pin.label}</text>
                    {active && (
                        <>
                            <line x1={pinTipX} y1={y} x2={SVG_W - 60} y2={y} stroke={color} strokeWidth={2} strokeDasharray={isHover ? 'none' : '4 3'} opacity={isHover ? 1 : 0.7} />
                            <rect x={SVG_W - 60} y={y - 14} width={60} height={28} rx={6} fill={color + '15'} stroke={color} strokeWidth={1} />
                            <text x={SVG_W - 30} y={y - 3} textAnchor="middle" dominantBaseline="middle" fill={color} fontSize="11">{COMPONENT_ICONS[active.component] || '🔌'}</text>
                            <text x={SVG_W - 30} y={y + 9} textAnchor="middle" dominantBaseline="middle" fill={color} fontSize="6" fontWeight={700} fontFamily="monospace">{(MODE_LABELS[active.mode] || active.mode).substring(0, 8)}</text>
                        </>
                    )}
                    {isHover && active && (
                        <g>
                            <rect x={pinTipX + 15} y={y - 35} width={175} height={56} rx={8} fill="#0f0f17f0" stroke={color} strokeWidth={1.5} />
                            <text x={pinTipX + 20} y={y - 18} fill="#fff" fontSize="11" fontWeight={700}>GPIO{active.gpio} → {active.component}</text>
                            <text x={pinTipX + 20} y={y - 2} fill={color} fontSize="9" fontWeight={600}>{MODE_LABELS[active.mode] || active.mode}</text>
                            <text x={pinTipX + 20} y={y + 13} fill="#888" fontSize="8">{active.functions_used.map(f => f + '()').join(', ')}</text>
                        </g>
                    )}
                </g>
            );
        }
    };

    const activeCount = activeMap.size;

    return (
        <div className="h-full flex flex-col overflow-hidden bg-[#08080c]">
            {/* Top bar with zoom controls */}
            <div className="flex items-center justify-between px-4 py-2 border-b border-white/5 flex-shrink-0 bg-black/40">
                <div className="flex items-center gap-2">
                    <Cpu className="w-4 h-4 text-violet-400" />
                    <span className="text-xs font-bold text-neutral-300">ESP32-WROOM-32</span>
                    {analyzing && <Loader2 className="w-3 h-3 text-violet-400 animate-spin" />}
                    {activeCount > 0 && (
                        <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 font-mono border border-emerald-500/20">
                            {activeCount} active
                        </span>
                    )}
                </div>
                <div className="flex items-center gap-1">
                    <button onClick={() => setZoom(z => Math.max(0.3, z - 0.15))} className="p-1.5 rounded-lg hover:bg-white/5 text-neutral-500 hover:text-neutral-300 transition-colors" title="Zoom Out"><ZoomOut className="w-3.5 h-3.5" /></button>
                    <span className="text-[10px] font-mono text-neutral-600 w-10 text-center">{Math.round(zoom * 100)}%</span>
                    <button onClick={() => setZoom(z => Math.min(3, z + 0.15))} className="p-1.5 rounded-lg hover:bg-white/5 text-neutral-500 hover:text-neutral-300 transition-colors" title="Zoom In"><ZoomIn className="w-3.5 h-3.5" /></button>
                    <button onClick={resetView} className="p-1.5 rounded-lg hover:bg-white/5 text-neutral-500 hover:text-neutral-300 transition-colors ml-1" title="Reset View"><Maximize2 className="w-3.5 h-3.5" /></button>
                </div>
            </div>

            {/* Warnings */}
            {pinAnalysis?.warnings && pinAnalysis.warnings.length > 0 && (
                <div className="flex items-center gap-2 px-4 py-1.5 bg-amber-500/5 border-b border-amber-500/10 flex-shrink-0 overflow-x-auto">
                    <AlertTriangle className="w-3 h-3 text-amber-400 flex-shrink-0" />
                    <div className="flex gap-3">
                        {pinAnalysis.warnings.map((w, i) => (
                            <span key={i} className="text-[10px] text-amber-400 whitespace-nowrap">{w}</span>
                        ))}
                    </div>
                </div>
            )}

            {/* Zoomable Pannable SVG */}
            <div
                ref={containerRef}
                className="flex-1 overflow-hidden cursor-grab active:cursor-grabbing"
                onWheel={handleWheel}
                onMouseDown={handleMouseDown}
                onMouseMove={handleMouseMove}
                onMouseUp={handleMouseUp}
                onMouseLeave={handleMouseUp}
            >
                <div style={{
                    transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
                    transformOrigin: 'center center',
                    transition: isPanning ? 'none' : 'transform 0.15s ease-out',
                    width: '100%',
                    height: '100%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                }}>
                    <svg viewBox={`0 0 ${SVG_W} ${SVG_H}`} className="w-full h-full" style={{ maxWidth: 900, maxHeight: SVG_H }}>
                        {/* Grid background */}
                        <defs>
                            <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
                                <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#111" strokeWidth="0.5" />
                            </pattern>
                            <linearGradient id="boardGrad" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="0%" stopColor="#1a1a3e" />
                                <stop offset="100%" stopColor="#0a0a1e" />
                            </linearGradient>
                            <filter id="glow">
                                <feGaussianBlur stdDeviation="3" result="blur" />
                                <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
                            </filter>
                        </defs>
                        <rect width={SVG_W} height={SVG_H} fill="url(#grid)" />

                        {/* Board body with gradient */}
                        <rect x={BOARD_X - 2} y={BOARD_Y - 2} width={BOARD_W + 4} height={BOARD_H + 4} rx={10} fill="#222" opacity={0.5} />
                        <rect x={BOARD_X} y={BOARD_Y} width={BOARD_W} height={BOARD_H} rx={8} fill="url(#boardGrad)" stroke="#333" strokeWidth={2} />
                        <rect x={BOARD_X + 3} y={BOARD_Y + 3} width={BOARD_W - 6} height={BOARD_H - 6} rx={6} fill="none" stroke="#252540" strokeWidth={1} />

                        {/* Antenna */}
                        <rect x={BOARD_X + 20} y={BOARD_Y + 5} width={BOARD_W - 40} height={40} rx={4} fill="#15152a" stroke="#333" strokeWidth={0.5} />
                        <line x1={BOARD_X + 40} y1={BOARD_Y + 15} x2={BOARD_X + BOARD_W - 40} y2={BOARD_Y + 15} stroke="#333" strokeWidth={0.3} />
                        <line x1={BOARD_X + 40} y1={BOARD_Y + 25} x2={BOARD_X + BOARD_W - 40} y2={BOARD_Y + 25} stroke="#333" strokeWidth={0.3} />
                        <line x1={BOARD_X + 40} y1={BOARD_Y + 35} x2={BOARD_X + BOARD_W - 40} y2={BOARD_Y + 35} stroke="#333" strokeWidth={0.3} />
                        <text x={BOARD_X + BOARD_W / 2} y={BOARD_Y + 22} textAnchor="middle" fill="#444" fontSize="7" fontFamily="monospace">ANT</text>

                        {/* ESP32 chip */}
                        <rect x={BOARD_X + 25} y={BOARD_Y + 55} width={BOARD_W - 50} height={80} rx={4} fill="#0a0a15" stroke="#444" strokeWidth={1.5} />
                        <circle cx={BOARD_X + 35} cy={BOARD_Y + 65} r={3} fill="none" stroke="#555" strokeWidth={0.5} />
                        <text x={BOARD_X + BOARD_W / 2} y={BOARD_Y + 88} textAnchor="middle" fill="#555" fontSize="7" fontFamily="monospace" fontWeight={700}>ESP-WROOM-32</text>
                        <text x={BOARD_X + BOARD_W / 2} y={BOARD_Y + 100} textAnchor="middle" fill="#3a3a5a" fontSize="6" fontFamily="monospace">Dual-Core 240MHz</text>
                        <text x={BOARD_X + BOARD_W / 2} y={BOARD_Y + 110} textAnchor="middle" fill="#3a3a5a" fontSize="5" fontFamily="monospace">4MB Flash | WiFi+BT</text>

                        {/* Reset + Boot buttons */}
                        <rect x={BOARD_X + 15} y={BOARD_Y + BOARD_H - 80} width={24} height={12} rx={2} fill="#222" stroke="#444" strokeWidth={0.5} />
                        <text x={BOARD_X + 27} y={BOARD_Y + BOARD_H - 71} textAnchor="middle" fill="#444" fontSize="4" fontFamily="monospace">RST</text>
                        <rect x={BOARD_X + BOARD_W - 39} y={BOARD_Y + BOARD_H - 80} width={24} height={12} rx={2} fill="#222" stroke="#444" strokeWidth={0.5} />
                        <text x={BOARD_X + BOARD_W - 27} y={BOARD_Y + BOARD_H - 71} textAnchor="middle" fill="#444" fontSize="4" fontFamily="monospace">BOOT</text>

                        {/* USB connector */}
                        <rect x={BOARD_X + 45} y={BOARD_Y + BOARD_H - 30} width={BOARD_W - 90} height={28} rx={3} fill="#1a1a1a" stroke="#555" strokeWidth={1.5} />
                        <rect x={BOARD_X + 55} y={BOARD_Y + BOARD_H - 25} width={BOARD_W - 110} height={18} rx={2} fill="#111" stroke="#333" strokeWidth={0.5} />
                        <text x={BOARD_X + BOARD_W / 2} y={BOARD_Y + BOARD_H - 13} textAnchor="middle" fill="#444" fontSize="6" fontFamily="monospace">MICRO-USB</text>

                        {/* Board label */}
                        <text x={BOARD_X + BOARD_W / 2} y={BOARD_Y + BOARD_H - 50} textAnchor="middle" fill="#7c3aed" fontSize="11" fontFamily="monospace" fontWeight={900} filter="url(#glow)">ESP32 DevKit V1</text>

                        {/* Power indicators */}
                        <circle cx={BOARD_X + 12} cy={BOARD_Y + 12} r={2.5} fill="#ef4444" />
                        <text x={BOARD_X + 20} y={BOARD_Y + 13} fill="#555" fontSize="4" fontFamily="monospace">PWR</text>
                        <circle cx={BOARD_X + BOARD_W - 12} cy={BOARD_Y + 12} r={2.5} fill="#22c55e" />
                        <text x={BOARD_X + BOARD_W - 20} y={BOARD_Y + 13} textAnchor="end" fill="#555" fontSize="4" fontFamily="monospace">IO</text>

                        {/* Pin rows */}
                        {LEFT_PINS.map((pin, i) => renderPin(pin, i, 'left'))}
                        {RIGHT_PINS.map((pin, i) => renderPin(pin, i, 'right'))}

                        {/* Glow animations for active pins */}
                        {Array.from(activeMap.entries()).map(([gpio, info]) => {
                            const color = MODE_COLORS[info.mode] || '#9ca3af';
                            const leftIdx = LEFT_PINS.findIndex(p => p.gpio === gpio);
                            const rightIdx = RIGHT_PINS.findIndex(p => p.gpio === gpio);
                            if (leftIdx === -1 && rightIdx === -1) return null;
                            const y = BOARD_Y + 25 + (leftIdx >= 0 ? leftIdx : rightIdx) * PIN_GAP;
                            const cx = leftIdx >= 0 ? (BOARD_X - PIN_LEN) : (BOARD_X + BOARD_W + PIN_LEN);
                            return (
                                <circle key={`glow-${gpio}`} cx={cx} cy={y} r={8} fill="none" stroke={color} strokeWidth={0.5} opacity={0.3}>
                                    <animate attributeName="r" values="6;12;6" dur="2.5s" repeatCount="indefinite" />
                                    <animate attributeName="opacity" values="0.3;0.08;0.3" dur="2.5s" repeatCount="indefinite" />
                                </circle>
                            );
                        })}
                    </svg>
                </div>
            </div>

            {/* Legend footer */}
            <div className="flex items-center justify-center gap-4 px-4 py-2 border-t border-white/5 flex-shrink-0 flex-wrap bg-black/40">
                {[
                    { label: 'OUTPUT', color: '#22c55e' }, { label: 'INPUT', color: '#3b82f6' },
                    { label: 'ANALOG', color: '#a78bfa' }, { label: 'PWM', color: '#f59e0b' },
                    { label: 'I2C', color: '#ec4899' }, { label: 'SPI', color: '#14b8a6' },
                    { label: 'UART', color: '#f43f5e' },
                ].map(l => (
                    <div key={l.label} className="flex items-center gap-1.5">
                        <div className="w-2 h-2 rounded-full" style={{ backgroundColor: l.color }} />
                        <span className="text-[9px] text-neutral-500 font-mono">{l.label}</span>
                    </div>
                ))}
                <span className="text-[9px] text-neutral-700 ml-2 hidden sm:inline">Scroll to zoom · Drag to pan</span>
            </div>
        </div>
    );
};

export default PinDiagramTab;
