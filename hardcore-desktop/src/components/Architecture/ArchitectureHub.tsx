import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Layout, ShieldAlert, Landmark, ShoppingCart, AlertTriangle, Code } from 'lucide-react';
import { useAppState } from '../../context/AppStateContext';
import ArchitectureOverview from './ArchitectureOverview';
import SecurityArchitecture from './SecurityArchitecture';
import ComplianceReport from './ComplianceReport';
import BOMGenerator from './BOMGenerator';
import RiskAnalysis from './RiskAnalysis';
import MonacoEditor from '../Editor/MonacoEditor';

const ArchitectureHub: React.FC = () => {
    const { state } = useAppState();
    const { activeTab } = state;

    const renderContent = () => {
        switch (activeTab) {
            case 'architecture':
                return <ArchitectureOverview />;
            case 'security':
                return <SecurityArchitecture />;
            case 'compliance':
                return <ComplianceReport />;
            case 'bom':
                return <BOMGenerator />;
            case 'risk':
                return <RiskAnalysis />;
            default:
                return <ArchitectureOverview />;
        }
    };

    return (
        <div className="h-full w-full overflow-y-auto">
            {renderContent()}
        </div>
    );
};

export default ArchitectureHub;
