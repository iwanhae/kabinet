import React from 'react';
import { Loader2 } from 'lucide-react';

interface LoadingSpinnerProps {
  message?: string;
}

export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({ 
  message = "데이터를 불러오는 중..." 
}) => {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <Loader2 className="w-8 h-8 text-twitter-blue animate-spin mb-4" />
      <p className="text-gray-600 text-sm">{message}</p>
    </div>
  );
};