import React from 'react';
import type { InvestigationStatus, AgentPlan } from '../../types/agent';
import {
    Card,
    CardContent,
    Box,
    Typography,
    Chip,
    LinearProgress,
    alpha
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import SearchIcon from '@mui/icons-material/Search';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import TipsAndUpdatesIcon from '@mui/icons-material/TipsAndUpdates';
import TerminalIcon from '@mui/icons-material/Terminal';
import AccessTimeIcon from '@mui/icons-material/AccessTime';

interface AgentStepProps {
    status: InvestigationStatus;
    thought: string;
    hypothesis: string;
    query?: AgentPlan['query'];
}

export const AgentStep: React.FC<AgentStepProps> = ({ status, thought, hypothesis, query }) => {
    if (status === 'idle' || status === 'error') return null;

    const isComplete = status === 'complete';

    const getStatusIcon = () => {
        switch (status) {
            case 'planning': return <AutoAwesomeIcon />;
            case 'querying': return <SearchIcon />;
            case 'analyzing': return <AutorenewIcon sx={{ animation: 'spin 2s linear infinite' }} />;
            case 'complete': return <CheckCircleOutlineIcon />;
            default: return <AutoAwesomeIcon />;
        }
    };

    const getStatusColor = () => {
        switch (status) {
            case 'complete': return 'success';
            case 'analyzing': return 'warning';
            default: return 'primary';
        }
    };

    return (
        <Card
            variant="outlined"
            sx={{
                my: 3,
                position: 'relative',
                overflow: 'hidden',
                boxShadow: 3
            }}
        >
            {!isComplete && <LinearProgress sx={{ position: 'absolute', top: 0, left: 0, right: 0 }} />}

            <CardContent>
                {/* Status Header */}
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
                    <Chip
                        icon={getStatusIcon()}
                        label={isComplete ? 'Investigation Complete' : `${status.charAt(0).toUpperCase() + status.slice(1)}...`}
                        color={getStatusColor()}
                        variant="outlined"
                        sx={{ textTransform: 'capitalize' }}
                    />
                </Box>

                {/* Thought Process */}
                {thought && !isComplete && (
                    <Box sx={{ pl: 2, borderLeft: 2, borderColor: 'divider', mb: 2 }}>
                        <Typography variant="body2" color="text.secondary" fontStyle="italic">
                            {thought}
                        </Typography>
                    </Box>
                )}

                {/* Current Hypothesis */}
                {hypothesis && !isComplete && (
                    <Box sx={{
                        pl: 2,
                        borderLeft: '3px solid',
                        borderColor: 'info.main',
                        bgcolor: (theme) => alpha(theme.palette.info.main, 0.05),
                        p: 1.5,
                        borderRadius: 1,
                        mb: 2
                    }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                            <TipsAndUpdatesIcon fontSize="small" color="info" />
                            <Typography variant="caption" sx={{ fontWeight: 700, letterSpacing: 0.5, color: 'info.main' }}>
                                WORKING HYPOTHESIS
                            </Typography>
                        </Box>
                        <Typography variant="body2">
                            {hypothesis}
                        </Typography>
                    </Box>
                )}

                {/* Active Query */}
                {query && status === 'querying' && (
                    <Box sx={{ mt: 2 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                            <TerminalIcon color="action" fontSize="small" />
                            <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary' }}>
                                EXECUTING SQL QUERY
                            </Typography>
                        </Box>
                        <Box
                            sx={{
                                p: 2,
                                bgcolor: 'grey.900',
                                color: 'grey.100',
                                borderRadius: 2,
                                fontFamily: 'monospace',
                                fontSize: '0.875rem',
                                whiteSpace: 'pre-wrap',
                                wordBreak: 'break-all',
                                border: '1px solid',
                                borderColor: 'grey.800'
                            }}
                        >
                            {query.sql}
                        </Box>
                        <Box sx={{ display: 'flex', gap: 2, mt: 1, color: 'text.secondary' }}>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                <AccessTimeIcon fontSize="small" />
                                <Typography variant="caption">
                                    Start: {new Date(query.start).toLocaleTimeString()}
                                </Typography>
                            </Box>
                        </Box>
                    </Box>
                )}
            </CardContent>
            <style>
                {`
                    @keyframes spin {
                        from { transform: rotate(0deg); }
                        to { transform: rotate(360deg); }
                    }
                `}
            </style>
        </Card>
    );
};
