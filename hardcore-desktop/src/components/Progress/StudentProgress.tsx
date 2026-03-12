import React from 'react';
import { motion } from 'framer-motion';
import { Trophy, Target, Zap, BookOpen, X } from 'lucide-react';

interface StudentProgressProps {
    isOpen: boolean;
    onClose: () => void;
}

const StudentProgress: React.FC<StudentProgressProps> = ({ isOpen, onClose }) => {
    // Mock data - in real app, this would come from state/context
    const stats = {
        lessonsCompleted: 2,
        totalLessons: 4,
        projectsBuilt: 3,
        skillsLearned: ['Digital I/O', 'PWM', 'Sensors', 'Serial Communication'],
        totalSkills: 12,
        hoursSpent: 5.5
    };

    const achievements = [
        { id: 1, name: 'First Blink', description: 'Completed your first LED blink', unlocked: true },
        { id: 2, name: 'Input Master', description: 'Successfully read button input', unlocked: true },
        { id: 3, name: 'Motor Driver', description: 'Controlled a motor with PWM', unlocked: false },
        { id: 4, name: 'Sensor Expert', description: 'Read data from 3 different sensors', unlocked: false },
    ];

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
            <motion.div
                initial={{ scale: 0.9, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                className="w-full max-w-2xl max-h-[80vh] bg-neutral-900 rounded-lg border border-neutral-800 shadow-2xl overflow-hidden"
            >
                {/* Header */}
                <div className="px-6 py-4 border-b border-neutral-800 bg-neutral-950 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <Trophy className="w-6 h-6 text-yellow-400" />
                        <h2 className="text-lg font-bold text-white">Your Learning Progress</h2>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-1 hover:bg-neutral-800 rounded text-neutral-400 hover:text-white"
                    >
                        <X className="w-5 h-5" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 overflow-y-auto max-h-[calc(80vh-80px)]">
                    {/* Stats Grid */}
                    <div className="grid grid-cols-3 gap-4 mb-6">
                        <div className="p-4 rounded-lg bg-violet-950/30 border border-violet-500/50">
                            <div className="flex items-center gap-2 mb-2">
                                <BookOpen className="w-4 h-4 text-violet-400" />
                                <span className="text-xs text-violet-300 font-medium">Lessons</span>
                            </div>
                            <div className="text-2xl font-bold text-white">{stats.lessonsCompleted}/{stats.totalLessons}</div>
                            <div className="mt-2 h-1.5 bg-neutral-800 rounded-full overflow-hidden">
                                <motion.div
                                    initial={{ width: 0 }}
                                    animate={{ width: `${(stats.lessonsCompleted / stats.totalLessons) * 100}%` }}
                                    className="h-full bg-violet-500"
                                />
                            </div>
                        </div>

                        <div className="p-4 rounded-lg bg-blue-950/30 border border-blue-500/50">
                            <div className="flex items-center gap-2 mb-2">
                                <Zap className="w-4 h-4 text-blue-400" />
                                <span className="text-xs text-blue-300 font-medium">Projects</span>
                            </div>
                            <div className="text-2xl font-bold text-white">{stats.projectsBuilt}</div>
                            <div className="text-xs text-blue-300 mt-1">Built & Flashed</div>
                        </div>

                        <div className="p-4 rounded-lg bg-green-950/30 border border-green-500/50">
                            <div className="flex items-center gap-2 mb-2">
                                <Target className="w-4 h-4 text-green-400" />
                                <span className="text-xs text-green-300 font-medium">Skills</span>
                            </div>
                            <div className="text-2xl font-bold text-white">{stats.skillsLearned.length}/{stats.totalSkills}</div>
                            <div className="mt-2 h-1.5 bg-neutral-800 rounded-full overflow-hidden">
                                <motion.div
                                    initial={{ width: 0 }}
                                    animate={{ width: `${(stats.skillsLearned.length / stats.totalSkills) * 100}%` }}
                                    className="h-full bg-green-500"
                                />
                            </div>
                        </div>
                    </div>

                    {/* Skills Learned */}
                    <div className="mb-6">
                        <h3 className="text-sm font-bold text-white mb-3">Skills Mastered</h3>
                        <div className="flex flex-wrap gap-2">
                            {stats.skillsLearned.map((skill, i) => (
                                <motion.span
                                    key={i}
                                    initial={{ scale: 0 }}
                                    animate={{ scale: 1 }}
                                    transition={{ delay: i * 0.1 }}
                                    className="px-3 py-1.5 rounded-full bg-violet-500/20 text-violet-300 text-xs font-medium border border-violet-500/50"
                                >
                                    ✓ {skill}
                                </motion.span>
                            ))}
                        </div>
                    </div>

                    {/* Achievements */}
                    <div>
                        <h3 className="text-sm font-bold text-white mb-3">Achievements</h3>
                        <div className="space-y-2">
                            {achievements.map((achievement) => (
                                <div
                                    key={achievement.id}
                                    className={`p-3 rounded-lg border ${achievement.unlocked
                                            ? 'bg-yellow-950/20 border-yellow-500/50'
                                            : 'bg-neutral-950 border-neutral-800 opacity-50'
                                        }`}
                                >
                                    <div className="flex items-center gap-3">
                                        <div className={`p-2 rounded-full ${achievement.unlocked ? 'bg-yellow-500/20' : 'bg-neutral-800'
                                            }`}>
                                            <Trophy className={`w-4 h-4 ${achievement.unlocked ? 'text-yellow-400' : 'text-neutral-600'
                                                }`} />
                                        </div>
                                        <div className="flex-1">
                                            <div className="text-sm font-semibold text-white">{achievement.name}</div>
                                            <div className="text-xs text-neutral-400">{achievement.description}</div>
                                        </div>
                                        {achievement.unlocked && (
                                            <span className="text-xs text-yellow-400 font-medium">Unlocked!</span>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
            </motion.div>
        </div>
    );
};

export default StudentProgress;
