import { useState } from 'react';
import { Header } from './components/Header';
import { Dashboard } from './components/Dashboard';
import { TimeRange } from './types';
import { getTimeRanges } from './utils/date';

function App() {
  const [selectedTimeRange, setSelectedTimeRange] = useState<TimeRange>(
    getTimeRanges()[2] // 기본값: 지난 24시간
  );

  return (
    <div className="min-h-screen bg-gray-50">
      <Header 
        selectedTimeRange={selectedTimeRange}
        onTimeRangeChange={setSelectedTimeRange}
      />
      
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Dashboard timeRange={selectedTimeRange} />
      </main>
    </div>
  );
}

export default App;