
import { useSettings } from '../hooks/agent/useSettings';
import { useInvestigation } from '../hooks/agent/useInvestigation';
import { SettingsModal } from '../components/Agent/SettingsModal';
import { ChatInterface } from '../components/Agent/ChatInterface';
import { Box, IconButton, Tooltip, Card } from '@mui/material';
import SettingsIcon from '@mui/icons-material/Settings';
import AddIcon from '@mui/icons-material/Add';



const AgentPage = () => {
    const { config, saveConfig, isOpen, openSettings, closeSettings } = useSettings();
    const {
        messages,
        status,
        currentThought,
        currentHypothesis,
        currentQuery,
        start,
        stop
    } = useInvestigation(config);

    return (
        <Box sx={{ width: "100%", height: "calc(100vh - 100px)", display: "flex", flexDirection: "column" }}>
            {/* Chat Container */}
            <Card sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', mb: 2, position: 'relative' }}>
                <Box sx={{ position: 'absolute', top: 12, right: 12, zIndex: 10, bgcolor: 'background.paper', borderRadius: 2, boxShadow: 1 }}>
                    <Tooltip title="New Chat">
                        <IconButton onClick={() => window.location.reload()} size="small">
                            <AddIcon />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Settings">
                        <IconButton onClick={openSettings} size="small">
                            <SettingsIcon />
                        </IconButton>
                    </Tooltip>
                </Box>
                <ChatInterface
                    messages={messages}
                    status={status}
                    currentThought={currentThought}
                    currentHypothesis={currentHypothesis}
                    currentQuery={currentQuery}
                    onStartInvestigation={start}
                    onStop={stop}
                />
            </Card>

            {/* Settings Modal */}
            <SettingsModal
                isOpen={isOpen}
                onClose={closeSettings}
                config={config}
                onSave={saveConfig}
            />
        </Box>
    );
};

export default AgentPage;
