import { Plus } from 'lucide-react';
import { useSettings } from './hooks/useSettings';
import { useInvestigation } from './hooks/useInvestigation';
import { SettingsModal } from './components/SettingsModal';
import { ChatInterface } from './components/ChatInterface';


function App() {
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
    <div className="flex h-screen bg-bg-app text-text-primary overflow-hidden">

      {/* Main Content */}
      <main className="flex-1 flex flex-col relative w-full">
        {/* Header */}
        <header className="h-16 flex items-center justify-between px-4 md:px-6">
          <div className="flex items-center gap-3">
            <span className="text-lg font-medium text-text-secondary">Kabinet</span>
          </div>

          <button
            onClick={() => window.location.reload()}
            className="flex items-center gap-2 bg-bg-panel hover:bg-gray-700 text-text-primary px-4 py-2 rounded-full text-sm font-medium transition-colors"
          >
            <Plus className="w-4 h-4" />
            New Chat
          </button>
        </header>

        <div className="flex-1 overflow-hidden relative">
          <ChatInterface
            messages={messages}
            status={status}
            currentThought={currentThought}
            currentHypothesis={currentHypothesis}
            currentQuery={currentQuery}
            onStartInvestigation={start}
            onStop={stop}
          />
        </div>
      </main>

      {/* Modals */}
      <SettingsModal
        isOpen={isOpen}
        onClose={closeSettings}
        config={config}
        onSave={saveConfig}
      />
    </div>
  );
}

export default App;
