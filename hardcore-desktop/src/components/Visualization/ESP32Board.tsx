import React from 'react';
import { motion } from 'framer-motion';

interface ESP32BoardProps {
    activePins?: number[];
    pinStates?: { [pin: number]: 'HIGH' | 'LOW' };
    onPinClick?: (pin: number) => void;
}

const ESP32Board: React.FC<ESP32BoardProps> = ({ activePins = [], pinStates = {}, onPinClick }) => {
    // ESP32 DevKit GPIO pins
    const leftPins = [
        { num: 36, label: 'VP' }, { num: 39, label: 'VN' }, { num: 34, label: '34' },
        { num: 35, label: '35' }, { num: 32, label: '32' }, { num: 33, label: '33' },
        { num: 25, label: '25' }, { num: 26, label: '26' }, { num: 27, label: '27' },
        { num: 14, label: '14' }, { num: 12, label: '12' }, { num: 13, label: '13' },
        { num: -1, label: 'GND' }, { num: -2, label: 'VIN' },
    ];

    const rightPins = [
        { num: -3, label: '3V3' }, { num: 23, label: '23' }, { num: 22, label: '22' },
        { num: 1, label: 'TX' }, { num: 3, label: 'RX' }, { num: 21, label: '21' },
        { num: 19, label: '19' }, { num: 18, label: '18' }, { num: 5, label: '5' },
        { num: 17, label: '17' }, { num: 16, label: '16' }, { num: 4, label: '4' },
        { num: 2, label: '2' }, { num: 15, label: '15' },
    ];

    const isPinActive = (pinNum: number) => activePins.includes(pinNum);
    const getPinState = (pinNum: number) => pinStates[pinNum];

    const renderPin = (pin: typeof leftPins[0], side: 'left' | 'right', index: number) => {
        const isActive = isPinActive(pin.num);
        const state = getPinState(pin.num);
        const isHigh = state === 'HIGH';
        const isLow = state === 'LOW';

        return (
            <g key={`${side}-${pin.num}`}>
                {/* Pin Metal Texture */}
                <motion.rect
                    x={side === 'left' ? 5 : 315}
                    y={50 + index * 18}
                    width="15"
                    height="10"
                    rx="1"
                    fill={isActive ? (isHigh ? '#22c55e' : isLow ? '#3b82f6' : '#7c3aed') : '#262626'}
                    className="cursor-pointer transition-colors duration-300"
                    stroke={isActive ? '#fff' : '#444'}
                    strokeWidth={isActive ? 1 : 0.5}
                    onClick={() => pin.num > 0 && onPinClick?.(pin.num)}
                    whileHover={{ scale: 1.2, x: side === 'left' ? -2 : 2 }}
                />

                {/* Neon Glow on Active Pins */}
                {isActive && (
                    <motion.rect
                        x={side === 'left' ? 2 : 318}
                        y={48 + index * 18}
                        width="21"
                        height="14"
                        rx="3"
                        fill={isHigh ? '#22c55e' : isLow ? '#3b82f6' : '#7c3aed'}
                        opacity="0.15"
                        animate={{ opacity: [0.1, 0.3, 0.1] }}
                        transition={{ duration: 2, repeat: Infinity }}
                        pointerEvents="none"
                    />
                )}

                {/* Label */}
                <text
                    x={side === 'left' ? 25 : 310}
                    y={58 + index * 18}
                    fontSize="9"
                    fontWeight="900"
                    fill={isActive ? '#fff' : '#666'}
                    textAnchor={side === 'left' ? 'start' : 'end'}
                    className="font-mono tracking-tighter"
                >
                    {pin.label}
                </text>
            </g>
        );
    };

    return (
        <div className="w-full h-full flex items-center justify-center p-6 select-none drop-shadow-[0_20px_50px_rgba(0,0,0,0.5)]">
            <svg viewBox="0 0 335 360" className="w-full h-full drop-shadow-2xl">
                <defs>
                    <linearGradient id="pcbGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" stopColor="#0a0a1a" />
                        <stop offset="100%" stopColor="#050505" />
                    </linearGradient>
                    <filter id="glow">
                        <feGaussianBlur stdDeviation="2.5" result="coloredBlur" />
                        <feMerge>
                            <feMergeNode in="coloredBlur" />
                            <feMergeNode in="SourceGraphic" />
                        </feMerge>
                    </filter>
                </defs>

                {/* PCB MAIN BODY */}
                <rect x="25" y="20" width="285" height="320" rx="12" fill="url(#pcbGrad)" stroke="#1a1a1a" strokeWidth="2" />
                <rect x="28" y="23" width="279" height="314" rx="10" fill="none" stroke="rgba(255,255,255,0.03)" strokeWidth="1" />

                {/* ANTENNA TRACE AREA */}
                <rect x="110" y="25" width="115" height="40" rx="4" fill="#111" />
                <path d="M120 35 H215 M125 45 H180" stroke="#333" strokeWidth="2" strokeLinecap="round" />

                {/* ESP32 METAL CAN (The Chip) */}
                <rect x="90" y="110" width="155" height="110" rx="6" fill="#1a1a1a" stroke="#222" strokeWidth="1" />
                <rect x="95" y="115" width="145" height="100" rx="4" fill="#0c0c0c" />

                {/* Chip Laser Engraving */}
                <text x="167" y="155" fontSize="14" fill="#fff" textAnchor="middle" className="font-black tracking-widest opacity-80">ESP32</text>
                <text x="167" y="175" fontSize="8" fill="#555" textAnchor="middle" className="font-bold tracking-widest uppercase">WROOM-32D</text>
                <text x="167" y="195" fontSize="6" fill="#333" textAnchor="middle" className="font-mono">FCC ID: 2AC7Z-ESP32WROOM32</text>

                {/* Micro-Controller Components */}
                <rect x="110" y="230" width="12" height="12" rx="1" fill="#111" />
                <rect x="130" y="230" width="12" height="12" rx="1" fill="#111" />
                <rect x="150" y="230" width="35" height="12" rx="1" fill="#222" />

                {/* USB-C PORT (Modernized) */}
                <rect x="140" y="325" width="55" height="15" rx="3" fill="#000" stroke="#333" />
                <rect x="145" y="333" width="45" height="2" rx="1" fill="#222" />

                {/* Header Pins Row */}
                <rect x="25" y="45" width="12" height="270" rx="2" fill="#080808" />
                <rect x="298" y="45" width="12" height="270" rx="2" fill="#080808" />

                {/* Left side pins */}
                {leftPins.map((pin, i) => renderPin(pin, 'left', i))}

                {/* Right side pins */}
                {rightPins.map((pin, i) => renderPin(pin, 'right', i))}

                {/* Board Silkscreen */}
                <text x="167" y="85" fontSize="10" fill="rgba(124, 58, 237, 0.5)" textAnchor="middle" className="font-black uppercase tracking-[0.5em]">HardcoreAI</text>
                <circle cx="280" cy="35" r="3" fill="#1a1a1a" stroke="#333" />
                <circle cx="55" cy="35" r="3" fill="#1a1a1a" stroke="#333" />
            </svg>
        </div>
    );
};

export default ESP32Board;
